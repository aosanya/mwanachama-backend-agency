# Role powers (AG82)

Member app decisions 263, 265, 267 and 269 (2026-10-22): what each role can
do is a list of named powers, declared in the agency definition, drafted and
published with the rest of its design.

## Where it lives

- `role.powers` (`agency.blueprint.json`, `models.Role.Powers`) is a JSON
  list of power codes. Empty declares none; an agency whose roles declare
  none keeps today's behaviour (decision 267). It rides the role's create and
  update bodies unchanged, so routes and MCP tools carry it with no new
  address.
- The vocabulary is one declared place: `models/powers_catalog.json`, read
  through `models.PowerCatalog()` and `models.LookupPower(code)`. Each entry
  has a `code`, a display `name` and `locked`.
- `models.DocumentRole.Powers` carries the list into the publication
  document, and forking a draft from a publication restores it, so powers
  version with the draft. The role list in the document is what wakala-api
  turns into held role kinds (decision 269, WK181).

## What the publish scan refuses

`checkPowers` in the Issue scan raises an `error` (so publishing is refused
with `ErrPublishValidationFailed`) when a role declares:

- a code not in the vocabulary, or
- a power locked to owners (decision 265). Locked powers are held by owners
  whatever the definition says, so no role may claim one.

## The vocabulary, and what is not yet settled

Decision 263 names powers by example ("publish a report, approve forms,
decide join requests, invite, record a promise's outcome, …") and decision
265 names the locked owner powers. The catalog holds exactly those, with
codes chosen here:

| Code | Power | Locked |
| --- | --- | --- |
| `report.publish` | Publish a report | no |
| `form.approve` | Approve forms | no |
| `join_request.decide` | Decide join requests | no |
| `member.invite` | Invite | no |
| `promise.record_outcome` | Record a promise's outcome | no |
| `agent.manage` | Add or remove agents | yes |
| `api_key.manage` | Manage API keys | yes |
| `agency.go_live` | Go live | yes |
| `agency.delete` | Delete the agency | yes |
| `power.override` | Change overrides on the powers page | yes |

Open: the full list (the "…" in decision 263) and whether these codes match
what permissions PM21 and wakala-api WK179 will enforce. The catalog is the
one file to change when that is settled. The owner, agent and member columns
of decision 264 are not agency roles and are not modelled here.

## Powers map to actions, and a role can be staff (AG88, decisions 410, 411)

Each power in `models/powers_catalog.json` lists the declared action ids it
allows (`actions`), so the permission check and the studio read one list.
`GET /powers` (`agency.power.list`, tool `agency_list_powers`) serves the
catalog. A power whose module has no matching declared action yet carries an
empty list: `register.edit`, `service.edit` and `appointment.manage`.
`board.manage` maps to catalog's grouping (`catalog.raised_item.group`,
`ungroup`) and `catalog.board_item.set_outcome`; `task.assign` to
taskmanager's `task.assign`, `task.assign_holder` and `task.unassign`.
`poll.run` names `"module": "voting"`, as the engage powers name `engage`.

A duplicate tick is dropped on save: `CreateRole` and `UpdateRole` keep the
first occurrence of each code, in order. `checkPowers` still refuses an
unknown or locked code at publish.

`role.staff` is one tick per definition role. It is published in the
document and carried through a fork. Whoever holds a staff role counts as
staff beside owners and agents, for tasks and for board replies; the check
itself is the host's (wakala-api WK218).

## The older powers list their actions (AG89, decisions 439–441, 451)

Powers are added as a feature needs them (decision 439). The nine older
powers now list the actions the sibling modules declare today, read from
their `*.operations.json`:

| Power | Actions |
| --- | --- |
| `member.invite` Invite | `actor.invitation.create`, `actor.invitation.check_batch`, `actor.invitation.send_batch` |
| `join_request.decide` Decide join requests | `actor.join_request.approve`, `actor.join_request.decline` |
| `form.approve` Approve forms | `forms.approval.approve`, `forms.approval.approve_slot`, `forms.approval.refuse` |
| `report.publish` Publish a report | `forms.results.publish` |
| `poll.run` Run a poll | `voting.poll.create`, `voting.choice.create` |
| `notice.post` Post a notice | `engage.post.create` |
| `event.post` Post an event or meeting | `engage.post.create` |
| `volunteer_call.post` Post a volunteer call | `engage.post.create` |
| `promise.record_outcome` Record a promise's outcome | none yet |

