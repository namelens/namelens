package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/namelens/namelens/internal/daemon"
	errwrap "github.com/namelens/namelens/internal/errors"
)

var (
	stopForce       bool
	stopGracePeriod time.Duration
	cleanupForce    bool
)

var serveStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a running server",
	Long: `Stop a running NameLens server.

By default, sends SIGTERM for graceful shutdown. If the server doesn't stop
within the grace period (default 10s), SIGKILL is sent.

Examples:
  namelens serve stop                    # Stop server on default port (8080)
  namelens serve stop --port 9000        # Stop server on port 9000
  namelens serve stop --force            # Force kill immediately`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if stopForce {
			err = daemon.ForceStop(serverPort)
		} else {
			err = daemon.Stop(serverPort, stopGracePeriod)
		}

		if err != nil {
			if errors.Is(err, daemon.ErrServerNotRunning) {
				fmt.Printf("No server running on port %d\n", serverPort)
				return nil
			}
			if errors.Is(err, daemon.ErrServerNotManaged) {
				fmt.Printf("Refusing to stop non-managed process on port %d\n", serverPort)
				fmt.Println("  Use 'namelens serve cleanup --port <port>' if you want to terminate it")
				return nil
			}
			return errwrap.WrapInternal(cmd.Context(), err, "failed to stop server")
		}

		fmt.Printf("Server on port %d stopped\n", serverPort)
		return nil
	},
}

var serveStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check server status",
	Long: `Check the status of NameLens servers.

Shows whether a server is running on the specified port, its PID,
uptime, and process information.

Examples:
  namelens serve status                  # Check server on default port (8080)
  namelens serve status --port 9000      # Check server on port 9000
  namelens serve status --all            # List all known server instances`,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")

		if all {
			return showAllServers()
		}

		status, err := daemon.Status(serverPort)
		if err != nil {
			return errwrap.WrapInternal(cmd.Context(), err, "failed to get server status")
		}

		if status.Stale {
			fmt.Printf("Port %d: Stale PID file (PID %d no longer running)\n", serverPort, status.PID)
			fmt.Println("  Run 'namelens serve cleanup' to remove stale PID file")
			return nil
		}

		if !status.Running {
			fmt.Printf("No server running on port %d\n", serverPort)
			return nil
		}

		fmt.Printf("Server running on port %d\n", serverPort)
		fmt.Printf("  PID:     %d\n", status.PID)
		fmt.Printf("  Name:    %s\n", status.Name)
		if status.Cmdline != "" {
			fmt.Printf("  Command: %s\n", status.Cmdline)
		}
		if !status.StartTime.IsZero() {
			fmt.Printf("  Started: %s\n", status.StartTime.Format(time.RFC3339))
			fmt.Printf("  Uptime:  %s\n", formatDuration(status.Uptime))
		}

		return nil
	},
}

var serveCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Kill process on port and remove stale PID files",
	Long: `Kill any process listening on the specified port and clean up stale PID files.

This is useful for cleaning up orphaned processes that weren't
properly stopped, or removing stale PID files from crashed servers.
By default, attempts graceful termination first.

Examples:
  namelens serve cleanup                 # Cleanup default port (8080)
  namelens serve cleanup --port 9000     # Cleanup port 9000
  namelens serve cleanup --force         # Force kill without grace period`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Show what we're about to clean up
		proc, _ := daemon.FindProcessOnPort(serverPort)
		if proc != nil {
			fmt.Printf("Found process on port %d:\n", serverPort)
			fmt.Printf("  PID:  %d\n", proc.PID)
			fmt.Printf("  Name: %s\n", proc.Name)
		}

		result, err := daemon.Cleanup(serverPort, cleanupForce)
		if err != nil {
			return errwrap.WrapInternal(cmd.Context(), err, "failed to cleanup")
		}

		if result.ProcessKilled {
			fmt.Printf("Process terminated (PID %d)\n", result.PID)
		}
		if result.PIDFileRemoved {
			fmt.Printf("Stale PID file removed\n")
		}
		if !result.ProcessKilled && !result.PIDFileRemoved {
			fmt.Printf("Nothing to clean up on port %d\n", serverPort)
		}

		return nil
	},
}

func showAllServers() error {
	servers, err := daemon.ListServers()
	if err != nil {
		return err
	}

	if len(servers) == 0 {
		fmt.Println("No known server instances")
		return nil
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"Port", "Status", "PID", "Uptime", "Name"})

	for _, s := range servers {
		var status string
		var pid string
		var uptime string
		var name string

		switch {
		case s.Stale:
			status = "Stale"
			pid = fmt.Sprintf("%d", s.PID)
		case s.Running:
			status = "Running"
			pid = fmt.Sprintf("%d", s.PID)
			uptime = formatDuration(s.Uptime)
			name = s.Name
		default:
			status = "Stopped"
		}

		t.AppendRow(table.Row{s.Port, status, pid, uptime, name})
	}

	fmt.Println(t.Render())
	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

func init() {
	// Stop subcommand
	serveCmd.AddCommand(serveStopCmd)
	serveStopCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "server port")
	serveStopCmd.Flags().BoolVarP(&stopForce, "force", "f", false, "force kill without grace period")
	serveStopCmd.Flags().DurationVar(&stopGracePeriod, "grace-period", daemon.DefaultGracePeriod, "grace period for graceful shutdown")

	// Status subcommand
	serveCmd.AddCommand(serveStatusCmd)
	serveStatusCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "server port")
	serveStatusCmd.Flags().Bool("all", false, "show all known server instances")

	// Cleanup subcommand
	serveCmd.AddCommand(serveCleanupCmd)
	serveCleanupCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "server port")
	serveCleanupCmd.Flags().BoolVarP(&cleanupForce, "force", "f", false, "force kill without grace period")
}
