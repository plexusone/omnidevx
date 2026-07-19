package omnidevx

import (
	"context"
	"errors"
	"fmt"

	core "github.com/plexusone/omnidevx-core"
)

// Engine runs a set of collectors over one request. Composition is by
// constructor injection; there is no global registry.
type Engine struct {
	collectors []core.Collector
}

// New returns an Engine over the given collectors.
func New(collectors ...core.Collector) *Engine {
	return &Engine{collectors: collectors}
}

// Add appends further collectors, returning the engine for chaining.
func (e *Engine) Add(collectors ...core.Collector) *Engine {
	e.collectors = append(e.collectors, collectors...)
	return e
}

// Collectors returns the composed collectors in registration order.
func (e *Engine) Collectors() []core.Collector {
	return e.collectors
}

// Collect runs every collector with the same request. One collector failing
// does not stop the others: successful results are always returned, and the
// error joins each failure annotated with its source. Callers therefore may
// receive both results and a non-nil error.
func (e *Engine) Collect(ctx context.Context, req core.CollectRequest) ([]*core.CollectionResult, error) {
	var (
		results []*core.CollectionResult
		errs    []error
	)
	for _, c := range e.collectors {
		if err := ctx.Err(); err != nil {
			return results, errors.Join(append(errs, err)...)
		}
		result, err := c.Collect(ctx, req)
		if err != nil {
			src := c.Source()
			errs = append(errs, fmt.Errorf("collect %s/%s: %w", src.Provider, src.Product, err))
			continue
		}
		results = append(results, result)
	}
	return results, errors.Join(errs...)
}

// Events flattens collection results into a single event slice, preserving
// per-collector order.
func Events(results []*core.CollectionResult) []core.Event {
	var events []core.Event
	for _, r := range results {
		if r != nil {
			events = append(events, r.Events...)
		}
	}
	return events
}
