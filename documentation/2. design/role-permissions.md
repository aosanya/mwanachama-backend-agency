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

- a code not in the vocabulary, unless it is a record code that a record of the draft
  gives: `record.<record code>.<add|edit|view>.<field code or *>` or
  `record.<record code>.delete` (ISS-151, ISS-182, see
  [workflow-records.md](workflow-records.md)). The record code is the record's
  readable code, such as `record.tender.edit.closing_date`; the counter code `RC-n`
  is deprecated as a record code (D58) and is no longer given by any record. A
  rename of the record code moves the roles that carry it in the same save, or
- a `record.` code that no record of the draft gives, among them the plain
  `record.<record code>.add`, `.edit` and `.view`, which no longer exist, a field the
  record does not have, and a `delete` with a field part (a field's code is its own
  stored label on a hidden key; following a rename of a field code is not built yet, see
  [workflow-records.md](workflow-records.md)). The star is accepted as written, even for
  a record with no fields (D30). The refusal says why, by what the code is (ISS-183, D20, D30, D36):
  a plain code is told it is now `record.<record code>.<action>.*` for every field or
  `record.<record code>.<action>.<field>` for one; a code that names a field the record
  lacks is told `record "tender" has no field "nothing"`; a code of a record the draft
  lacks is told no record of the draft gives it. The refusal on create, update, set,
  import and publish names the role and the code, and the finding on a field removed
  from a record while a role still names it names the role, the code, the record and
  the field, or
- a permission locked to owners (decision 265). Locked permissions are held by owners
  whatever the definition says, so no role may claim one.

## The vocabulary, and what is not yet settled

Decision 263 names permissions by example ("publish a report, approve forms,
decide join requests, invite, record a promise's outcome, …") and decision
265 names the locked owner permissions. These ten are the ones those two
decisions name, with codes chosen here. The catalogue has grown since: later
sections add some, the last table lists the rest, and
`models/permissions_catalog.json` is the whole list:

| Code | Permission | Locked |
| --- | --- | --- |
| `report.publish` | Publish a report | no |
| `report.read` | Find and read a report published to your groups | no |
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
`board.read` maps to `catalog.board_item.list` and `catalog.board_item.tally_by_area` and allows reading only.
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
| `report.publish` Publish a report | `forms.results.publish`, `forms.results.files`, `voting.result.publish` |
| `report.read` Find and read a report published to your groups | `forms.results.list_mine`, `forms.results.read_text` |
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

`asset.manage` lets an Assets Keeper register assets (`assetmanager.asset.create`, `.update`, `.list`, `.read`, `.balance`), look at locations (`assetmanager.location.list`, `.read`) and add one (`assetmanager.location.create`), allocate stock to a holder with `assetmanager.movement.post`, and read the movements (`assetmanager.movement.list`, `.read`). Accounting has its own permissions, `accounting.manage` and `accounting.view` (MUN1052, below). No permission exists yet for holding a seat or proposing a draft: `assetmanager.hold.create` reserves a quantity at a place until a time, which is not a seat, and `workspace.layout.propose` only previews a patch, so neither was invented into one.

## accounting.manage, accounting.view (MUN1052)

`accounting.manage` lets a finance officer keep the books and set budgets: open and close accounts (`accounting.account.open`, `.close`), post entries (`accounting.entry.post`), set budget lines (`accounting.budget.set`), and read everything below. `accounting.view` is the read-only half: list and read accounts and their balance (`accounting.account.list`, `.read`, `.balance`), list entries (`accounting.entry.list`), list and read budget lines (`accounting.budget.list`, `.read`) and financial reports (`accounting.report.list`, `.read`). Both belong to the `accounting` module and are listed in `permissions_catalog.json` of wakala-api, the way `council.manage` and `council.view` are.

## ai.account.manage, ai.role.manage, ai.use (HS128, decision 48)

`ai.account.manage` adds, changes and removes AI accounts, sets their keys and tests them (`ai.account.create`, `.update`, `.delete`, `.key_set`, `.test`, `.default_set`), and refreshes the model lists (`ai.model.refresh`). It is held by owners; no role definition claims it by default. `ai.role.manage` covers AI roles, their fine-tuning and history, and the limits on accounts and roles (`ai.limits.set`, `ai.role.create`, `.update`, `.delete`, `.list`, `.read`, `.map`, `.resolve`, `.resolve_agency_role`, `.set_limits`, `ai.revision.list`, `ai.revision.restore`, and `ai.account.test`); owners and staff seats hold it. `ai.use` lets an agent call through the AI gateway (`ai.account.list`, `ai.usage.list`, `ai.usage.record`, `ai.limits.check`, `ai.role.resolve_agency_role`, HS176 step 2, `ai.account.chat`, `ai.model.list`, `ai.chat_model.list` and `ai.chat_model.pick`); agents hold it. `ai.account.test` is deliberately not in `ai.use`. The `ai.role.*` ids are listed explicitly because a permission never carries a wildcard; they must be reconciled with the ai module's operations file when it declares its role addresses.

## The permissions a Hub Steward agent needs when it is linked to a role (HS176 step 2, W7)

A hub's agents (dispatcher, farmers' coordinator, network watcher, the answers
desk) reach the hub as one service identity that the owner links to one agency
role. From then on wakala-api lets the agent call only the actions that role's
permissions list (`agentPermissioned`), on every actor, comm, forms, sources and ai route
and tool. Before this change the catalog named none of the chat-thread, forms
authoring, answers, group-listing or role-resolve calls the agents make, so a
linked agent was refused all of them. The new permissions are read from what the
agents call (`submission/code` of the Hub Steward repo) against the
`*.operations.json` of the modules; a tool is its action id with the dots turned
into underscores, except the actor tools, which declare their own name.

