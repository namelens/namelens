package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/namelens/namelens/internal/config"
	"github.com/namelens/namelens/internal/core"
	"github.com/namelens/namelens/internal/core/engine"
	"github.com/namelens/namelens/internal/output"
)

var batchCmd = &cobra.Command{
	Use:   "batch <file>",
	Short: "Check multiple names from file",
	Long:  "Read names from file (one per line) and check availability",
	Args:  cobra.ExactArgs(1),
	RunE:  runBatch,
}

func init() {
	rootCmd.AddCommand(batchCmd)

	batchCmd.Flags().String("profile", "minimal", "Profile to use")
	batchCmd.Flags().String("output", "table", "Output format: table, json, markdown")
	batchCmd.Flags().Bool("available-only", false, "Only show names fully available across all checks")
	batchCmd.Flags().Int("concurrency", 3, "Concurrent checks")
}

func runBatch(cmd *cobra.Command, args []string) error {
	profileName, err := cmd.Flags().GetString("profile")
	if err != nil {
		return err
	}
	if strings.TrimSpace(profileName) == "" {
		return errors.New("profile is required")
	}

	formatValue, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	format, err := output.ParseFormat(formatValue)
	if err != nil {
		return err
	}

	availableOnly, err := cmd.Flags().GetBool("available-only")
	if err != nil {
		return err
	}

	concurrency, err := cmd.Flags().GetInt("concurrency")
	if err != nil {
		return err
	}
	if concurrency < 1 {
		return errors.New("concurrency must be at least 1")
	}

	names, err := readBatchNames(args[0])
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return errors.New("no names found in batch file")
	}

	ctx := cmd.Context()
	startedAt := time.Now()

	store, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close() // nolint:errcheck // best-effort cleanup; errors logged internally

	cfg := config.GetConfig()
	if cfg == nil {
		return errors.New("config not loaded")
	}

	profile, err := resolveProfile(ctx, store, profileName, nil, nil, nil)
	if err != nil {
		return err
	}
	if len(profile.TLDs) == 0 && len(profile.Registries) == 0 && len(profile.Handles) == 0 {
		return errors.New("at least one check target is required")
	}

	orchestrator := buildOrchestrator(cfg, store, true)

	results, err := runBatchChecks(ctx, orchestrator, profile, names, concurrency)
	if err != nil {
		return err
	}

	results = filterBatchResults(results, availableOnly)

	rendered, err := output.FormatBatchList(format, results)
	if err != nil {
		return err
	}
	if strings.TrimSpace(rendered) != "" {
		fmt.Println(rendered)
	}

	logThroughput(totalChecks(results), startedAt)
	return nil
}

type batchJob struct {
	index int
	name  string
}

func runBatchChecks(ctx context.Context, orchestrator *engine.Orchestrator, profile core.Profile, names []string, concurrency int) ([]*core.BatchResult, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]*core.BatchResult, len(names))
	jobs := make(chan batchJob)

	var (
		wg       sync.WaitGroup
		errOnce  sync.Once
		firstErr error
	)

	setErr := func(err error) {
		if err == nil {
			return
		}
		errOnce.Do(func() {
			firstErr = err
			cancel()
		})
	}

	worker := func() {
		defer wg.Done()
		for job := range jobs {
			if ctx.Err() != nil {
				return
			}
			checks, err := orchestrator.Check(ctx, job.name, profile)
			if err != nil {
				setErr(err)
				return
			}
			results[job.index] = summarizeResults(job.name, checks, nil, nil, nil, nil, nil, nil)
		}
	}

	if concurrency > len(names) {
		concurrency = len(names)
	}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go worker()
	}

sendLoop:
	for i, name := range names {
		select {
		case <-ctx.Done():
			break sendLoop
		case jobs <- batchJob{index: i, name: name}:
		}
	}
	close(jobs)
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}

func readBatchNames(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close() // nolint:errcheck // best-effort cleanup on read-only file

	names := make([]string, 0)
	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		name := strings.ToLower(raw)
		if err := validateName(name); err != nil {
			return nil, fmt.Errorf("invalid name on line %d: %w", line, err)
		}
		names = append(names, name)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return names, nil
}

func filterBatchResults(results []*core.BatchResult, availableOnly bool) []*core.BatchResult {
	if !availableOnly {
		return results
	}

	filtered := make([]*core.BatchResult, 0, len(results))
	for _, result := range results {
		if result == nil {
			continue
		}
		if result.Total > 0 && result.Score == result.Total {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

func totalChecks(results []*core.BatchResult) int {
	total := 0
	for _, result := range results {
		if result == nil {
			continue
		}
		total += result.Total
	}
	return total
}
