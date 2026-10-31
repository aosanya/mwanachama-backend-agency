# Role permissions (AG82)

Member app decisions 263, 265, 267 and 269 (2026-10-22): what each role can
do is a list of named permissions, declared in the agency definition, drafted and
published with the rest of its design.

## Where it lives

- `role.permissions` (`agency.blueprint.json`, `models.Role.Permissions`) is a JSON
  list of permission codes. Empty declares none; an agency whose roles declare
  none keeps today's behaviour (decision 267). It rides the role's create and
  update bodies unchanged, so routes and MCP tools carry it with no new
  address.
- The vocabulary is one declared place: `models/permissions_catalog.json`, read
  through `models.PermissionCatalog()` and `models.LookupPermission(code)`. Each entry
  has a `code`, a display `name` and `locked`.
- `models.DocumentRole.Permissions` carries the list into the publication
  document, and forking a draft from a publication restores it, so permissions
  version with the draft. The role list in the document is what wakala-api
  turns into held role kinds (decision 269, WK181).

## What the publish scan refuses

`checkPermissions` in the Issue scan raises an `error` (so publishing is refused
with `ErrPublishValidationFailed`) when a role declares:

- a code not in the vocabulary, or
- a permission locked to owners (decision 265). Locked permissions are held by owners
  whatever the definition says, so no role may claim one.

## The vocabulary, and what is not yet settled

Decision 263 names permissions by example ("publish a report, approve forms,
decide join requests, invite, record a promise's outcome, …") and decision
265 names the locked owner permissions. The catalog holds exactly those, with
codes chosen here:

| Code | Permission | Locked |
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
| `permission.override` | Change overrides on the permissions page | yes |

Open: the full list (the "…" in decision 263) and whether these codes match
what permissions PM21 and wakala-api WK179 will enforce. The catalog is the
one file to change when that is settled. The owner, agent and member columns
of decision 264 are not agency roles and are not modelled here.

## Permissions map to actions, and a role can be staff (AG88, decisions 410, 411)

Each permission in `models/permissions_catalog.json` lists the declared action ids it
allows (`actions`), so the permission check and the studio read one list.
`GET /permissions` (`agency.permission.list`, tool `agency_list_permissions`) serves the
catalog. A permission whose module has no matching declared action yet carries an
empty list: `register.edit`, `service.edit` and `appointment.manage`.
`board.manage` maps to catalog's grouping (`catalog.raised_item.group`,
`ungroup`) and `catalog.board_item.set_outcome`; `task.assign` to
taskmanager's `task.assign`, `task.assign_holder` and `task.unassign`.
`poll.run` names `"module": "voting"`, as the engage permissions name `engage`.

A duplicate tick is dropped on save: `CreateRole` and `UpdateRole` keep the
first occurrence of each code, in order. `checkPermissions` still refuses an
unknown or locked code at publish.

`role.staff` is one tick per definition role. It is published in the
document and carried through a fork. Whoever holds a staff role counts as
staff beside owners and agents, for tasks and for board replies; the check
itself is the host's (wakala-api WK218).

## The older permissions list their actions (AG89, decisions 439–441, 451)

Permissions are added as a feature needs them (decision 439). The nine older
permissions now list the actions the sibling modules declare today, read from
their `*.operations.json`:

