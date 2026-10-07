package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"

	"github.com/plexusone/omnidevx"
	core "github.com/plexusone/omnidevx-core"
	"github.com/plexusone/omnidevx-core/store"
)

func newCollectCmd() *cobra.Command {
	var (
		person   string
		since    string
		until    string
		storeDir string
		dryRun   bool
	)
	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect events from local AI agent history and write to store",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return runCollect(person, since, until, storeDir, dryRun)
		},
	}
	f := cmd.Flags()
	f.StringVar(&person, "person", "", "subject person ID (required, e.g. person:jane)")
	f.StringVar(&since, "since", "", "start date YYYY-MM-DD (required)")
	f.StringVar(&until, "until", "", "end date YYYY-MM-DD (required)")
	f.StringVar(&storeDir, "store", "", "store directory (default ~/.plexusone/omnidevx/data)")
	f.BoolVar(&dryRun, "dry-run", false, "collect but don't write to store")
	return cmd
}

func runCollect(person, since, until, storeDir string, dryRun bool) error {
	if person == "" || since == "" || until == "" {
		return errors.New("--person, --since, and --until are required")
	}

	start, err := time.Parse("2006-01-02", since)
	if err != nil {
		return fmt.Errorf("parse --since: %w", err)
	}
	end, err := time.Parse("2006-01-02", until)
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
		Subject: core.SubjectRef{PersonID: person},
	}

	log.Printf("collecting events for %s from %s to %s", person, since, until)
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

	if dryRun {
		log.Printf("dry-run: skipping store write")
		return nil
	}

	s, err := store.Open(store.Options{Dir: storeDir})
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
