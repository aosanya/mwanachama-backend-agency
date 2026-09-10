# Design & architecture board

New board — this repo isn't part of the mirrored 7-repo platform set
(`developer`, `mwanachama-frontend-*`, `mwanachama-network-admin*`,
`mwanachama-orchestrator`, `mwanachama-website*`) that shares a `DSN-XXX`
board verbatim, so it gets its own local numbering: **`AGD-XXX`**
(Agency Design gap), starting at AGD-001. Completed rows move to
`todo_done.md` (create it on first completion) — this file is
pending-only.

Row format: `| ID | Pri | Status | Responsible | Phase | Task | Raised by | Depends On |`
— 📋 not started; Responsible is 🤖 agent (a routine works it), 👤 human (a
decision the user makes), or 🦸 super human (owner review).

| ID | Pri | Status | Responsible | Phase | Task | Raised by | Depends On |
|---|---|---|---|---|---|---|---|
| AGD-001 | P1 | 📋 | 🤖 agent | 0 | **`Objective` (new, 1:1 with `Goal`, Draft-scoped), `Agency.Values`/`.Rules` (agency-wide, immutable), Objective-scoped Value/Rule (Accountable-RACI-gated writes), RACI moved from `WorkItem` to `Objective`, and Mission/Vision deleted everywhere.** Full decision record, cross-repo impact list, and flagged open questions: [todo_details/AGD-001.md](todo_details/AGD-001.md). Touches `mwanachama-wakala-api` (AG11 HTTP handlers + MCP tools) and `mwanachama-wakala-studio` (identity UI) as breaking changes, and `mwanachama-backend-api-kazi`'s K6 (RACI lookup path changes). | 👤 human (2026-09-10 session) | — |
