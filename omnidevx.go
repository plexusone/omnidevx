package omnidevx

import (
	githubdevx "github.com/plexusone/omni-github/omnidevx"
	codex "github.com/plexusone/omni-openai/omnidevx"
	core "github.com/plexusone/omnidevx-core"
	"github.com/plexusone/omnidevx-core/providers/claudecode"
	gitprovider "github.com/plexusone/omnidevx-core/providers/git"
)

// Canonical types re-exported from omnidevx-core so consumers need one
// import path.
type (
	Collector        = core.Collector
	CollectRequest   = core.CollectRequest
	CollectionResult = core.CollectionResult
	Diagnostic       = core.Diagnostic
	Event            = core.Event
	EventContext     = core.EventContext
	EventType        = core.EventType
	CollectionMode   = core.CollectionMode
	Period           = core.Period
	Provenance       = core.Provenance
	Source           = core.Source
	SubjectRef       = core.SubjectRef
)

// Event types re-exported from omnidevx-core.
const (
	EventSessionStarted   = core.EventSessionStarted
	EventSessionEnded     = core.EventSessionEnded
	EventPromptSubmitted  = core.EventPromptSubmitted
	EventMessageCompleted = core.EventMessageCompleted
	EventToolCompleted    = core.EventToolCompleted
	EventPatchApplied     = core.EventPatchApplied
	EventTaskStarted      = core.EventTaskStarted
	EventTaskCompleted    = core.EventTaskCompleted
	EventUsageRecorded    = core.EventUsageRecorded
	EventChangeCommitted  = core.EventChangeCommitted

	EventContributionRecorded = core.EventContributionRecorded
	EventContributionSnapshot = core.EventContributionSnapshot
	EventProfileSnapshot      = core.EventProfileSnapshot
)

// Collection modes re-exported from omnidevx-core.
const (
	ModeHistory = core.ModeHistory
	ModeOTel    = core.ModeOTel
	ModeHooks   = core.ModeHooks
	ModeAPI     = core.ModeAPI
	ModeSurvey  = core.ModeSurvey
)

// Diagnostic severities re-exported from omnidevx-core.
const (
	SeverityWarning = core.SeverityWarning
	SeverityError   = core.SeverityError
)

// Provider constructors and options re-exported so consumers can compose
// collectors without importing each provider module.
var (
	NewClaudeCodeCollector = claudecode.New
	NewCodexCollector      = codex.New
	NewGitCollector        = gitprovider.New
	NewGitHubCollector     = githubdevx.New
)

type (
	ClaudeCodeOptions = claudecode.Options
	CodexConfig       = codex.Config
	GitOptions        = gitprovider.Options
	GitHubConfig      = githubdevx.Config
)

// NewDefault returns an Engine with every collector that can be constructed
// in the current environment (local Claude Code and Codex CLI stores under
// the user's home directory). The git collector needs explicit repository
// roots, so callers add it themselves:
//
//	engine.Add(must(omnidevx.NewGitCollector(omnidevx.GitOptions{
//	    Roots: []string{"~/go/src"},
//	})))
func NewDefault() (*Engine, error) {
	claude, err := claudecode.New(claudecode.Options{})
	if err != nil {
		return nil, err
	}
	cdx, err := codex.New(codex.Config{})
	if err != nil {
		return nil, err
	}
	return New(claude, cdx), nil
}