| Permission | Name | Module | Actions | Agent tools |
| --- | --- | --- | --- | --- |
| `chat.thread` | Open and keep up with chat threads | `comm` | `comm.chat_thread.list`, `comm.chat_thread.read`, `comm.chat_thread.create`, `comm.chat_thread.mark_read`, `comm.chat_room.set_question` | `comm_chat_thread_list`, `comm_chat_thread_read`, `comm_chat_thread_create`, `comm_chat_thread_mark_read`, `comm_chat_room_set_question` |
| `form.compose` | Compose forms | `forms` | `forms.form.create`, `forms.question.create`, `forms.question.list` | `forms_form_create`, `forms_question_create`, `forms_question_list` |
| `form.answers.read` | Read the answers to a form | `forms` | `forms.answer.list_all`, `forms.question.list` | `forms_answer_list_all`, `forms_question_list` |
| `document.read` | Read documents | `sources` | `sources.source.list`, `sources.source.read`, `sources.chunk.list` | `sources_source_list`, `sources_chunk_list` |
| `group.read` | See the hub's rooms | none | `actor.group.list` | `actor_list_groups` |
| `member.read` | See who is in the hub | none | none | none |

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
- `member.read` carries no action. Listing the people of a hub is
  `resident.read`, which opens a person's name and address, so no hub agent
  lists members and no hub agent holds `resident.read`.
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
codes are all still present; a role may declare every one and still publish; and
every call the agents make is declared by its module under the tool name the
agents call, as is every action of `chat.thread`, `chat.reach`, `form.compose`,
`form.answers.read`, `document.read`, `group.read`, `member.read`, `member.view`
and `ai.use`. The declarations are read from the copies under `testdata/operations/`,
never from a sibling checkout: a missing copy fails the test, and
`documentation/4. qa/test-fixtures.md` says where each copy came from.

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
holds it. The risk manager's operations file is read from the copy at
`testdata/operations/riskmanager.operations.json`, and a missing copy fails the test.

The module is not in the module catalog and is not mounted: the mount is
HS95 on the hub project's board and RM1 on the risk manager's, and waits on the
owner's go-ahead. Until it is mounted the permission names actions no route serves.

## Controls (AG105, AGD-011)

Two unlocked permissions, module `agency`, held by roles like any other:

