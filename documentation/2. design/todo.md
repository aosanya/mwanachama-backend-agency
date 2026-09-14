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
| AGD-002 | P1 | 📋 | 🤖 agent | 3. implementation | `Module` catalog (git/taskmanager/actor/assetmanager/auth/comm/custody/forms/...), Draft-scoped, `Enabled` gates whether the gateway registers that module's already-existing `routes/` package (full replace of its per-module env var, not layered); `ObjectiveModule` cross table, free many-to-many, Draft-scoped, same-scope-only; `routes/module.go`+`mcp/module.go` confirmed in scope. Full decision record: [todo_details/AGD-002.md](todo_details/AGD-002.md). Two items left open there (catalog completeness for `digitaltwin`/`accounting` — each now has its own board task to grow a `routes/` package, [accounting W14](../../mwanachama-backend-accounting/documentation/3.%20implementation/todo.md) / [digitaltwin W11](../../mwanachama-backend-digitaltwin/documentation/3.%20implementation/todo.md) — and catalog authorship) — flagged for whoever implements, not blocking a start. | 👤 human (2026-09-13 dev-research.md session) | — |

AGD-001 is done for this repo — moved to [todo_done.md](todo_done.md).
Its cross-repo follow-ups (`mwanachama-wakala-api`, `mwanachama-wakala-studio`,
`mwanachama-backend-api-kazi`, the gateway's Postman coverage) are tracked
on [../3. implementation/todo.md](../3.%20implementation/todo.md), not
here — they're implementation-time work now, not open design questions.
