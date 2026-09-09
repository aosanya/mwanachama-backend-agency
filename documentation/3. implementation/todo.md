# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG7 | P2 | 📋 | Seed the NeoLabHQ test Agency: all 21 personas from [NeoLabHQ/context-engineering-kit](https://github.com/NeoLabHQ/context-engineering-kit)'s `agents/` directory (`bug-hunter`, `business-analyst`, `change-expectation-agent`, `change-failure-agent`, `change-impact-agent`, `change-story-agent`, `code-explorer`, `code-quality-reviewer`, `code-reviewer`, `contracts-reviewer`, `developer`, `fpf-agent`, `historical-context-reviewer`, `judge`, `meta-judge`, `researcher`, `security-auditor`, `software-architect`, `tech-lead`, `tech-writer`, `test-coverage-reviewer`) become Roles under one software-type department, then exercise AG6's RACI matrix end-to-end against them. | AG6 |
| AG9 | P2 | 📋 | Add a `routes/` package covering the domain, matching `mwanachama-backend-actor`'s HTTP route conventions. | AG3, AG5, AG6 |
| AG10 | P3 | 📋 | Wire into `mwanachama-backend-api-gateway` (deps.go, migration, capability-gated routes) — same pattern as actor/taskmanager/git's own wiring tasks. | AG9 |
