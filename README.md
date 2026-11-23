# mwanachama-backend-agency

A Go library, in the declared-domain shape of [mwanachama-backend-catalog](../mwanachama-backend-catalog), that models an **Agency**: goals, objectives, workflows, the work items that are their steps, roles, and the RACI labels (Responsible, Accountable, Consulted, Informed) that put a role on a step. Work happens in drafts; a publication is a tagged snapshot, and the issue scan reports what a draft is missing.

The objects are declared in `agency.blueprint.json` (39 objects) and the route table in `agency.operations.json` (151 operations); storage goes through `mwanachama-backend-shared/specstore` and the routes and MCP tools through `dispatch`. It is a plain Go library with no service of its own; `mwanachama-wakala-api` mounts it per instance.

See [CLAUDE.md](CLAUDE.md) for the current model and [documentation/](documentation/) for requirements, design and the task board.