| Code | Name | Actions |
|---|---|---|
| `controls.edit` | Edit controls | `agency.control.create`, `agency.control.update`, `agency.control.delete`, `agency.control.attach`, `agency.control.detach`, `agency.hazard.create`, `agency.hazard.update`, `agency.hazard.delete`, `agency.threat.create`, `agency.threat.update`, `agency.threat.delete`, `agency.consequence.create`, `agency.consequence.update`, `agency.consequence.delete`, `agency.risk_threat.create`, `agency.risk_threat.delete`, `agency.risk_consequence.create`, `agency.risk_consequence.delete` |
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

## The coordinator's and the member's actions (HS532, HS537)

A seated agent is refused any action its role's permissions do not carry, so the actions the coordinator and member flows call are named here. `permission_catalogue_covers_test.go` lists each one against its permission.

| Permission | Change | Actions |
| --- | --- | --- |
| `market.group_order` | now "Start, revise, run and call off a group order" | adds `market.group_order.poster`, `market.group_order.line_paid`, `market.group_order.decide_silent_line` |
| `group.join` | new, module none | `actor.group.list_joinable`, `list_discoverable`, `find_by_code`, `join`, `join_by_code`, `actor.invitation.accept`, `decline` |
| `chat.archive` | new, module `comm` | `comm.chat_thread.archive`, `unarchive`, `list_archived` |
| `invitation.read` | new, module none | `actor.invitation.list_for_group`, `actor.invitation.read` |
| `member.view` | adds | `actor.actor.read_screen_name_by_code`, `actor.actor.standing` |
| `group.view` | adds | `actor.assignment.home_counts` |
| `market.ask_answer` | new, module `market` | `market.ask.answer` |

Why this shape:

- The coordinator's three steps are the default workflow's coordinator steps, so they ride with the permission that starts and calls off the order. A role that already holds `market.group_order` gains them.
- `chat.thread` is pinned to what the hub agents do (`permission_hub_agent_test.go`), and `member.invite` to the actions AG89 listed, so archiving a thread and reading invitations are new permissions, not a widening of those two.
- `market.group_order.answer_revision` is in no permission. It is the picker's own answer to revised terms, and the market test refuses any permission carrying an answer except `market.ask_answer`. The coordinator uses `decide_silent_line`.
- `market.ask.answer` is the poster's own reply to an ask, so it is a permission of its own, "Answer an ask as the poster", and `market_permissions_test.go` fails if any other permission carries it.
- The actions that read or write only the caller's own state (`actor.read_mine`, `group.list_mine`, `forms.form.list_waiting`, the live records and the like) are in no permission at all. The host lets every seated agent do them; the list is in wakala-api's `agent-role-permissions.md`.

## Every other permission in the catalogue (HS724)

The tables above leave some permissions out or give them only a paragraph. This table gives each of the rest one row, so every permission in `models/permissions_catalog.json` has a row on this page. The five permissions locked to owners are in the table under "The vocabulary".

"Owning module" is the module the permission's action ids belong to, the part before the first dot. The catalogue's own `module` field is empty on some of these (`form.send` and the actor ones among them), so this column does not read it. "What a holder can do" says the permission's actions in words. For the actor, comm, forms, sources, ai and riskmanager modules the words come from the titles and descriptions in their snapshots under `testdata/operations/`. `instance`, `market`, `catalog`, `digitaltwin`, `assetmanager` and `taskmanager` have no snapshot here, so for them the words are the action ids and the permission's catalogue name and nothing more. A permission with no action grants nothing (decision 439).

