// Command omnidevx collects developer-experience telemetry from local
// AI coding agent history (Claude Code, Codex CLI, Kiro CLI) and writes
// canonical events to the OmniDevX store.
//
// Usage:
//
//	omnidevx collect --person person:jane --since 2026-07-01 --until 2026-07-31
//
// The collect command reads session history from each agent's local store
// and writes canonical events to ~/.plexusone/omnidevx/data/. Events are
// deduplicated by ID, so re-running the same period is safe.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/plexusone/omnidevx"
	core "github.com/plexusone/omnidevx-core"
	"github.com/plexusone/omnidevx-core/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "collect":
		if err := runCollect(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "version":
		fmt.Println("omnidevx v0.1.0")
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `omnidevx - developer-experience telemetry collector

Commands:
  collect   Collect events from local AI agent history and write to store
  version   Print version
  help      Show this help

Use "omnidevx collect -h" for collect options.
`)
}

func runCollect(args []string) error {
	fs := flag.NewFlagSet("collect", flag.ExitOnError)
	person := fs.String("person", "", "subject person ID (required, e.g. person:jane)")
	since := fs.String("since", "", "start date YYYY-MM-DD (required)")
	until := fs.String("until", "", "end date YYYY-MM-DD (required)")
	storeDir := fs.String("store", "", "store directory (default ~/.plexusone/omnidevx/data)")
	dryRun := fs.Bool("dry-run", false, "collect but don't write to store")
	fs.Parse(args)

	if *person == "" || *since == "" || *until == "" {
		fs.Usage()
		return errors.New("--person, --since, and --until are required")
	}

	start, err := time.Parse("2006-01-02", *since)
	if err != nil {
		return fmt.Errorf("parse --since: %w", err)
	}
	end, err := time.Parse("2006-01-02", *until)
	if err != nil {
		return fmt.Errorf("parse --until: %w", err)
	}
	end = end.Add(24*time.Hour - time.Nanosecond)

	engine, err := omnidevx.NewDefault()
	if err != nil {
		return fmt.Errorf("create engine: %w", err)
	}

	ctx := context.Background()
	req := core.CollectRequest{
		Period:  core.Period{Start: start, End: end},
		Subject: core.SubjectRef{PersonID: *person},
	}

	log.Printf("collecting events for %s from %s to %s", *person, *since, *until)
	log.Printf("collectors: %d (Claude Code, Codex CLI, Kiro CLI)", len(engine.Collectors()))

	results, collectErr := engine.Collect(ctx, req)

	var totalEvents int
	var totalDiagnostics int
	for _, r := range results {
		if r != nil {
			totalEvents += len(r.Events)
			totalDiagnostics += len(r.Diagnostics)
			log.Printf("  %s/%s: %d events, %d diagnostics",
				r.Source.Provider, r.Source.Product, len(r.Events), len(r.Diagnostics))
		}
	}

	if collectErr != nil {
		log.Printf("collection errors (partial results may still be usable): %v", collectErr)
	}

	if totalEvents == 0 {
		log.Printf("no events found")
		return nil
	}

	log.Printf("total: %d events, %d diagnostics", totalEvents, totalDiagnostics)

	if *dryRun {
		log.Printf("dry-run: skipping store write")
		return nil
	}

	s, err := store.Open(store.Options{Dir: *storeDir})
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}

	events := omnidevx.Events(results)
	writeResult, err := s.Write(ctx, events)
	if err != nil {
		return fmt.Errorf("write to store: %w", err)
	}

	log.Printf("wrote %d events to %s (%d duplicates skipped)",
		writeResult.Written, s.Dir(), writeResult.Duplicates)

	return nil
}
