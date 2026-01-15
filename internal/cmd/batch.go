package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
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
	batchCmd.Flags().String("output-format", "table", "Output format: table, json, markdown")
	batchCmd.Flags().String("out", "", "Write output to a file (default stdout)")
	batchCmd.Flags().String("out-dir", "", "Write per-name outputs to a directory")
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

	formatValue, err := cmd.Flags().GetString("output-format")
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

	names, err := readNamesFile(args[0])
	if err != nil {
		return err
	}
	// readNamesFile already validates non-empty

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

	outPath, outDir, err := resolveOutputTargets(cmd)
	if err != nil {
		return err
	}

	ext := outputExtension(format)
	rendered, err := output.FormatBatchList(format, results)
	if err != nil {
		return err
	}

	if outDir != "" {
		outDir, err := ensureOutDir(outDir)
		if err != nil {
			return err
		}

		indexPath := filepath.Join(outDir, fmt.Sprintf("batch.index.%s", ext))
		indexSink, err := openSink(indexPath)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprint(indexSink.writer, rendered); err != nil {
			_ = indexSink.close()
			return err
		}
		if err := indexSink.close(); err != nil {
			return err
		}

		formatter := output.NewFormatter(format)
		for _, result := range results {
			if result == nil {
				continue
			}
			name := sanitizeFilename(result.Name)
			path := filepath.Join(outDir, fmt.Sprintf("%s.batch.%s", name, ext))
			sink, err := openSink(path)
			if err != nil {
				return err
			}

			var content string
			if format == output.FormatJSON {
				payload, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					_ = sink.close()
					return err
				}
				content = string(payload)
			} else {
				content, err = formatter.FormatBatch(result)
				if err != nil {
					_ = sink.close()
					return err
				}
			}

			if _, err := fmt.Fprint(sink.writer, content); err != nil {
				_ = sink.close()
				return err
			}
			if err := sink.close(); err != nil {
				return err
			}
		}
	} else {
		sink, err := openSink(outPath)
		if err != nil {
			return err
		}
		if strings.TrimSpace(rendered) != "" {
			if _, err := fmt.Fprint(sink.writer, rendered); err != nil {
				_ = sink.close()
				return err
			}
		}
		if err := sink.close(); err != nil {
			return err
		}
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
