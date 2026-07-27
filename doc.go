// Package omnidevx is the batteries-included distribution of the OmniDevX
// developer-experience telemetry domain. It re-exports the canonical types
// from omnidevx-core and composes the available collectors — Claude Code
// (thin, from core), Codex CLI (thick, from omni-openai), and Kiro CLI
// (thick, from omni-aws) — behind one import path.
//
//	engine := omnidevx.New(claudeCollector, codexCollector, kiroCollector)
//	results, err := engine.Collect(ctx, omnidevx.CollectRequest{...})
//
// OmniDevX is not OmniDXI (github.com/plexusone/omnidxi), which is a
// Digital Experience Intelligence facade over product-analytics platforms.
package omnidevx
