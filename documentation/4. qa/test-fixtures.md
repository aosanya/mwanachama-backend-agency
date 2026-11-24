# Test fixtures

Every test in this repo reads only files inside this repo (`testdata/`, plus `agency.operations.json` and `agency.agency.json`, which the repo ships). Nothing under `../` is opened, so `go test ./...` passes on a clone that has no sibling checkout and no `developer` repo. A missing fixture fails the test; it never skips it.

## Where each fixture came from

| Fixture | Used by | Copied from |
|---|---|---|
| `testdata/mwanachama-wakala.agency.json` | `import_test.go` | the developer repo, `documentation/4. qa/mwanachama-wakala/agency.json` |
| `testdata/hub-demo/farmers/agency.json`, `siaya-lab`, `boda-boda-riders` | `permission_hub_templates_test.go`, `permission_mount_list_test.go`, `risk_permissions_test.go`, `hub_demo_fixture_shape_test.go` | the Hub Steward demo, `submission/demo/<name>/agency.json` |
| `testdata/operations/comm.operations.json` | `permission_hub_agent_test.go` | `mwanachama-backend-comm/comm.operations.json` |
| `testdata/operations/forms.operations.json` | `permission_hub_agent_test.go` | `mwanachama-backend-forms/forms.operations.json` |
| `testdata/operations/sources.operations.json` | `permission_hub_agent_test.go` | `mwanachama-backend-sources/sources.operations.json` |
| `testdata/operations/actor.operations.json` | `permission_hub_agent_test.go` | `mwanachama-backend-actor/actor.operations.json` |
| `testdata/operations/ai.operations.json` | `permission_hub_agent_test.go` | `mwanachama-backend-ai/ai.operations.json` |
| `testdata/operations/riskmanager.operations.json` | `risk_permissions_test.go` | `mwanachama-backend-riskmanager/riskmanager.operations.json` |

The agency templates are in the AGD-009 shape: `workflow_objectives` link workflows to objectives and a step names its accountable role in `accountable_role_id`.

The three hub-demo files carry no `role_assignments`, no `groups` or `tags` key, no `powers` on a role and no `role_id`, `edit_role_id`, `delete_role_id` or `viewer_role_id` on a record. `hub_demo_fixture_shape_test.go` asserts it.

## Who may do what with a record in the hub-demo files

A record has no role of its own; a role may act on it only by carrying its permission codes. The record codes are the readable ones (`payments`, `paid_rides`).

| File | Record | Role that holds it | Carries |
|---|---|---|---|
| `farmers/agency.json` | `payments` | Group coordinator | `record.payments.add.*`, `.edit.*`, `.view.*` and `record.payments.delete` |
| `boda-boda-riders/agency.json` | `paid_rides` | Rider | `record.paid_rides.add.*`, `.edit.*`, `.view.*` and `record.paid_rides.delete` |

No other role carries a code of these records, and no step of these files is about a record, so no role gets a view through a step. `siaya-lab/agency.json` declares no record.

## What the fixtures are for

- The templates are imported through `ImportDocument`; the tests assert what the import and publish produce, so a template in an older shape fails them.
- The operations files let the permission tests check that every action a permission carries is declared by the module that serves it, under the tool name the agents call.

## Refreshing

A fixture is a snapshot. When a source file changes in a way a test here should see, copy the file over its fixture from the path in the table, run `go test ./...` and commit the fixture with the change that needed it. Nothing refreshes them on its own, and no test reaches back to the source.

The farmers template is kept in three places: the Hub Steward demo, `submission/demo/farmers/agency.json`, which is the source; this repo's `testdata/hub-demo/farmers/agency.json`; and wakala-api's `internal/api/http/testdata/agencies/farmers/agency.json`. A change to the demo is copied into the other two in the same sitting. To see whether a copy has drifted, from the container folder:

```sh
cmp projects/meta-global-ai-hackathon-community-hubs/submission/demo/farmers/agency.json mwanachama-backend-agency/testdata/hub-demo/farmers/agency.json
cmp projects/meta-global-ai-hackathon-community-hubs/submission/demo/farmers/agency.json mwanachama-wakala-api/internal/api/http/testdata/agencies/farmers/agency.json
```

The Siaya lab template drifts the same way (`siaya-lab/agency.json` here against the demo's), so the same check applies to it.
