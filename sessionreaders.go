package omnidevx

import (
	codex "github.com/plexusone/omni-openai/omnidevx"
	"github.com/plexusone/omnidevx-core/providers/claudecode"
	"github.com/plexusone/omnidevx-core/sessions"
)

// Session catalog types re-exported from omnidevx-core so consumers need
// one import path. Unlike the telemetry collectors, session readers read
// titles and prompt text from local harness storage; see the sessions
// package for the content-access contract.
type (
	Session            = sessions.Session
	SessionReader      = sessions.Reader
	SessionCatalog     = sessions.Catalog
	SessionListOptions = sessions.ListOptions
	SessionState       = sessions.State
	ResumeSpec         = sessions.ResumeSpec
	Harness            = sessions.Harness
)

// Session harnesses and states re-exported from omnidevx-core.
const (
	HarnessClaudeCode = sessions.HarnessClaudeCode
	HarnessCodex      = sessions.HarnessCodex

	SessionResumable = sessions.StateResumable
	SessionRunning   = sessions.StateRunning
	SessionUnknown   = sessions.StateUnknown
)

// Session reader constructors re-exported from the provider modules.
var (
	NewClaudeCodeSessionReader = claudecode.NewSessionReader
	NewCodexSessionReader      = codex.NewSessionReader
	NewSessionCatalog          = sessions.NewCatalog
	ResolveSession             = sessions.Resolve
)