| Code | Permission | Locked | Owning module | What a holder can do |
| --- | --- | --- | --- | --- |
| `join_request.operate` | Read join requests and decide them as the operator | no | `actor` | List the join requests made to one group and read one. Approve or decline a pending request on the agency's behalf, with no leading seat asked: an approval registers the actor at the group in the same write, and a declined request is kept so the actor may ask again. The actor module says its mount grants the two decisions only to the agency's operators. |
| `mount.list` | See the agency's modules and their names | no | `instance` | See the agency's modules and the names it gave them. Its one action is `instance.mount.list`. |
| `role.assign` | Give a role and take it back | no | `actor` | Grant an actor a seat of some kind in a group, and revoke a seat. A grant is refused when the kind is retired, when the actor or the group does not exist, and when the actor already holds a live seat of that kind at that group; the grantor is the caller, never the body. Revoking a seat that already ended keeps the first reason and date. |
| `market.browse` | Browse the market | no | `market` | List the posters, offers, notices and group orders, and list the kinds. Read one group order. Read the trail of a group order and of an order. |
| `market.categorise` | Put a product under another category | no | `market` | Put a product under another category. |
| `market.offer` | Post offers and keep stock | no | `market` | Create and update offers. Set and read stock. |
| `market.notice` | Post a market notice | no | `market` | Create a market notice. |
| `market.ask` | Ask a poster a question | no | `market` | Create an ask to a poster. List asks and list yeses. |
| `market.advise` | Advise a farmer on the quotes to a notice | no | `market` | List quotes. Create and list drafts. |
| `market.group_order_join` | Join a group order | no | `market` | Join a group order. |
| `market.group_order_send` | Send a group order to the supplier | no | `market` | Send a group order to the supplier. |
| `market.group_order_promise` | Answer a group order as the supplier | no | `market` | Answer a group order as the supplier (`market.group_order.promise`). |
| `market.poster` | Apply as a poster | no | `market` | Apply as a poster. |
| `form.send` | Send a form | no | `forms` | Open a form for answers. A public form gets its shareable link here, a member form reaches every group it was aimed at, and opening it twice changes nothing. |
| `form.submit` | Submit a form for approval | no | `forms` | Hand a draft form to whoever clears it, which opens a round that waits for an answer. Pull a form back to draft, which closes the round that was waiting. |
| `form.answer` | Answer a form | no | `forms` | Send every answer to a form at once, as one of the groups the form was aimed at that the caller belongs to. All are checked before any is written, and answering again as the same group edits what was given. |
| `form.close` | Close a form | no | `forms` | Close an open form so it stops taking answers. Closing twice changes nothing. |
| `form.payment` | Mark a form payment received | no | `forms` | Record that the person who sent a form has paid, or has not. Nothing here takes a payment: a person records it. |
| `form.manage` | Edit, delete and translate forms | no | `forms` | List and read forms. Change a draft's title, window, audience and how it is collected, and delete a form that has not been published. Rewrite, remove and move a draft's questions, and replace a published question with a new version. List a question's choices and add one. Aim a draft at a group, list its aims and remove one. Put a translation, list every translation of a form, list the reviewed texts in one language, and list the languages that have a reviewed text. |
| `form.results` | Read a form's results, sends and approvals | no | `forms` | Read the newest report on a form, for a caller in a group it was published to or inside one. Read the caller's own answers, and the previous period's. List who has sent a form and who missed a closed period. List the groups anonymous respondents declared and count them per group. List a form's rounds of approval and the forms waiting for a decision. List the groups a form reached and how far it got. |
| `member.remove` | Remove a member | no | `actor` | Its one action, `actor.assignment.delete`, is not declared in the actor snapshot, so nothing in the snapshot is open to a holder through it. |
| `group.settings` | Change group settings | no | `actor` | Choose how actors join a group: open lets them in at once, confirmed holds each request for a decision. Hide a group, so it is reached only by invitation and never listed as joinable, or show it again. |
| `group.manage` | Create and arrange groups | no | `actor` | Create a group. Change a group's name, node type or anchor level. Move a group under another; the root cannot move and no group may move beneath itself. Set or clear the id of the digital twin node that is a group's area. Give a tag to a group and take one off, and disable and enable a group (`instance.group_tag.give`, `instance.group_tag.take`, `instance.group.disable`, `instance.group.enable`, served by the hub). A group administrator is whoever holds this permission (D78). |
| `role_kind.manage` | Manage role kinds | no | `actor` | Create, list and read role kinds. Retire a kind, which is refused while anyone still holds one, and make a retired kind grantable again. Mark whether a kind leads its group. List who holds a kind. |
| `code_format.manage` | Set the member code format | no | `actor` | Read and set how many groups of three characters newly drawn member and group codes have. Codes already drawn keep theirs. |
| `board_item.reply` | Reply to board items | no | `catalog` | Send a reply to a board item. |
| `board_item.hide` | Hide board items | no | `catalog` | Set whether a board item is hidden. Remove a raised item. |
| `board.manage` | Manage the board | no | `catalog` | Group raised items and ungroup them. Set the outcome of a board item. |
| `board.read` | Read the board and its counts | no | `catalog` | List the board's items and read the count by area. It never groups, hides, reassigns or sets an outcome; the hub's copy of the catalogue follows in the hub-copy issue, and which template roles hold it is the owner's call. |
| `resident.read` | Open a resident's name, address and bills | no | none | List the people of the hub, read one person's record (name and address) and read the bills and payments of a resident. The Zöldhalom template gives it to the town clerk and the finance officer; the owner gives it to other seats from Roles and Permissions. A person without it sees neither the list of residents nor a resident's page. |
| `register.edit` | Edit the register | no | none yet | No action is declared for it yet, so it grants nothing. |
| `map.edit` | Edit the map | no | `digitaltwin` | Create and update nodes and set a node's status. Create and delete links. Upsert a metric. Record observations one at a time or in a batch. Set retention. The reads are not in it; see "Edit the map". |
| `task.assign` | Assign tasks | no | `taskmanager` | Assign a task, assign it to a holder, and unassign it. |
| `service.edit` | Edit services | no | none yet | No action is declared for it yet, so it grants nothing. |
| `service.publish` | Publish services | no | `catalog` | Set the visibility of a service entry. |
| `service.read` | See the services | no | `catalog` | List the service entries. |
| `appointment.manage` | Manage appointments | no | none yet | No action is declared for it yet, so it grants nothing. |
| `chat.post` | Post in chat | no | `comm` | Post a message in a thread of a room the caller belongs to; the author is always the session's. List the posts of a thread, oldest first, and read one message by its id. |
| `notification.read` | Read notifications and their deliveries | no | `comm` | List the caller's own notifications, newest first. List the push attempts one notification made, which is the record of whom it reached. |
| `chat.reach` | See which rooms and threads one person can read | no | `comm` | List every thread one named person may read across the rooms they belong to, each with its structure, label, audience and the reason they can read it: group, asked or leads. It carries no post, no unread count and no read point, and never lists direct messages. |
| `document.search` | Search documents | no | `sources` | Find the passages closest in meaning to a query vector, each with its source and that source's verification. List sources and read one. |
| `asset.manage` | Register and allocate assets | no | `assetmanager` | Create, update, list and read assets, and read an asset's balance. List, read and create locations. Post a movement, and list and read movements. See "asset.manage" above for what it leaves out. |
| `ai.account.manage` | Manage AI accounts | no | `ai` | Add, change and remove AI accounts. Replace an account's key, which is written and never read back. Test an account. Refresh the model list from a provider. Mark the agency's default account. |
| `ai.role.manage` | Manage AI roles and limits | no | `ai` | Add, change, remove, list and read AI roles. Map an AI role to an account and to the agency role it speaks for. List an AI role's revisions and restore one. Set the limits of an AI role and of an account. Resolve an AI role, by its own id or by the agency role it speaks for. Test an account. |
| `ai.use` | Call through the AI gateway | no | `ai` | List an agency's accounts and their use. Record one call's use, and check whether a call may go ahead. Resolve the AI role that speaks for an agency role. Chat with an account's model. List the models and the models the chat offers, and pick the caller's chat model. |
| `ai.account.read` | Read AI account settings | no | `ai` | Read one account, and read the agency's default account. |
| `ai.chat` | Ask the assistant | no | `ai` | Chat with an account's model, list the models the chat offers and pick the caller's chat model, without recording usage or listing accounts. Read the assistant's limits and the caller's own use today and this month. |
| `risk.read` | Ask what could go wrong | no | `riskmanager` | Read the risk register: list and read risks, read a risk's bow-tie, list threats, consequences, preventive and mitigating measures, controls and gaps, and read what relies on a control. It writes nothing; see "The risk manager's reads". |
| `review.run` | Run a review of a draft | no | `agency` | Start a review of a draft. The review is signed by the caller. No role is given it by default. |
| `insight.count` | Count insights by subject | no | `insights` | Count the insights recorded about one subject. Small cells fold into an other cell so a count never names a few people. |
| `task.read` | See tasks, their states, backlogs and what they wait on | no | `taskmanager` | List tasks, those overdue and those pending, find tasks of a backlog by id, code or words, read one by id or by name, list assignments and task states, list backlogs and who has access, and traverse the relationships between tasks. |
| `task.edit` | Add and change tasks, and move them between steps and backlogs | no | `taskmanager` | Create, update, place and move tasks, move one to a step, and create and delete the relationships between tasks. |
| `backlog.manage` | Add, change and delete backlogs and their steps | no | `taskmanager` | Create, rename, archive, unarchive, delete and manage backlogs, add, rename, remove and reorder their steps, give and take access to a backlog, and set its owner group. |
| `late_matters.read` | See the raised items past their answer time | no | `forms` | List the raised items whose answer time has passed. |
| `rule.read` | Read rules, decisions and questions | no | `taskmanager` | Read and list rules, decisions and questions, list the issues of a rule and the rules of an issue, list the file links of a rule, the rules a decision produced and the rules that may contradict each other, and list the rules, decisions and questions that bear on a backlog. |
| `rule.edit` | Write and change rules, decisions and questions | no | `taskmanager` | Create, update, set the status of, supersede and answer rules, decisions and questions, link a rule to an issue and unlink it, add and remove a rule's file links, and add a reach tag to a backlog. |
| `case.vote` | Vote on cases, see the votes and take back your own | no | `taskmanager` | Vote on a case, take the caller's own vote back, and list the votes on a case. |
| `consent.read` | Read the consent text in force and one's own consent | no | `custody` | Read the consent text in force and the caller's own consent. |
| `consent.give` | Agree to the consent text and withdraw | no | `custody` | Agree to the consent text in force and withdraw that agreement. |
| `consent.publish` | List the consent versions and publish a consent text | no | `custody` | List the consent versions and publish a new consent text. |
| `accounting.manage` | Keep the books and set budgets | no | `accounting` | Open and close accounts, post entries, set and delete budgets, record bills, payments, tenders and quotes, send upward reports, and read accounts, budgets, reports, milestones and certificates. |
| `accounting.view` | Read accounts, budgets and reports | no | `accounting` | Read accounts, entries, budgets, reports, bills, payments, tenders, quotes, spending totals and entries by dimension, milestones and certificates. |
| `application.correct` | Correct a sent application | no | `forms` | Correct a sent application. |
| `application.decide` | Approve or refuse a sent application | no | `forms` | Approve or refuse a sent application. |
| `application.file` | File an application for a resident | no | `forms` | File an application for a resident, and list the published forms that reach one person. |
| `application.inspect` | Record an inspector's finding on a sent application | no | `forms` | Record an inspector's finding on a sent application, and list the sends the caller's role inspects. |
| `appointment.view_person` | See one person's bookings | no | `forms` | List one person's bookings. |
| `audit.view` | View the audit log | no | none | List the entries of the audit log. |
| `consent.record` | Record a person's consent given on paper | no | `custody` | Take a person's consent given on paper. |
| `council.manage` | Put council meetings on record and read them | no | `council` | Create, update, list and read council meetings, set and read their agendas and minutes, and create, update, list and read resolutions and bylaws. |
| `council.view` | Read council meetings | no | `council` | List and read council meetings, their agendas and minutes, resolutions and bylaws. |
| `declaration.read` | Read the register of declarations of interest | no | none | List the register of declarations of interest. |
| `digitaltwin.manage` | Record addresses, holdings and land use of places and read them | no | `digitaltwin` | Create, update, list and read addresses, holdings and land use, and list and read the places they belong to. |
| `digitaltwin.view` | Read addresses, holdings and land use of places | no | `digitaltwin` | List and read addresses, holdings and land use, and the places they belong to. |
| `location.edit` | Change a place | no | `assetmanager` | Change a place. |
| `member.add` | Add a person who lives here | no | none | Add a person who lives here. |
| `member.edit` | Change a person's name and details | no | none | Change a person's display name and details, and list the properties a detail may take. |
| `notification.raise` | Tell one person about a report | no | `comm` | Tell one person about a report, and read who can be told. |
| `payment.receive_unpaid_list` | Receive the list of who has not paid | no | `comm` | Names the seat that is sent the list of who has not paid. It carries no action of its own, so holding it opens nothing. |
| `person_raised_items.read` | See the wishes and issues one resident raised | no | `forms` | List the wishes and issues one resident raised. |
| `risk.record` | Record proof on a risk and keep its actions | no | `riskmanager` | Record evidence on a risk, and update and set the status of its actions. |
| `tag.manage` | Create, rename and delete a tag | no | none | Create, rename and delete a tag of the Instance. Giving a tag to a group and taking one off belong to `group.manage`, not to this permission. The hub's guard for both (ISS-216) reads these codes. |