Why these and not fewer:

- The bulk invite is two actions: `check_batch` writes nothing and is the
  step before `send_batch`, so a holder who may send may check.
- `approve_slot` is the approval of a form whose round declares slots;
  without it a holder could approve only single-approver forms.
- A poll is made by `voting.poll.create` and then given its choices with
  `voting.choice.create`; one without the other is not a poll.
- engage declares one posting action for all three kinds. The kind is in
  the body, so the host checks that the caller holds the power for the kind
  posted (engage's own rule, decision 348); the action list alone does not
  tell a notice from an event.

The powers other than these keep the lists AG88 gave them, with two
changes:

- `board_item.hide` also allows `catalog.raised_item.remove`: removing a
  raised item with a ground goes with hiding (decision 441).
- `catalog_entry.publish` is now `service.publish`, "Publish services"
  (decision 440). Nothing in this repo or the studio carried the old code; a
  stored definition that still does is refused at publish as an unknown
  power.

Not listed, because nothing declares the action yet:

- voting has no close or publish operation; `poll.run` gains them when
  voting declares them.
- engage's `engage.pick.create` and `engage.post.fill` are the poster's own
  steps on a volunteer call and are left to the host's poster check rather
  than granted by the power.
- Recording a promise's outcome is wakala-api WK178's, not declared.
- Acknowledging a raised item goes with Reply to board items once catalog
  CAT56 declares it (decision 441).
- Assigning, unassigning and closing a board item go with Manage the board
  once catalog CAT134 declares them (decision 451).
- Edit the register, Edit services and Manage appointments gain theirs in
  the rows that declare them (CAT50, CAT124, WK198).

A power with no actions grants nothing; owners and agents act (decision
439).

## Edit the map (AG93, decision 498)

`map.edit`, "Edit the map", is unlocked and names `"module": "digitaltwin"`.
It lists every write the twin declares in `digitaltwin.operations.json`:

| Object | Actions |
| --- | --- |
| node | `digitaltwin.node.create`, `digitaltwin.node.update`, `digitaltwin.node.set_status` |
| link | `digitaltwin.link.create`, `digitaltwin.link.delete` |
| metric | `digitaltwin.metric.upsert` |
| observation | `digitaltwin.observation.record`, `digitaltwin.observation.record_batch` |
| retention | `digitaltwin.retention.set` |

The reads (`node.read`, `node.list`, `link.list`, `metric.list`,
`retention.list`, `observation.query`, `observation.latest`) are not in the
power: staff read the twin without it (decision 498), and that check is the
host's.

## asset.manage (HS89, decision 39)

`asset.manage` lets an Assets Keeper register assets (`assetmanager.asset.create`, `.update`, `.list`, `.read`, `.balance`), look at locations, and allocate stock to a holder with `assetmanager.movement.post`. Accounting is deliberately not a power. No power exists yet for holding a seat or proposing a draft: `assetmanager.hold.create` reserves a quantity at a place until a time, which is not a seat, and `workspace.layout.propose` only previews a patch, so neither was invented into one.

## ai.account.manage, ai.role.manage, ai.use (HS128, decision 48)

`ai.account.manage` adds, changes and removes AI accounts, sets their keys and tests them (`ai.account.create`, `.update`, `.delete`, `.key_set`, `.test`). It is held by owners; no role definition claims it by default. `ai.role.manage` covers AI roles, their fine-tuning and history, and the limits on accounts and roles (`ai.limits.set`, `ai.role.create`, `.update`, `.delete`, `.list`, `.read`, `.revision_list`, `.restore`, and `ai.account.test`); owners and staff seats hold it. `ai.use` lets an agent call through the AI gateway (`ai.account.read`, `ai.account.list`, `ai.usage.record`, `ai.limits.check`); agents hold it. `ai.account.test` is deliberately not in `ai.use`. The `ai.role.*` ids are listed explicitly because a power never carries a wildcard; they must be reconciled with the ai module's operations file when it declares its role addresses.
