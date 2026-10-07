// Command omnidevx collects developer-experience telemetry from local
// AI coding agent history (Claude Code, Codex CLI, Kiro CLI) and lists and
// resumes coding-agent sessions.
//
// Usage:
//
//	omnidevx collect --person person:jane --since 2026-07-01 --until 2026-07-31
//	omnidevx sessions
//	omnidevx sessions show <id>
//	omnidevx sessions resume <id>
//
// The collect command reads session history from each agent's local store
// and writes canonical events to ~/.plexusone/omnidevx/data/. Events are
// deduplicated by ID, so re-running the same period is safe.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd(defaultDeps()).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "omnidevx:", err)
		os.Exit(1)
	}
}

func newRootCmd(deps deps) *cobra.Command {
	root := &cobra.Command{
		Use:   "omnidevx",
		Short: "Developer-experience telemetry and coding-agent session recovery",
		Long: `omnidevx collects developer-experience telemetry from local AI coding agent
history, and lists and resumes coding-agent sessions (Claude Code, Codex CLI).`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newCollectCmd(),
		newSessionsCmd(deps),
		&cobra.Command{
			Use:   "version",
			Short: "Print version",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, _ []string) {
				fmt.Fprintln(cmd.OutOrStdout(), "omnidevx v0.1.0")
			},
		},
	)
	return root
}