`permission_doc_test.go` keeps this page and the catalogue in step. It reads every table on this page whose first column is headed `Code` or `Permission` and takes the code in backticks that opens each row. It fails naming each catalogue permission that has no such row, and each such row whose code is not in the catalogue. A table headed anything else is not read, so a table of questions or objects may open a row with a backticked word. Setting `ROLE_PERMISSIONS_DOC` to a path runs the test on another copy of the page.

## Where the field permission names are built and shown (ISS-205, D66)

A record gives its field permissions as `models.Permission` values, each with a derived `code` and a `name`. One function builds every name, and everything else only carries or shows it.

- Built: `recordPermissionName(item, action, field)` in `record_permissions.go`, called once by `RecordPermissionsOf` in the same file. The code beside it comes from `RecordPermissionCode` in the same file.
- Tested: `record_permissions_test.go`. `TestRecordPermissions_TenderNamesEachFieldStarAndDeleteInTheLongWordingBesideItsCode` pins each code beside its name; `TestRecordPermissions_ItemNameFallsBackToFormNameThenName` pins the item in the name; `TestRecordPermissions_EveryFieldLevelPermissionHasAName` pins that none is empty.
- Carried, never built: `role_record_permissions.go` and the scan in `issue_impl.go` read the codes only, so a change of wording moves no role.
- Shown in the hub console, outside this repository: `data/permissionNames.ts` reads `agency_list_permissions` into a code to name map, and `pages/spec/rolesDetail.tsx` writes the name with the code beside it (ISS-148, ISS-160). The console builds no wording of its own for a record permission.

