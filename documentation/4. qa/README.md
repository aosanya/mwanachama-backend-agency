# QA

Test coverage for `mwanachama-backend-agency`. Tests live beside the code
they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
```

`go test ./...` is green apart from the deliberate red listed under Red by
design. As of 2026-10-03 (confirmed by the integration sweep with `grep -rc
"^func Test" *.go mcp/*.go routes/*.go`) there were
141 test functions in the root package (across 32 `*_test.go` files:
`agency.go`'s own lifecycle, `drafts.go`'s fork/promote, RACI, Value/Rule
attachments, the Issue scan, publication/document, `filters.go`'s per-module
filters, `security_wk17_test.go`), 23 in `mcp/` (10 files), 64 in `routes/`
(24 files).

## What is guarded, and how

- **No live row to protect** (AGD-006) — `ErrCannotEditLiveRACI` and
  `PromoteDraft` are gone; every row belongs to a Draft and is edited there.
  The one immutability left is `ErrCannotEditLive`: an Agency-wide
  Value/Rule (attached to nothing) refuses `Update`/`Delete`.
- **Attachments resolve by `Code`, not row id**, so a Value/Rule attached
  in one Draft still resolves in a Draft forked from it, where every row
  has a fresh id. Covered by `value_test.go`/`rule_test.go`.
- **`security_wk17_test.go`**: every `Create<Type>` clears a caller-supplied
  `id`/`created_at` rather than honouring it, and a non-empty `draft_id`
  naming an unknown or archived Draft is refused
  (`ErrDraftNotFound`/`ErrDraftNotOpen`) rather than silently attached to a
  Draft the write could never reach.
- **The publish gate** (`AGD-004`): `PublishAgency` runs the Issue scan —
  dangling references, an Objective with no Accountable, a Goal with no
  Objectives, a Workflow with no WorkItems — before hashing, and refuses
  with `ErrPublishValidationFailed` if any check family reports `error`
  severity. `SeedSWELab` publishing clean on the first try is the check
  set's own calibration test.
- **Delete cascades** (`DeleteGoal`/`DeleteObjective`/`DeleteWorkflow`/
  `DeleteRole`) clean up every row a delete would otherwise strand —
  `WorkItem`, `RoleAssignment` — closing the exact orphan shapes the Issue
  scan's dangling-reference checks exist to catch.
- **A row's draft is fixed on update** — pinned for Roles by
  `routes/draft_scope_security_test.go`'s
  `TestRoleUpdateIgnoresABodyDraftID`: a body `draft_id` on `PATCH /roles/{id}`
  is ignored. Goals, Objectives, Workflows and Work Items do not hold it yet —
  see Red by design.
- **Every draft door refuses a draft that does not exist, and a draft's own
  id, status and clock are the server's** — pinned by
  `routes/draft_doors_security_test.go`: publish, fork, update, archive and a
  draft-scoped create each answer `404` for an unknown draft id, and a body
  `id`/`status`/`created_at` on `POST /drafts` or `PATCH /drafts/{id}` is
  ignored. The issue scan is the one door that does not hold yet — see Red by
  design (AG50).
- **An archived draft's own description is closed, and a body cannot reopen
  it** — pinned by `routes/archived_draft_describe_security_test.go`:
  `PATCH /drafts/{id}` on an archived draft answers `409` for a new
  description, for `{"status":"open"}` and for `{}`, and the draft reads back
  archived with the description it was archived with. Mutation-checked:
  dropping `UpdateDraftDescription`'s open-status guard turns all three red.
  The rows inside an archived draft do not hold yet — see Red by design (AG72).
- **The composite tool never reaches across drafts** — pinned by
  `mcp/composite_draft_scope_security_test.go`: an `agency_design_from_spec`
  `raci` entry naming another draft's Role or Objective by id is refused
  (`role not found` / `objective not found`) and nothing is stored in either
  draft. Mutation-checked against `validateRoleAssignment`'s draft scope.
  The REST half is pinned by `routes/roleassignment_draft_scope_security_test.go`
  (2026-09-30): `POST /role-assignments` and `PATCH /role-assignments/{id}`
  naming another draft's Role or Objective each answer `404`, store nothing,
  and leave the existing assignment as it was — mutation-checked the same way
  (4 of 4 red with the scope dropped). Other references are not held at
  create: `POST /objectives` with another draft's `goal_id` and `POST
  /work-items` with another draft's (or a nonexistent) `workflow_id`/
  `objective_id` answer `201`; the issue scan flags the Work Item as a
  dangling Objective, but has no Objective→Goal check (AG27's open "Create
  still accepts a nonexistent reference" class — correctness, not a boundary).
- **An objective's nested listings answer only the draft they name** — pinned
  by `routes/nested_listing_draft_scope_security_test.go` (2026-10-08):
  with an Accountable assignment and a capability link on an objective in
  draft A, `GET /objectives/{id}/role-assignments` and
  `GET /objectives/{id}/modules` return draft A's one row for `?draft_id=<A>`
  and `[]` for another open draft, for no `draft_id` at all, and for a draft
  that does not exist. Mutation-checked: dropping `scopeQuery` from
  `ListRoleAssignmentsForObjective` turns all three RACI cases red.
- **A publication is the server's, and only its channel moves** — pinned by
  `routes/publication_forged_fields_security_test.go` (2026-09-29,
  twenty-third sweep): `POST /publications` carrying `version: 99`,
  `content_hash`, a `document`, `id` and `created_at` stores version 1, the
  draft's own hash and document, and a server id and clock; `PATCH
  /publications/1` carrying `tag`/`content_hash`/`document` changes only the
  channel; a version or fork tag that does not exist answers 404. Not a
  bounded door: `GET /publications` returns every publication's whole
  document and drops `limit` — 200 publications answered 4,960,632 bytes in
  127 ms with or without `?limit=1` (the uncapped-list class tracked as
  wakala-api's WK36).
- **A domain that widens an enum opens no gate** — pinned by
  `widened_vocabulary_security_test.go` (2026-10-02, fifty-ninth security
  sweep). Shared's S37 lets a domain add `values` to a field the module
  declares; this loads `agency.agency.json` with draft `status` widened by
  `locked`, RACI `label` by `lead` and value-attachment `target_type` by
  `department`, and asserts each widened value is refused where a gate reads
  it: `CreateRoleAssignment` → `ErrInvalidRACILabel`, `CreateValueAttachment`
  → `ErrInvalidTargetType`, and a draft stored in `locked` refuses a create,
  a fork and an archive with `ErrDraftNotOpen`. Every gate is an allow-list
  (`== DraftStatusOpen`, `RACILabel.Valid`, `targetRole`'s `switch`).
  Mutation-checked: `validateOpenDraftID` rewritten as `== archived` turns it
  red.
- **An MCP create refuses a server-owned field in any spelling** — pinned by
  `mcp/case_variant_args_security_test.go` (2026-10-04, eighty-third
  security sweep). `encoding/json` matches struct fields case-insensitively,
  so a whole-body decode would take `Deleted`, `ID`, `Draft_ID` or `Code` as
  if they were the lower-case field; the REST half of that is AG57. On the
  MCP plane the go-sdk validates arguments against the declared schema by
  exact name first, and `agency_create_goal` answered every variant with
  `unexpected additional properties ["Deleted"]` (and the same for `DELETED`,
  `ID`, `Created_At`, `CreatedAt`, `Draft_ID`, `DraftID`, `Code`), storing no
  goal in either draft. Not mutation-checked — the refusal lives in the
  go-sdk, not in this module.
- **A fork names a publication by its exact tag** — pinned by
  `routes/publication_fork_tag_security_test.go` (2026-10-04, eighty-fourth
  security sweep). With one publication tagged `release-one`, `POST /drafts`
  forking `forked_from_type: "publication"` from `%`, `release%`,
  `release_one`, `RELEASE-ONE`, `release`, ` release-one`, `x' OR '1'='1`
  and `release-one' --` each answer `404 publication not found`, and a
  miscased `forked_from_type` (`PUBLICATION`, `Publication`,
  `publications`) answers `400`. The lookup is `tag = ?`, bound. Mutation-
  checked: rewriting it as `LOWER(tag) LIKE LOWER(?)` turns the first four
  red, each materializing the whole published document into a new draft.

- **Detaching reaches only its own attachment table** — pinned by
  `routes/attachment_table_confusion_security_test.go` (2026-10-05, eighty-
  seventh security sweep). With one Value and one Rule each attached to the
  same Role, `DELETE /values/{value}/attachments/{rule attachment id}` and
  `DELETE /rules/{rule}/attachments/{value attachment id}` each answer `404`
  and both attachments survive. The parent id in those two paths is declared
  `ignored` in `agency.operations.json` — a Value's door reaches any of the
  Value attachments in the instance, which is the declared shape, not a hole.
  Mutation-checked: pointing `DeleteValueAttachment` at the rule-attachment
  table deletes the Rule's attachment through the Value door (`204`).

- **A capability is switched on only in its own draft** — pinned
  by `routes/module_enabled_scope_security_test.go` (2026-10-12,
  hundred-and-sixty-ninth security sweep, rewritten for the agency-level
  switch by AGD-008), the first pin on the catalog doors
  `GET /modules` and `GET /modules/{id}`. With the bootstrap draft A and a
  fork B (each holding its own `git` row): `PUT /modules/{id}/enabled` on B's
  `git` through draft A answers `404` and switches nothing on; the same call
  on A's `git` answers `200` and `GET /modules?draft_id=<A>&enabled=true`
  lists `git`, while B's listing and `GetModule` on B's `git` still read it
  off. `POST /modules`, `PATCH`/`DELETE /modules/{id}` answer `405` — the
  switch is the only write door. Also probed, not pinned: `enabled=1`/`TRUE`/`yes` → `400`, `code=%` → `[]`.

- **Two Instances on one database never reach each other's rows** — pinned
  by `routes/instance_table_set_isolation_security_test.go` (2026-10-12,
  hundred-and-sixty-sixth security sweep). Two managers built from
  `SpecFor("isoalice")` and `SpecFor("isobob")` over one sqlite file, each
  behind its own route table: through alice's routes, detaching bob's rule
  attachment (under her own rule's path and under his), detaching his value
  attachment, reading, patching and attaching to his rule, and reading and
  deleting his value all answer `404`; bob's rows, text and both attachments
  survive, and bob's own detach of the same attachment then answers `204`.
  The separation is the hashed table set alone — no `instance` column is
  consulted. Mutation-checked: making `SpecForMount` ignore its instance
  argument turns eight of the nine doors red.

- **An update never seats a second Accountable** — pinned by
  `routes/raci_update_accountable_security_test.go` (2026-10-05, ninety-
  fourth security sweep). With one Accountable already on an objective,
  `PATCH /role-assignments/{id}` relabelling a Consulted seat to
  `accountable`, re-pointing another objective's Accountable onto it, or
  doing either while claiming `draft_id`/`id`/`created_at` in the body each
  answer `409`, and the objective keeps exactly its original Accountable.
  The create half was already covered by `TestCreateRoleAssignment_DuplicateAccountable`;
  the concurrent half is AG53. Mutation-checked: skipping
  `validateRoleAssignment` in `UpdateRoleAssignment` turns all four
  assertions red.

- **No draft-scoped create lands outside an open draft** — pinned by
  `routes/create_draft_gate_security_test.go` (2026-10-06, hundred-and-third
  security sweep). `POST` to `/goals`, `/objectives`, `/workflows`,
  `/work-items`, `/roles`, `/role-assignments`, `/values` and `/rules`
  naming an archived draft answers `409`, an unknown one `404`, none at all
  `400`, and listing either scope afterwards returns `[]`. The manager-level
  Goal case was already in `security_wk17_test.go`; this covers every type
  through the route table. Updates are not covered here — moving a row into
  another draft by `PATCH` is AG48, and editing or deleting a row in place
  inside an archived draft (`200`/`204`, where create answers `409`) is
  AG72, red in `routes/archived_draft_edit_security_test.go`. Mutation-checked: letting
  `validateOpenDraftID` pass an archived draft turns all eight archived
  cases red.

- **A later create or a rename never moves the served agency** — pinned by
  `routes/agency_singleton_security_test.go` (2026-10-07, hundred-and-
  fourteenth security sweep). A second `POST /agency` carrying `id`,
  `created_at`, `enabled` and `deleted` keeps none of the body's id or
  clock and never becomes what `GET /agency` serves; `PATCH /agency`
  carrying every setting changes only `name` and `description`, and the
  served row's `id`, `created_at`, `enabled`, `deleted`,
  `inactivity_timeout_seconds`, `default_failure_pipeline_budget` and
  `event_flows` read back unchanged. That the second create is stored at
  all is AG73, red in `routes/second_agency_security_test.go`.
  Mutation-checked: ordering `findAgency` newest-first turns the
  displacement assertion red; copying the whole body onto the stored row
  *and* dropping `UpdateAgency`'s skipped columns turns the settings
  assertion red (either alone does not: the two are independent guards).

- **Event-flow-step reads never reach another draft's steps** — pinned by
  `routes/event_flow_step_scope_security_test.go` (2026-10-06,
  hundred-and-sixth security sweep). A step created in one draft is found by
  `GET /event-flow-steps` and by `GET /event-flow-steps/lookup` under all
  three keys (`handler_code`, `code`, `trigger_topic`+`consumer`) only when
  the request names that draft; naming another draft, an unknown one or none
  at all lists `[]` and looks up `404`. There is no event-flow-step write
  route, so the read pair is the whole surface. Mutation-checked: dropping
  the `draft_id` predicate from `scopeQuery` turns twelve assertions red.

- **Event-flow-step reads match the exact spelling only** — pinned by
  `routes/event_flow_step_exact_match_security_test.go` (2026-10-08,
  hundred-and-twenty-ninth security sweep). Inside the owning draft, the
  lookup under each of its three keys and the listing's `workflow_code`
  filter find nothing for `%`, `_`, a case variant or a trailing space:
  `code=%25` looks up `404`, and `workflow_code=%25` lists `[]`. The scope
  pin above held the draft boundary but not the spelling. Mutation-checked:
  turning the lookup's `code = ?` into `LIKE` and folding the listing's
  `workflow_code` to lower case turns four assertions red.

- **No stored row reaches a tool's name, description or schema** — pinned by
  `mcp/tool_list_stored_text_security_test.go` (security sweep, 2026-10-09,
  OWASP MCP03 tool poisoning). It lists `tools/list` on a fresh session,
  stores a Goal, Role, Workflow, Value and Rule whose text carries a marker
  and an injected instruction through the tools themselves, then lists again
  on a second session: the marker is absent and the listing is byte-identical.
  The control is `registerDeclaredTools` building every tool once from the
  embedded blueprint over `ManagerShape()`, never from an Instance's rows.
  Mutation-checked: a handler that registers a tool described by its own
  arguments turns it red.

- **A tool given `null` for every required argument refuses** — pinned by
  `mcp/tool_null_required_security_test.go` (security sweep, 2026-10-12).
  `dispatch`'s required check tests presence only, so `{"draft_id": null}`
  passes it and binds `""`; every tool with a required argument besides
  `instance_id` (over forty) is called that way and each must answer
  `IsError`. The control is the manager's own refusals —
  `ErrDraftRequired`, the not-found sentinels — not the binder. Not
  mutation-checked: the run's mutation of `draft_impl.go` was refused by the
  environment, and the sweep does not touch product code to get one.

- **No hand-written tool shares a name with a declared one** — pinned by
  `mcp/tool_name_shadowing_security_test.go` (security sweep, 2026-10-12,
  OWASP MCP03/MCP07). The go-sdk's `AddTool` keeps whichever registration of
  a name lands last and drops the other without a word, and `RegisterTools`
  registers the declared tools first and the hand-written ones
  (`agency_design_from_spec`, `agency_list_types`) after. A declared operation later naming one of those
  would be silently replaced, and a hand-written tool registered first would
  be silently replaced by a declared one gated differently. The test lists
  the hand-written tools on their own, asserts none is declared, and asserts
  `RegisterTools` publishes exactly declared + hand-written. Shared's S63
  covers the same collision between two declared operations; this is the
  seam S63 cannot see. Mutation-checked: renaming `agency_list_types` to
  `agency_list_goals` turns both assertions red (one tool fewer published than
  declared plus hand-written).

- **`agency_design_from_spec` plants nothing in a closed, unknown or unnamed
  Draft** — pinned by `mcp/design_spec_closed_draft_security_test.go`
  (security sweep, 2026-10-09). One spec carrying a Role, Goal, Objective,
  Workflow, Work Item and a RACI entry by real id is sent naming an archived
  Draft, `no-such-draft` and no Draft at all. All six entries come back in
  `errors` (`draft is not open`, `draft not found`, `a draft id is
  required`, or an unresolved key), nothing is reported created, and none of
  the three scopes holds a planted row afterwards. The control is
  `validateOpenDraftID`, which every `Create<Type>` the composite calls runs
  first; this pins it through the composite, where AG72's archived-draft
  update/delete gap does not reach. Mutation-checked: letting an archived
  Draft pass `validateOpenDraftID` turns five assertions red.

- **A listing filter is a literal, never SQL** — pinned by
  `mcp/list_filter_sql_metachar_security_test.go` (security sweep,
  2026-10-10, OWASP MCP05 command injection). Two Goals are stored, then
  `agency_list_goals` is called with `' or '1'='1`, `x') or 1=1 --`,
  `x'; delete from goals; --` and two quote variants in each of `code`,
  `title` and `description`: every call lists none, and an unfiltered
  listing afterwards still holds both. The control is `likeFilter` binding
  the escaped value as a parameter; every raw `deleteWhere` clause in the
  package is a constant with bound arguments. Mutation-checked: splicing the
  value into the `LIKE` text turns it red, the first payload listing both
  Goals.

## Red by design

- `provision_test.go`'s `TestAG51_OpenBug_ProvisionReseedsAfterAllDraftsAreArchived`
  (AG51, bug) — left failing on purpose by the 2026-10-01 integration-test
  sweep: `Provision`'s `upToDate` treats "zero open Drafts" as "never
  seeded", so archiving an agency's only Draft (an ordinary, legitimate
  operation under AGD-006) makes the next `Provision` call mint a second,
  spurious "Initial draft" beside the archived one. It turns green once
  `upToDate` uses a signal other than the open-Draft count.
- `routes/role_hint_wildcard_test.go`'s
  `TestAG56_OpenBug_RoleHintTreatsUnderscoreAndPercentAsWildcards` (AG56,
  bug) — left failing on purpose by the 2026-10-01 integration-test sweep:
  `GET /roles/{id}`'s "did you mean" hint builds its `LIKE` pattern from the
  unknown id with no escaping, so a bare `_` or `%` id discloses every
  Role's code, name and id in the Instance. It turns green once
  `roleNotFoundError` escapes both characters the way CAT22/PM7 need to for
  their own unescaped filters.
- `goal_delete_cross_draft_test.go`'s
  `TestAG75_OpenBug_DeleteGoalCascadesIntoAnotherDraftsWorkItem` (AG75,
  bug) — left failing on purpose by the 2026-10-10 integration-test sweep:
  `DeleteGoal`'s Work Item cascade matches `objective_id` with no
  `draft_id` predicate, so deleting a Goal in one Draft can delete a Work
  Item that lives in an unrelated Draft (`DeleteObjective`'s and
  `DeleteWorkflow`'s own Work Item cascades share the identical shape,
  confirmed by probe, not just by reading). It turns green once each
  cascade is scoped to the owning Draft, or `CreateWorkItem` refuses a
  cross-draft `objective_id`/`workflow_id` so the dangerous state can
  never exist.
- `routes/draft_scope_security_test.go`'s
  `TestAG48_OpenHole_UpdateMovesRowIntoAnotherDraft` (AG48, P1) — left
  failing on purpose by the 2026-09-28 security sweep: `PATCH` on a Goal,
  Objective, Workflow or Work Item moves it into whatever draft the body
  names, archived ones included. It turns green when the four `Update<Type>`
  methods pin the stored draft, as `UpdateRole` does.
- `routes/issue_draft_security_test.go`'s
  `TestAG50_OpenHole_IssueScanWritesUnderADraftThatDoesNotExist` (AG50, P2) —
  left failing on purpose by the 2026-09-29 security sweep:
  `GET /drafts/{id}/issues` answers `200` for an id no draft carries and stores
  the scan's findings under it, where `GET /drafts/{id}` and every create
  answer `404`. It turns green when `ListIssues` refuses an unknown draft
  before `rescanIssues` writes.
- `publication_race_security_test.go`'s
  `TestAG52_OpenHole_TwoPublicationsHoldOneVersionAndOneChannel` (AG52, P2) —
  left failing on purpose by the 2026-09-30 security sweep: concurrent
  `PublishAgency` calls on Postgres each take `max(version)+1` and each hold
  the channel (through `mwanachama-wakala-api`'s mount, three `201`s at
  version 3 and two publications left on `production`). SQLite serialises the
  writers, so the test writes the losing writer's row directly and turns green
  once the store refuses a second publication at one version.
- `roleassignment_race_security_test.go`'s
  `TestAG53_OpenHole_TwoAccountablesHoldOneObjective` (AG53, P2) — left
  failing on purpose by the 2026-09-30 security sweep (thirty-eighth run):
  eight concurrent `CreateRoleAssignment(label: accountable)` calls on one
  Objective over Postgres stored two, three or four Accountables in three of
  twenty-five rounds, the Issue scan reported nothing and the draft still
  publishes. SQLite serialises the writers, so the test writes the racing
  row directly and turns green once the store refuses it or the publish
  gate refuses an Objective with more than one Accountable.
- `routes/draft_fork_cost_security_test.go`'s
  `TestAG54_OpenHole_OneForkRequestCopiesEveryRowWithoutLimit` (AG54, P2) —
  left failing on purpose by the 2026-09-30 security sweep (fortieth run):
  a ~100-byte `POST /drafts` forking a draft copies every row it holds, and
  nothing caps how many drafts an agency keeps — fifty fork-then-archive
  cycles of a 200-goal draft are all accepted. It turns green once a fork
  past a per-agency cap answers a 4xx.
- `routes/create_deleted_security_test.go`'s
  `TestAG57_OpenHole_CreateStoresABodySetDeletedFlag` (AG57, P2) — left
  failing on purpose by the 2026-10-01 security sweep (forty-eighth run):
  `POST /goals`, `/roles` and `/workflows` store a body `"deleted": true`
  while minting their own id, code and created_at over the body's. It turns
  green once `Create<Type>` pins `Deleted` to false.
- `mcp/design_spec_cost_security_test.go`'s
  `TestAG58_OpenHole_OneDesignFromSpecCallAppliesUnboundedEntities` (AG58,
  P2) — left failing on purpose by the 2026-10-01 security sweep
  (forty-ninth run): one `agency_design_from_spec` call carrying 20,000
  goals was applied whole in 29.6 s through `mwanachama-wakala-api`'s
  `/mcp`, bounded only by that host's 1 MiB body cap. It turns green once
  the tool refuses a spec past a per-call entity cap.
- `mcp/tool_text_reference_security_test.go`'s
  `TestAG59_OpenHole_ReferenceFieldForgesLinesInAnotherCallersToolText`
  (AG59, P2) — left failing on purpose by the 2026-10-01 security sweep
  (fifty-first run): a Work Item's `objective_id` carrying newlines is
  written verbatim into `agency_list_work_items`' text, forging a second
  Work Item line and a free-text instruction in the reading model's
  context. It turns green once every reference in a tool line is quoted
  the way labels are, or the dangling `objective_id` is refused. The
  labels' quoting is pinned green beside it in
  `mcp/tool_text_label_security_test.go`.
- `routes/publication_list_cost_security_test.go`'s
  `TestAG67_OpenHole_PublicationListingCarriesEveryWholeDocument` (AG67,
  P2) — left failing on purpose by the 2026-10-05 security sweep (ninetieth
  run): `GET /publications` (and `agency_list_publications`) answers every
  publication with its whole frozen document and no page, so the answer
  grows with the square of the publish count — 40 publications of
  200,000-byte Values answered 164,106,934 bytes. It turns green once a
  listing names each publication without its document, leaving
  `get_publication` as the document read.
- `mcp/publish_refusal_text_security_test.go`'s
  `TestAG68_OpenHole_PublishRefusalForgesLinesFromAStoredReference` (AG68,
  P2) — left failing on purpose by the 2026-10-05 security sweep
  (ninety-second run): the same dangling `objective_id` AG59 plants reaches
  `agency_publish_agency`'s refusal unquoted, because `issue_impl.go`
  formats it with `%s` and the refusal never passes through `tools.go`'s
  `line` — one refusal reads as three lines, a forged work item and a bare
  `SYSTEM:` line. It turns green once issue messages quote every
  caller-written reference, or the dangling `objective_id` is refused.
- `mcp/fork_tag_text_security_test.go`'s
  `TestAG69_OpenHole_PublicationTagForgesLinesInEveryDraftListingThroughAFork`
  (AG69, P2) — left failing on purpose by the 2026-10-06 security sweep
  (hundred-and-eighth run): a publication tag carrying newlines is stored,
  a fork names it, and the fork's `forked from id` reaches every caller's
  `agency_list_drafts` text unquoted — three drafts read as five lines, a
  forged draft and a bare `SYSTEM:` line. It turns green once `tools.go`'s
  `line` quotes references, or a tag with control characters is refused.
- `mcp/fork_untyped_source_security_test.go`'s
  `TestAG70_OpenHole_UntypedForkSourceIsStoredUncheckedAndForgesDraftListings`
  (AG70, P2) — left failing on purpose by the 2026-10-06 security sweep
  (hundred-and-ninth run): `agency_create_draft` with a `forked_from_id`
  and no `forked_from_type` stores the id without looking it up, so one
  call puts a forged draft and a bare `SYSTEM:` line in every caller's
  `agency_list_drafts` text. It turns green once `CreateDraft` refuses (or
  drops) a fork source that names no type — quoting in `line` alone does
  not clear it.
- `mcp/publication_channel_text_security_test.go`'s
  `TestAG71_OpenHole_PublicationChannelForgesLinesInEveryPublicationListing`
  (AG71, P2) — left failing on purpose by the 2026-10-06 security sweep
  (hundred-and-tenth run): a publication channel carrying newlines is
  stored as given and reaches every caller's `agency_list_publications`
  text unquoted as a fact — one publication reads as three lines, a forged
  "version 99 on production" and a bare `SYSTEM:` line. It turns green once
  `tools.go`'s `line` quotes facts as well as labels, or a channel is held
  to a name pattern.
- `module_code_race_test.go`'s
  `TestAG62_OpenBug_TwoModulesShareTheSameCodeInOneDraft` (AG62, bug) and
  `valueattachment_race_test.go`'s
  `TestAG63_OpenBug_TwoValueAttachmentsLinkTheSameValueAndTarget` (AG63,
  bug) — left failing on purpose by the 2026-10-02 integration-test sweep,
  widening AG52/AG53's shape (a pre-check-then-insert duplicate rule with no
  backing unique index) to every other `ErrDuplicate*` rule in the package:
  neither `CreateModule` nor
  `CreateValueAttachment`/`CreateRuleAttachment` has a composite unique
  index behind its own duplicate check, so a direct insert placed where a
  concurrent winner would land it slips past the check undetected. Both
  turn green with the same fix shape as AG53 — a declared composite
  unique index per object, recognised and recovered from on the insert.
- `valueattachment_dangling_source_test.go`'s
  `TestAG64_OpenBug_DeletingAnAttachedValueOrphansItsAttachment` (AG64,
  bug) — left failing on purpose by the 2026-10-02 integration-test sweep,
  found widening AG29's publish→fork round-trip: `DeleteValue`/`DeleteRule`
  allow deleting a Value/Rule precisely because it is attached, but leave
  the `value_attachment`/`rule_attachment` row behind pointing at a
  `value_code`/`rule_code` nothing holds any more — `checkAttachments`
  (AG26's dangling-reference scan) validates only the target side of an
  attachment, never the source, so the scan reports 0 issues and
  `PublishAgency` succeeds over the orphan. It turns green once
  `DeleteValue`/`DeleteRule` cascade-delete their own attachments, the way
  `DeleteRole`/`DeleteGoal`/`DeleteObjective` already do (AG27).
- `eventflowstep_handlercode_ambiguous_test.go`'s
  `TestAG65_OpenBug_LookupFlowStepByHandlerCodePicksAnArbitraryStepWhenSeveralShareIt`
  (AG65, bug) — left failing on purpose by the 2026-10-03 integration-test
  sweep, found walking `agency.operations.json`'s `lookup_event_flow_step`
  against its own blueprint (catalogue entry 1): `handler_code`'s own
  index is declared non-unique and documented as "which **steps** a given
  handler runs," yet `LookupFlowStep`'s `HandlerCode` branch resolves
  through `specstore.Store.Take`'s `Limit(1)`, answering whichever step
  was inserted first with no signal that a second one also matches. It
  turns green once `LookupFlowStep` counts matches on the `HandlerCode`
  (and `TriggerTopic`+`Consumer`) branches and refuses an ambiguous one
  rather than resolving it silently.
- `routes/archived_draft_edit_security_test.go`'s
  `TestAG72_OpenHole_RowsInAnArchivedDraftStillTakeUpdatesAndDeletes` (AG72,
  P2) — left failing on purpose by the 2026-10-07 security sweep: the open-draft
  fence `validateOpenDraftID` runs in every `Create<Type>` but no
  `Update<Type>`/`Delete<Type>`, so a role, work item, goal or RACI link inside
  an archived draft still takes a rename or a delete (`200`/`204`) where the
  same draft's create already answers `409`. It turns green once
  `Update<Type>`/`Delete<Type>` run the same open-draft check against the
  row's existing `draft_id` before writing.
- `routes/second_agency_security_test.go`'s
  `TestAG73_OpenHole_ASecondCreateAgencyStoresAnOrganizationNothingCanReadOrDelete`
  (AG73, P2) — left failing on purpose by the 2026-10-07 security sweep:
  `POST /agency` on an instance that already has its one agency answers `201`
  and stores another organization that no route lists, serves or deletes —
  `findAgency` keeps serving the oldest row, so the extra rows are invisible,
  permanent, and `GET /agency` slows as they pile up (0.83 ms → 52.9 ms over
  5,000 of them). It turns green once `CreateAgency` refuses with a new
  `ErrAgencyExists` when `findAgency` already finds one.
- `routes/agency_wide_attach_detach_security_test.go`'s
  `TestAG74_OpenHole_AttachEditDetachRewritesAnAgencyWideValue` (AG74, P2) —
  left failing on purpose by the 2026-10-08 security sweep: an agency-wide
  Value or Rule, declared permanently immutable by `update_value`/
  `delete_value`/`update_rule`/`delete_rule`, can still be rewritten or
  deleted by attaching it to any Role first and detaching it afterwards,
  because `ErrCannotEditLive` reads only whether the row holds an attachment
  at the moment of the write and attach/detach is itself unrestricted
  (AGD-003). The same sweep's later evidence shows the detour does not even
  need the target row: because `hasAnyAttachment` matches by Code across
  every draft and a fork from a publication carries each agency-wide Value's
  Code verbatim, attaching in a second draft forked from the same publication
  unlocks the first — reproduced for both the Value and the Rule half. It
  turns green once `CreateValueAttachment`/`CreateRuleAttachment` refuse an
  unattached Value/Rule's first attach, or agency-wide is recorded as a
  stored property set at create rather than derived from the current
  attachment count.

- `routes/goal_delete_bound_params_security_test.go`'s
  `TestAG76_OpenHole_AGoalWithMoreObjectivesThanOneStatementCanBindCanNeverBeDeleted`
  (AG76, P3, ≈6.5 s) — left failing on purpose by the 2026-10-10 security
  sweep: `DeleteGoal` binds every Objective id under the Goal into one
  `objective_id in ?`, so a Goal holding 32,767 Objectives answers
  `DELETE /goals/{id}` with `500` on every attempt and survives (sqlite's
  32,766-parameter ceiling; 65,535 under pgx). It turns green once the
  cascade deletes by subquery or in bounded batches.

## What this package does not cover

This library has no server and no session of its own (see CLAUDE.md's "no
auth model" note) — reachability, which caller may reach which Instance
(membership), and every other caller-identity concern are
`mwanachama-wakala-api`'s to test, against its own real router. The one
half of cross-Instance scope this package does own — that two Instances'
table sets on one database never reach each other — is pinned above. That repo's own `documentation/4. qa/` and
`cmd/server/selftest/security/` are where those checks live; this folder is
about correctness of the domain logic alone.
