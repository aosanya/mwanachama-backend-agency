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
- `mcp/module_objectives_text_security_test.go`'s
  `TestAG66_OpenHole_ModuleObjectivesToolTextForgesLinesFromStoredTitles`
  (AG66, P2) — left failing on purpose by the 2026-10-05 security sweep
  (eighty-eighth run): the hand-written `agency_list_module_objectives`
  writes an Objective title, its Goal's title and the link's rationale into
  its text unquoted, so one stored link carrying newlines lists as several
  objective lines and a bare `SYSTEM:` line. It turns green once those
  values are quoted the way `tools.go`'s `line` quotes every generated
  tool's labels.
- `routes/publication_list_cost_security_test.go`'s
  `TestAG67_OpenHole_PublicationListingCarriesEveryWholeDocument` (AG67,
  P2) — left failing on purpose by the 2026-10-05 security sweep (ninetieth
  run): `GET /publications` (and `agency_list_publications`) answers every
  publication with its whole frozen document and no page, so the answer
  grows with the square of the publish count — 40 publications of
  200,000-byte Values answered 164,106,934 bytes. It turns green once a
  listing names each publication without its document, leaving
  `get_publication` as the document read.
- `objectivemodule_race_test.go`'s
  `TestAG61_OpenBug_TwoObjectiveModulesHoldTheSameObjectiveAndModule` (AG61,
  bug), `module_code_race_test.go`'s
  `TestAG62_OpenBug_TwoModulesShareTheSameCodeInOneDraft` (AG62, bug) and
  `valueattachment_race_test.go`'s
  `TestAG63_OpenBug_TwoValueAttachmentsLinkTheSameValueAndTarget` (AG63,
  bug) — left failing on purpose by the 2026-10-02 integration-test sweep,
  widening AG52/AG53's shape (a pre-check-then-insert duplicate rule with no
  backing unique index) to every other `ErrDuplicate*` rule in the package:
  none of `CreateObjectiveModule`, `CreateModule` or
  `CreateValueAttachment`/`CreateRuleAttachment` has a composite unique
  index behind its own duplicate check, so a direct insert placed where a
  concurrent winner would land it slips past the check undetected. All
  three turn green with the same fix shape as AG53 — a declared composite
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

## What this package does not cover

This library has no server and no session of its own (see CLAUDE.md's "no
auth model" note) — reachability, cross-Instance scope, and every other
caller-identity concern are `mwanachama-wakala-api`'s to test, against its
own real router. That repo's own `documentation/4. qa/` and
`cmd/server/selftest/security/` are where those checks live; this folder is
about correctness of the domain logic alone.