The wording (D66): a field permission reads `Add the closing date of a tender`, `Edit the closing date of a tender` or `See the closing date of a tender`; a star reads `Add every field of a tender`; delete reads `Delete a tender`. The code is the permission's own `code`, shown beside the name. The full rules are in [workflow-records.md](workflow-records.md).

## Rules, decisions and questions (ISS-267)

Two permissions carry the twenty `taskmanager` actions that read and write rules, decisions and questions, so a role is no longer refused them unless it is owner or operator. They are read from `mwanachama-backend-taskmanager/taskmanager.operations.json`.

Read, held by `rule.read`: `taskmanager.rule.read`, `list`, `issue_list`, `list_for_issue`, `file_list`, `produced_list` and `overlap_list`, and `taskmanager.backlog.rules_list`, `decisions_list` and `questions_list`.

Write, held by `rule.edit`: `taskmanager.rule.create`, `update`, `set_status`, `supersede`, `answer`, `link`, `unlink`, `file_add` and `file_remove`, and `taskmanager.backlog_tag.add`.

The split follows the catalogue's own: a list or a read of one record is a read; a call that creates, changes, links or removes something is a write. `rule.edit` carries no read, as `task.edit` carries none of the repeating-task reads, so a role that posts rules is given both ticks. `taskmanager.backlog_tag.add` sits with the writes because it changes a backlog's reach, but not in `backlog.manage`, which also deletes backlogs and steps and would give an agent that files rules far more than it needs.

The host's guard is a separate matter: the hub's `backlogReadActions` and `backlogWriteActions` in `group_role_access.go` (hub repo) decide there whether these actions join those two lists. This repository only declares the vocabulary.

Tested in `models/permission_catalog_rules_test.go` (each action is carried by its permission, the permissions are unlocked and of module `taskmanager`, a role holding both may call every action, a role without them or with only one of them is refused the actions of the other) and `permission_rule_roles_test.go` (a role may declare both and publish, and the permission list serves them).