| Permission | Actions |
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
  the body, so the host checks that the caller holds the permission for the kind
  posted (engage's own rule, decision 348); the action list alone does not
  tell a notice from an event.

The permissions other than these keep the lists AG88 gave them, with two
changes:

- `board_item.hide` also allows `catalog.raised_item.remove`: removing a
  raised item with a ground goes with hiding (decision 441).
- `catalog_entry.publish` is now `service.publish`, "Publish services"
  (decision 440). Nothing in this repo or the studio carried the old code; a
  stored definition that still does is refused at publish as an unknown
  permission.

Not listed, because nothing declares the action yet:

- voting has no close or publish operation; `poll.run` gains them when
  voting declares them.
- engage's `engage.pick.create` and `engage.post.fill` are the poster's own
  steps on a volunteer call and are left to the host's poster check rather
  than granted by the permission.
- Recording a promise's outcome is wakala-api WK178's, not declared.
- Acknowledging a raised item goes with Reply to board items once catalog
  CAT56 declares it (decision 441).
- Assigning, unassigning and closing a board item go with Manage the board
  once catalog CAT134 declares them (decision 451).
- Edit the register, Edit services and Manage appointments gain theirs in
  the rows that declare them (CAT50, CAT124, WK198).

A permission with no actions grants nothing; owners and agents act (decision
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
permission: staff read the twin without it (decision 498), and that check is the
host's.

## asset.manage (HS89, decision 39)

`asset.manage` lets an Assets Keeper register assets (`assetmanager.asset.create`, `.update`, `.list`, `.read`, `.balance`), look at locations, and allocate stock to a holder with `assetmanager.movement.post`. Accounting is deliberately not a permission. No permission exists yet for holding a seat or proposing a draft: `assetmanager.hold.create` reserves a quantity at a place until a time, which is not a seat, and `workspace.layout.propose` only previews a patch, so neither was invented into one.

## ai.account.manage, ai.role.manage, ai.use (HS128, decision 48)

`ai.account.manage` adds, changes and removes AI accounts, sets their keys and tests them (`ai.account.create`, `.update`, `.delete`, `.key_set`, `.test`). It is held by owners; no role definition claims it by default. `ai.role.manage` covers AI roles, their fine-tuning and history, and the limits on accounts and roles (`ai.limits.set`, `ai.role.create`, `.update`, `.delete`, `.list`, `.read`, `.revision_list`, `.restore`, and `ai.account.test`); owners and staff seats hold it. `ai.use` lets an agent call through the AI gateway (`ai.account.list`, `ai.usage.list`, `ai.usage.record`, `ai.limits.check`, and `ai.role.resolve_agency_role`, HS176 step 2); agents hold it. `ai.account.test` is deliberately not in `ai.use`. The `ai.role.*` ids are listed explicitly because a permission never carries a wildcard; they must be reconciled with the ai module's operations file when it declares its role addresses.

## The permissions a Hub Steward agent needs when it is linked to a role (HS176 step 2, W7)

A hub's agents (dispatcher, farmers' coordinator, network watcher, the answers
desk) reach the hub as one service identity that the owner links to one agency
role. From then on wakala-api lets the agent call only the actions that role's
permissions list (`agentPowered`), on every actor, comm, forms, sources and ai route
and tool. Before this change the catalog named none of the chat-thread, forms
authoring, answers, group-listing or role-resolve calls the agents make, so a
linked agent was refused all of them. The new permissions are read from what the
agents call (`submission/code` of the Hub Steward repo) against the
`*.operations.json` of the modules; a tool is its action id with the dots turned
into underscores, except the actor tools, which declare their own name.

| Permission | Name | Module | Actions | Agent tools |
| --- | --- | --- | --- | --- |
| `chat.thread` | Open and keep up with chat threads | `comm` | `comm.chat_thread.list`, `comm.chat_thread.create`, `comm.chat_thread.mark_read` | `comm_chat_thread_list`, `comm_chat_thread_create`, `comm_chat_thread_mark_read` |
| `form.compose` | Compose forms | `forms` | `forms.form.create`, `forms.question.create`, `forms.question.list` | `forms_form_create`, `forms_question_create`, `forms_question_list` |
| `form.answers.read` | Read the answers to a form | `forms` | `forms.answer.list_all`, `forms.question.list` | `forms_answer_list_all`, `forms_question_list` |
| `document.read` | Read documents | `sources` | `sources.source.list`, `sources.source.read`, `sources.chunk.list` | `sources_source_list`, `sources_chunk_list` |
| `group.read` | See the hub's rooms | none | `actor.group.list` | `actor_list_groups` |
| `member.read` | See who is in the hub | none | `actor.actor.list` | `actor_list_actors` |

The agent role also holds permissions that already existed: `chat.post` (the chat
messages: `comm_chat_message_list`, `comm_chat_message_post`), `form.send`
(`forms_form_publish`), `document.search` and `ai.use`.

Why they are shaped this way:

- Existing permission ids and the actions of `chat.post`, `document.search` and
  `form.send` do not change, so a stored definition that names them keeps the
  same meaning. Widening `chat.post` or `document.search` would have given
  every current holder more without anyone choosing it; the new calls are new
  permissions a role takes on purpose.
- Reading a form's answers is its own permission, apart from composing a form,
  because the answers are what members typed. `forms.question.list` sits in
  both: a composer needs it to see the form it is building, and a reader needs
  it to know what each answer answers.
- `document.read` repeats `sources.source.list` and `sources.source.read` from
  `document.search` so that a role that only reads documents (the answers desk
  lists the documents and walks their passages itself) does not also need
  search. A permission is a set of actions, so holding both is no wider than either.
- `member.read` is separate from `group.read` because listing every person in a
  hub is wider than listing its rooms. No agent code calls `actor_list_actors`
  today; the permission exists because the service's allowed tool list names it, and
  a template that does not need it can leave it out.
- `ai.use` gains `ai.role.resolve_agency_role` because every agent reads its
  role's instructions from `GET /instances/{id}/ai/agency-roles/{id}/resolve`
  before it asks the model. It does not gain `ai.role.resolve` (by AI role id),
  `ai.role.map` or any role writing action: those stay with `ai.role.manage`,
  which already held the resolve action.

Calls no permission gates, so no permission is in the catalog for them. wakala-api checks
that the caller is a member of the hub, not a permission, before `agency_propose_write`,
`agency_list_approvals` and `agency_amend_approval` (and their REST routes under
`/instances/{id}/approvals`); they have no action id, and amending is limited by
the approvals store to the proposer or an owner. `agency_list_instances` and
`GET /me` are also ungated. A catalog entry for them would promise a gate that
does not exist. `TestNoPermissionClaimsToGateTheApprovalToolsBecauseNoActionGatesThem`
fails if one is added before the host gates them.

Checked by `permission_hub_agent_test.go`: every call the agents make is carried by an
unlocked permission; each new permission lists exactly the actions above and nothing
nearby (no archive, no target, no create beyond the agents' own); the earlier
codes are all still present; a role may declare every one and still publish; and,
when the sibling repos are checked out beside this one, every action is declared
by its module under the tool name the agents call.

## The risk manager's reads (HS97)

`risk.read`, "Ask what could go wrong", module `riskmanager`, carries only the
reads a person asks the risk manager in the risk-owner journey:

| Question | Action |
|---|---|
| What are the threats | `riskmanager.threat.list` |
| What are the consequences | `riskmanager.consequence.list` |
| What are the preventive measures | `riskmanager.preventive_measure.list` |
| What are the mitigating measures | `riskmanager.mitigating_measure.list` |
| What relies on a control | `riskmanager.control.reliance` |
| What are the gaps | `riskmanager.gap.list` |
| Finding what to ask about | `riskmanager.risk.list`, `riskmanager.risk.read`, `riskmanager.risk.bowtie`, `riskmanager.control.list`, `riskmanager.control.read` |

No write action is carried by any unlocked permission. The work that closes a gap is
raised by the manager after approving a draft, never by the agent's own role.
`risk_permissions_test.go` fails if a non-GET action of the risk manager joins an
unlocked permission, if `risk.read` carries anything the risk manager does not
declare as a read, or if a demo template other than the lab's Hub Steward
holds it. The risk manager's operations file is read from the sibling checkout
and the test skips when it is absent.

The module is not in the module catalog and is not mounted: the mount is
HS95 on the hub project's board and RM1 on the risk manager's, and waits on the
owner's go-ahead. Until it is mounted the permission names actions no route serves.

## Controls (AG105, AGD-011)

Two unlocked permissions, module `agency`, held by roles like any other:

| Code | Name | Actions |
|---|---|---|
| `controls.edit` | Edit controls | `agency.control.create`, `agency.control.update`, `agency.control.delete`, `agency.control.attach`, `agency.control.detach` |
| `controls.approve` | Approve control changes | `agency.publication.create` |

Authoring and approving are separate codes so an agency can give them to different
roles. That is the maker and checker an auditor asks for. A role holding only
`controls.edit` drafts a control and cannot publish a change to an enforced one.

`ChangedEnforcedControls(ctx, draftID)` is what tells the host whether a publish needs
`controls.approve`. It returns the draft's enforced controls (those with an `action`)
that are new, or whose action, condition, outcome or workflow differ from the document
of the publication on the production channel. A workflow is compared by its code, not
its id, because a fork mints new ids. With no production publication every enforced
control counts as changed. An advisory control is never returned, and neither is an
enforced control that is unchanged since it was published.

The library does not know who is publishing. The host reads the caller's permissions:
when `ChangedEnforcedControls` is non-empty it demands `controls.approve` of the
publisher, and when it is empty publishing needs only the ordinary publish
permission. `control_changes_test.go` covers the library side; the host check belongs
on the wakala-api board.
