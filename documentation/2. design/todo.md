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

AGD-001, AGD-002, AGD-004 and AGD-005 are done for this repo — moved to
[todo_done.md](todo_done.md). Their cross-repo follow-ups
(`mwanachama-wakala-api`, `mwanachama-wakala-studio`,
`mwanachama-backend-api-kazi`, `mwanachama-backend-api-gateway`'s Postman
coverage / digitaltwin-accounting-assetmanager module wiring) are tracked
on [../3. implementation/todo.md](../3.%20implementation/todo.md) (this
repo) and each named repo's own board, not here — they're
implementation-time work now, not open design questions.
