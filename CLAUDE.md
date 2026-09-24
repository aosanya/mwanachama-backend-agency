# CLAUDE.md

Guidance for Claude Code (claude.ai/code) working in this repository.

## Project: mwanachama-backend-agency

A standalone Go library, sibling to
[`mwanachama-backend-actor`](../mwanachama-backend-actor), modeling an
Agency: departments under one Agency, Goals/Workflows/WorkItems, and a RACI
matrix (Responsible/Accountable/Consulted/Informed) assigning Roles to
WorkItems. Module path `github.com/aosanya/mwanachama-backend-agency` (not
yet scaffolded). Ported from a standalone gRPC microservice —
see [documentation/1. requirements/requirements.md](documentation/1.%20requirements/requirements.md)
for the full decision record from the 2026-09-09 scoping session.

Scoped 2026-09-09. No code yet — this session produced the requirements
record and the initial task board; the actual port from the standalone service,
the GORM conversion, and the new RACI/Role work are follow-up work (see
[documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md)).

## Porting notes

- **AG3 landed 2026-09-09: `models/`, `gormstore/`, root `doc.go`/
  `tables.go` for Agency/Draft/Goal/Workflow/WorkItem.** Package name is
  `mwanachamaagency`. No `AgencyID` field exists on any of these five
  types — confirmed with the user this is a single-agency deployment for
  now, so the todo's "drop AgencyID-style single-tenancy in favor of AG5's
  department scoping" is satisfied by simply not adding the field yet, not
  by inventing a placeholder. Revisit when AG5's department type lands and
  a real scoping column is needed.
- `Draft` ports the source's `AgencyDraft` as a header/pointer type
  only (`ID`, `Description`, `ForkedFromID`/`ForkedFromType`, `Status`,
  timestamps) — it does not carry a copy of the forked sub-graph.
  `AgencySnapshot`, `AgencyPublication`, and the `Draft*` sub-entity shadow
  types (`DraftGoal`, `DraftWorkflow`, etc. — which don't exist as Go
  structs in the source either, only as `schema.go` entitygraph property
  lists) are AG4's job, produced by the edit-lifecycle behavior it ports.
- `ConfiguredRole` was **not** ported — AG6 is building a proper `Role`
  model for RACI, and porting `ConfiguredRole` now would just be thrown
  away.
- `WorkItem` is owned by exactly one `Workflow` (`WorkflowID`) and
  optionally tagged to a `Goal` (`GoalID`, `""` = untagged) — the source's
  own schema notes WorkItem is Workflow-owned but goal-taggable, not
  strictly single-parent.
- `Agency.EventFlows`/`Workflow.EventFlows` stay opaque JSON strings, ported
  as-is — no typed parsing yet (AG4's `EventFlowStep` lookup is what reads
  them).
- Timestamps are hand-formatted RFC3339-nano strings (`models/time.go`,
  ported verbatim from actor's), not GORM-managed `time.Time` columns —
  same reasoning as actor's own `models/time.go` doc comment.
- **AG4 landed 2026-09-09: the first behavior/manager layer.** `AgencyManager`
  interface + `agencyManager` struct live in root `agency.go` (named after
  the domain, like the source's own `agency.go` — not copying actor's
  `UserManager`/`user.go` naming). Explicit user instruction: GORM only, no
  ArangoDB/entitygraph concepts, keep it simple.
- **Draft-scoped rows reuse the live tables via a nullable `DraftID`
  column** (`""` = live) on `Goal`/`Workflow`/`WorkItem`/`EventFlowStep`,
  not separate `DraftGoal`/etc. types — one CRUD method per type serves
  both scopes (`List<Type>(ctx, draftID)`). `PromoteDraft` deletes the old
  live rows and flips the draft's own rows' `DraftID` to `""` — no
  copy-and-swap needed at promote time, unlike source (which mints new IDs
  on both fork *and* promote; this port only mints once, at fork time).
- **`PublishAgency` requires `Draft.Status == promoted`** when given a
  draftID (`ErrDraftNotPromoted` otherwise), and hashes/versions whatever
  is currently live rather than redoing the copy-and-swap. This closes a
  real source bug: there, `PublishAgency` never checks draft status and
  always re-copies, so calling it after `PromoteDraft` silently
  double-promotes. `AgencyPublicationStatus` was folded into
  `AgencyPublication.Status` — a GORM row is already mutable, unlike an
  immutable entitygraph entity, so the second mutable-status entity source
  needed has no reason to exist here.
- `models.DraftStatus.CanTransitionTo` (added in AG3, unused until now) is
  actually called by `PromoteDraft`/`ArchiveDraft`, closing a source gap
  where the same rule was declared but re-implemented ad hoc at every call
  site instead of used.
- `CreateDraft`'s fork-copy remaps both `WorkItem.WorkflowID` and `.GoalID`
  via old→new id maps built while copying Goals/Workflows (source only
  remapped the Workflow ref — a correctness fix, not scope creep).
  `EventFlowStep` copies need no remap since they join `Workflow` by the
  `WorkflowCode` business key, not a row id.
- Content hash (`AgencyPublication.ContentHash`) covers live
  Goal/Workflow/WorkItem/EventFlowStep + Agency name/mission/vision, sorted
  JSON-per-row joined with `"\n"`, SHA-256, hex. Source's hash omitted
  `EventFlowStep` entirely and included `WorkPlan`, which isn't part of
  this repo yet (AG8) — extend the hash inputs when AG6/AG8 land their own
  content.
- `UpdateGoal`/`UpdateWorkflow`/`UpdateWorkItem` write an explicit
  `map[string]any` of columns rather than `Updates(&row)` — GORM's
  struct-based `Updates` silently skips zero-value fields, so a plain
  struct update would no-op on clearing a field to `""`/`0`/`false`.
- Minimal CRUD only (`Create`/`Get`/`List`/`Update`, no `Delete`) for
  Agency/Goal/Workflow/WorkItem — necessary supporting infrastructure for
  the draft lifecycle (nothing else on the board builds it), not itself a
  line item. `ConfiguredRole`/`WorkPlan`/`ContextSource` stay excluded
  (other tasks' scope); `Instruction`/`Deliverable`/`DeliverableResult`/
  `ContentRef` are never ported at all — source's own `drafts.go` never
  copies that family either despite the schema declaring them
  Draft-copyable, so nothing is lost leaving them out. The
  `Workflow.EventFlows`-JSON-to-`EventFlowStep` import pipeline is separate
  machinery in source (`import_event_flow_steps.go`), not part of
  `drafts.go`/`event_flow_steps.go` — out of scope; `CreateEventFlowStep`
  exists for direct population instead.
- **Resolved, not carried forward from AG3's note:** a peer session
  grepped the source's `raci.go` directly and confirmed it's genuinely
  all WorkPlan/`ContextSource` CRUD with zero RACI logic (every
  RACI-labeled hit is in generated proto code) — matching requirements.md's
  original read. AG8 does not need a closer look at it.
- **AG5 landed 2026-09-09: `Department`/`Level`, the org chart.** Net
  new — the source has no department concept at all. Modeled on
  actor's Group/Hierarchy but deliberately smaller: no `Hierarchy`
  container (this repo has exactly one implicit org chart, not actor's
  multi-hierarchy-per-gateway-mount case), no `Attributes`/`Discoverable`/
  `NodeType`/`AnchorLevelOverrideID` on `Department` (membership/directory
  extension points with no analog here), no `IsDefaultAnchor` on `Level`
  (a member-registration-default concept). `MoveDepartment`/`DeleteLevel`
  are direct ports of `MoveGroup`/`DeleteLevel`'s bodies — same Go-loop
  cycle/subtree check (not a recursive CTE), same targeted
  `Updates(map[string]any{...})` re-parent write.
- **`ErrRootCannotMove` is kept verbatim, confirmed with the user** even
  though it means a top-level department (e.g. "software" or "mechanical"
  in requirements.md's own example) can never be moved once created at the
  top level, and nothing enforces a single designated root the way actor's
  one-organizational-root assumption implicitly does. Fidelity to actor's
  behavior was chosen over allowing top-level reorganization.
- `RenameDepartment(ctx, id, name)` takes a plain name, not an
  `EditGroup`-style multi-field DTO with "empty clears every call"
  semantics — `Department` only has the one editable field, and (unlike
  `EditGroup`) an empty name is rejected (`ErrInvalidDepartment`); there's
  no legitimate "clear the name" case the way there is for `NodeType`.
- `DeleteLevel` only ports the "refuse while worn by Departments" half of
  actor's two-part check. The other half (`RoleKind.LevelID`) has no
  analog yet — **if AG6's `Role` ever gets scoped to a `Level`, `DeleteLevel`
  needs a second guard added then**, the same way actor's does for
  `RoleKind`.
- No `AgencyID`/`DraftID` on `Department`/`Level` (matches AG3's
  single-agency posture; org-chart edits are always-live, not
  draft-scoped like Goal/Workflow/WorkItem), and no `DepartmentID` wired
  onto `Goal`/`Workflow`/`WorkItem` — both explicitly out of AG5's scope,
  left for whichever future task (likely AG6) actually needs
  department-scoped RACI.
- **AG8 landed 2026-09-09: `WorkPlan`/`ContextSource`, the event-topic
  matching/dispatch logic `raci.go` actually contains.** Files are named
  after what they model (`models/workplan.go`, `contextsource.go`,
  `gormstore/workplan.go`, `contextsource.go`, root `workplan_impl.go`,
  `contextsource_impl.go`), not after the source filename. No `AgencyID`
  (matches AG3's single-agency posture) and deliberately no `DraftID` —
  requirements.md's port list only ever mentions WorkPlan/ContextSource in
  the raci.go paragraph, never as part of AG4's draft/publication scope, so
  they stay outside the draft lifecycle rather than being spec-creeped in.
  `CreateWorkPlan`/`UpdateWorkPlan` take `models.WorkPlan` directly instead
  of porting source's separate `CreateWorkPlanRequest`/
  `UpdateWorkPlanRequest` DTOs — matches Goal/Workflow/WorkItem's own
  full-struct-replace-plus-explicit-column-map convention, and as a side
  effect fixes a real source bug: `CreateWorkPlanRequest` had no `Code`
  field, so a WorkPlan's `Code` (its own doc says "the stable slug
  `OnFailurePipeline` references") could never be set through the source's
  actual CRUD despite `entityToWorkPlan` reading it back on every fetch.
  `ContextSource`'s three polymorphic kinds (Git/Comm/Work) — three
  heterogeneous entitygraph entity types in source — flatten onto one row
  with kind-prefixed columns (`GitSignals`, `CommLookbackDays`,
  `WorkIncludeHistory`, ...), since GORM has no polymorphic-entity
  equivalent. **`RemoveContextSource(workPlanID, sourceID)` now actually
  checks `sourceID` belongs to `workPlanID`** (new
  `ErrContextSourceNotFound` on mismatch, covered by
  `TestRemoveContextSource_WrongWorkPlan`) — the source method accepted a
  `workPlanID` parameter it never used, so any caller who knew a bare
  `sourceID` could delete it while claiming any `workPlanID` at all.
  `MatchWorkPlans` ported behavior-for-behavior otherwise: enabled +
  `TriggerTopic` regex match + optional `PayloadCondition` regex match,
  results sorted by `Ordinality`. Landed and verified building/testing
  alongside AG5 (Department/Level) — both additive to the same shared
  `gormstore/tables.go`/`errors.go`/root `agency.go` interface, no actual
  overlap.

- **AG6 landed 2026-09-09: `Role`/`RoleAssignment`, the RACI matrix.** Net
  new. `Role` stays unscoped — no `DepartmentID`, despite AG5 deferring
  exactly this question here and AG7's task text framing personas as
  "Roles under one software-type department." Revisit later if something
  concrete needs it.
- **"A live role cannot be edited. A draft is a draft of everything
  including the role."** `Role`/`RoleAssignment` carry the same nullable
  `DraftID` (`""` = live) as Goal/Workflow/WorkItem/EventFlowStep, and
  `UpdateRole`/`DeleteRole`/`UpdateRoleAssignment`/`DeleteRoleAssignment`
  refuse with `ErrCannotEditLiveRACI` on a live row — editing means fork a
  Draft, edit the draft-scoped copy, promote. This surfaced a real gap
  while designing AG6: `PromoteDraft` replaces *every* live WorkItem row
  on every promotion (minting new ids even for untouched ones), so a
  `RoleAssignment` referencing a `WorkItemID` would otherwise go stale
  after any future promotion — confirmed to matter in practice by a peer
  session working `mwanachama-backend-api-kazi`'s K6, which needs RACI to
  find who to notify for a WorkItem.
- **This reopens and extends AG4's already-shipped `CreateDraft`/
  `PromoteDraft`/`liveContentHash`**, not just two new tables in
  isolation: `CreateDraft`'s WorkItem-copy loop gained a `workItemIDMap`
  (nothing needed one before Role/RoleAssignment existed), plus two new
  copy loops (Role, then RoleAssignment remapping `RoleID`/`WorkItemID`
  via the id maps built while copying). `PromoteDraft`'s delete-old-live
  and flip-to-live blocks each gained `Roles`/`RoleAssignments` alongside
  the existing four tables — same shape, no new pattern. `liveContentHash`
  gained live Role/RoleAssignment records, so a RACI-only edit now
  requires republishing rather than silently taking effect.
- `CreateRoleAssignment`/`UpdateRoleAssignment` validate `Label` against
  the four RACI values, that the referenced `Role`/`WorkItem` exist *in
  the same scope* as the assignment (not just that the id exists
  somewhere — a draft-scoped Role cannot be assigned to a live WorkItem or
  vice versa), and the "at most one Accountable per WorkItem" rule within
  that scope. `ListRoleAssignmentsForWorkItem` is the lookup K6 actually
  needs ("who is assigned to this WorkItem, with what RACI label").

- **AG7 landed 2026-09-09: `SeedSWELab`, an exported seed function** (not
  a script — this repo has no CLI/deployment yet). Reaffirms, rather than
  reopens, AG6's unscoped-`Role` decision: creates a real "Software"
  `Department` for context, but `Role` still doesn't reference it, even
  though this task's own text frames personas as "Roles under one
  software-type department."

- **AG9 landed 2026-09-09: `routes/`, matching actor's `routes/` package
  exactly.** Pure `net/http`, Go 1.22+ `ServeMux` wildcard paths, no router
  dependency. `Route`/`ResourceNames` (per-domain-noun overrides) live in
  `routes.go`; every domain gets its own file with its own
  `<domain>StatusFor`/`write<Domain>Err` pair (not shared — sentinel sets
  differ per domain, matching actor's own duplication). No
  `HierarchyChecker`-equivalent needed: unlike actor's `CreateGroup`/
  `EditGroup`, none of Agency's `Create`/`Update` methods need an
  HTTP-layer cross-check the manager doesn't already do. `CreateEventFlowStep`
  is the one manager method deliberately not exposed (internal seeding
  hook, not a requested write surface).

- **AG10 landed 2026-09-09: wired into `mwanachama-backend-api-gateway`.**
  Last item on this repo's board — AG1–AG10 all done. Followed git/
  taskmanager's single-instance gateway-wiring pattern (one `*gorm.DB`
  wrapper around the gateway's existing shared connection, mounted at
  `/v1/agency/...` behind `AGENCY_ROUTES=true`, gated by a single
  `CapAgencyRead`/`CapAgencyWrite` pair, org-wide not chapter-scoped) —
  not actor's multi-instance `userInstances` registry, since this
  package's `DefaultTableNames` takes one fixed instance label the same
  way git's/taskmanager's do. The gateway repo has its own hard test gate
  requiring every registered route to carry a real Postman request
  (`TestPostmanCollectionCoversEveryRoute`); satisfying it needed 60
  chained coverage requests covering all 56 route patterns, added to that
  repo's `operator-capabilities.postman_collection.json` (see that repo's
  own CLAUDE.md/todo_done.md for the full record — this repo's board only
  tracks that the wiring task is done, not the gateway's own internals).

- **AG12 landed 2026-09-10: `Objective`, `Value`/`Rule`, RACI moved
  WorkItem→Objective, `Agency.Mission`/`.Vision` deleted.** Implements
  [documentation/2. design/todo_details/AGD-001.md](documentation/2.%20design/todo_details/AGD-001.md)
  in full for this repo — a design session (run via an adapted
  `dev-research.md` protocol) starting from "how do goals, values and
  rules interact?", motivated by AI agents acting as `Role`s (the
  SWELab personas) having no access to the tacit organizational
  knowledge a human Role-holder would just know. Superseded **AG11**
  (`UpdateAgency` for Mission/Vision editing), which a concurrent session
  landed the same day via the same shared working tree, purely by
  coincidental timing — AG12's forward edits removed Mission/Vision
  outright rather than reverting AG11's commit.
  - `Objective` is new, first-class, **1:1 with `Goal`** (reversed by
    AG14 below — a Goal may now have several) —
    `CreateObjective` returns new `ErrDuplicateObjective` if the
    same-scope `GoalID` already has one. Draft-scoped like
    `Goal`/`Workflow`/`WorkItem` (nullable `DraftID`, no
    live-immutability check) — deliberately not Role/RoleAssignment's
    stricter "live cannot be edited" rule, since the literal decision
    text grouped Objective with the laxer three, not the stricter two.
  - **`WorkItem.GoalID` (optional, `""` = untagged) is gone.** Replaced
    by required `ObjectiveID` — `CreateWorkItem`/`UpdateWorkItem` return
    new `ErrInvalidWorkItem` if empty. There is no more untagged
    WorkItem.
  - **RACI moved from WorkItem-scoped to Objective-scoped.**
    `RoleAssignment.WorkItemID` → `ObjectiveID`;
    `ListRoleAssignmentsForWorkItem` → `ListRoleAssignmentsForObjective`,
    moved from `WorkItemRoutes` to `ObjectiveRoutes` in the HTTP layer.
    **`mwanachama-backend-api-kazi`'s K6 needs its own follow-up** — its
    "who to notify for a WorkItem" lookup must resolve via
    `WorkItem.ObjectiveID` now, not a direct WorkItem-scoped RACI call.
  - **`Value`/`Rule` are new**, same shape (`ID`, `ObjectiveID`
    — `""` = Agency-wide —, `DraftID`, `Text`, timestamps). Plain-text
    statements an AI agent acting as a Role needs made explicit (e.g.
    "we prioritize correctness over speed", "never touch production
    data") — informational only, nothing here enforces them at runtime.
    Agency-wide rows are **permanently immutable**: new
    `ErrCannotEditLive` on `Update`/`Delete` when live, and `CreateDraft`
    deliberately excludes `objective_id IS NULL` rows from its
    fork-copy loop so an Agency-wide row can never even enter a Draft to
    be edited that way. Objective-scoped Value/Rule follow Role's
    stricter live-immutability rule instead (fork→edit→promote); note
    `PromoteDraft`'s delete-old-live step for these two tables is
    scoped to `objective_id IS NOT NULL` specifically, so the always-live
    Agency-wide rows are never caught by a promotion's cleanup.
  - **AGD-001's write-authorization rule — only an Objective's
    Accountable may write its Values/Rules — is explicitly NOT
    implemented.** Flagged in `AgencyManager`'s doc comment and in
    AGD-001 itself: this codebase has no per-RACI-label write gate
    anywhere (auth is capability-gated at the gateway level, per AG10),
    and designing one was out of scope for this pass.
  - **`Agency.Mission`/`.Vision` are deleted** (model, gormstore,
    `UpdateAgency`, `liveContentHash`) — confirmed explicitly with the
    user ("delete them") over deprioritizing. `UpdateAgency` now only
    ever touches `Name`. **Cross-repo follow-up, not done here** (closed
    by AG13, see below): at the time, `mwanachama-wakala-api`'s AG11 (its
    own numbering) HTTP handlers and MCP tools
    (`mcp_tools_agency.go`/`mcp_tools_composite.go` — the AI-agent-facing
    surface, not just the human-facing one) and
    `mwanachama-wakala-studio`'s identity UI still referenced
    Mission/Vision, having drifted from this repo's own model.
  - `SeedSWELab` restructured from one Goal tagging one of three
    WorkItems (RACI per-WorkItem) to **three independent Goal→Objective→
    WorkItem chains** — Objective-scoped RACI means three distinct RACI
    sets need three distinct Objectives, not three WorkItems sharing one
    Goal.

- **AG13 landed 2026-09-10: `mcp/`, exposing this library's own manager
  methods as MCP tools directly — the AI-agent-facing surface
  `mwanachama-wakala-api` previously owned entirely
  (`mcp_tools_agency.go`/`mcp_tools_goal.go`/etc., ~57 tools).** Moved
  because that split had already caused real drift once (AG12's Mission/
  Vision removal needed a separate, after-the-fact fix in wakala-api) and
  would keep recurring on every future model change otherwise. Structured
  exactly like `routes/` (AG9) — one file per domain group, same tool
  names/descriptions/param shapes as before — except `mcp.AddTool` wires a
  handler onto the server once, at registration time, not per-request the
  way `routes/` rebuilds `Routes(m, names)` fresh per HTTP call, so every
  tool here closes over a `ManagerResolver` (`func(ctx, id string)
  (AgencyManager, error)`) instead of a concrete manager — see `mcp/mcp.go`.
  Also introduces `ListResult[T]{Items []T}`, which every List/Match tool
  now returns instead of a bare slice: MCP's spec defines
  `CallToolResult.structuredContent` as a JSON object, and a bare slice —
  though it satisfies the go-sdk's own lenient test client — fails a
  spec-strict one ("expected record, received array"), caught live against
  a real MCP client while building this. `mwanachama-wakala-api`'s
  `mcp.go` now just supplies `resolveAgencyManager` as the
  `ManagerResolver`; its own `agency_list_instances` tool (exposing that
  repo's Registry, decision #15 — a concept this library has no notion of)
  stays there, unmoved. See `mwanachama-backend-actor/CLAUDE.md`'s new
  "MCP tools" section for the cross-repo standard this establishes, and
  when (not) to follow it.

- **AG14 landed 2026-09-10: reversed AG12's "`Objective` is 1:1 with
  `Goal`" — a Goal may now have several Objectives.** Prompted directly by
  the user: a Goal like "Ship reliable software" needs up to ~10 granular
  Objectives, each one the unit a single Role is RACI-assigned to, rather
  than one broad Objective carrying many RoleAssignments. Turned out to be
  a narrow change — `ObjectiveRow.GoalID` was already a plain index, not a
  unique one (`gormstore/objective.go`), and neither `CreateDraft`/
  `PromoteDraft`'s copy loops (keyed by Objective id, not GoalID) nor
  `liveContentHash` (hashes every live row unconditionally) assumed
  singularity — so the entire enforcement was one count-check in
  `objective_impl.go`'s `CreateObjective`, now deleted along with the
  `ErrDuplicateObjective` sentinel it returned (and that sentinel's 409
  mapping in `routes/objective.go`). `RoleAssignment`'s existing "at most
  one Accountable per Objective" rule is unchanged — it stays scoped to
  the (now more numerous, more granular) Objective, matching each
  Objective as one role's unit of responsibility. See
  [documentation/2. design/todo_details/AGD-001.md](documentation/2.%20design/todo_details/AGD-001.md)'s
  Decision #2 for the updated design record.

- **AG15 landed 2026-09-11: every `mcp/` tool's success path now returns a
  one-line human-readable `Content` instead of the go-sdk's default JSON
  dump.** Prompted directly by the user, who saw raw `{"created_at":...}`
  blobs under a bare "Wakala [agency_set_raci]" header in their IDE's tool
  call log and asked for something readable. Added `mcp/mcp.go`'s
  `summary(format string, args ...any) *mcp.CallToolResult` — every
  handler's success-path return became `summary("Created Role %q (%s)",
  out.Name, out.ID), out, err` instead of `nil, out, err`.
  `StructuredContent` is untouched (confirmed at the wire level with a raw
  MCP call): the go-sdk fills it from the handler's actual typed return
  value regardless of `Content`, so no programmatic caller sees any
  difference. Error paths keep returning `nil` unchanged — the go-sdk
  discards a handler's `*mcp.CallToolResult` outright whenever it also
  returns a non-nil error. See `mwanachama-backend-actor/CLAUDE.md`'s "MCP
  tools" section for the standard this adds.
  - **Same session, second half:** the user then pointed out the call's
    own displayed *name* — `agency_set_raci` — was still unfriendly (the
    IDE's tool-call header showed the raw snake_case identifier). Every
    `&mcp.Tool{...}` literal across all 8 files gained a `Title` (e.g.
    `agency_set_raci` → `"Assign RACI"`) — the MCP spec's own sanctioned
    display-name field, confirmed present in a raw `tools/list` response.
    `Name` itself is never touched — it's the stable identifier every
    caller invokes by.

- **AG16 landed 2026-09-11: every named business-entity type gained a
  stable, human-readable `Code`** (`"G-1"`, `"R-1"`, ...) — Goal/Objective/
  Role/Department/Level/Workflow/WorkItem/Value/Rule, prefixed `G`/`O`/`R`/
  `D`/`L`/`W`/`WI`/`V`/`RU` respectively. Assigned once at creation, never
  editable (not in any `Update<Type>`'s explicit column map, matching
  `ID`'s own immutability), and confirmed to survive the Draft fork→promote
  cycle unchanged even though promotion mints a brand-new row ID (new
  `TestGoalCode_SurvivesDraftPromote`/`TestRoleCode_SurvivesDraftPromote` —
  `CreateDraft`/`PromoteDraft` needed no code changes at all, since their
  existing copy loops already round-trip every field via
  `<Type>FromRow`/`<Type>ToRow` except ID/DraftID/CreatedAt). Out of scope,
  confirmed by the task: `Agency` (singleton), `WorkPlan` (already has its
  own user-chosen `Code` slug), `RoleAssignment`/`Draft`/`ContextSource`/
  `AgencyPublication`/`AgencySnapshot` (join/audit rows), `EventFlowStep`
  (already has its own business-key `Code`).
  - New `gormstore/codesequence.go`: `CodeSequenceRow` (one row per entity
    type, `NextNumber`), `NextCode` (mints `"<prefix>-<n>"` inside the
    caller's transaction, so the counter increment and the row insert
    commit atomically), `BackfillCodes` (idempotent, assigns a Code to any
    pre-existing empty-Code row ordered by `created_at` — runs at the end
    of every `Migrate`, covering rows that predate this change). New
    `TableNames.CodeSequences` (`<instance>_code_sequences`).
  - **Deviation from the original spec's literal `gorm:"uniqueIndex"`
    instruction**: the Row struct's `Code` column uses plain
    `gorm:"index"` for all seven Draft-scoped types (Goal/Objective/Role/
    Workflow/WorkItem/Value/Rule), not a unique index. A Draft-scoped copy
    deliberately carries the *same* Code as its live original while both
    exist side-by-side in the same physical table (between `CreateDraft`
    and `PromoteDraft`) — a real DB-level uniqueness constraint would
    reject that insert outright, breaking every draft fork. This matches
    this repo's own existing precedent: `EventFlowStepRow.Code` and
    `WorkPlanRow.Code` (the two Code-shaped columns that already existed
    before this task) both use plain `index`, not `uniqueIndex`, for the
    same reason. `DepartmentRow`/`LevelRow` — never Draft-scoped, so no
    live/draft-copy collision is possible — do keep `gorm:"uniqueIndex"`
    as originally specified.
  - Each `Create<Type>` now wraps its row insert in
    `m.db.WithContext(ctx).Transaction(...)`, minting the Code via
    `gormstore.NextCode` before the insert. Each full-replace
    `Update<Type>` additionally sets `row.Code = existing.Code` right
    alongside its pre-existing `row.CreatedAt = existing.CreatedAt` line —
    needed because `Update<Type>`'s returned domain value is built from
    `<Type>ToRow(callerInput)`, and the caller-supplied input never
    carries a real Code (it isn't exposed on any `mcp` Update params
    struct), so without this the method would hand back a blanked-out
    Code even though the column itself was correctly left alone in the
    `cols` map.
  - `mcp/goal.go`/`objective.go`/`orgchart.go`/`raci.go`/`value_rule.go`:
    no structural change needed — each tool's structured output is the
    `models.X` value straight from the manager, so `Code` appears in
    `structuredContent` automatically once the model carries it. Reworked
    every affected `summary(...)` one-liner to lead with `Code` ahead of
    the Name/Title/Text, e.g. `"Created Goal %s %q (%s)", out.Code,
    out.Title, out.ID` — Code reads as the natural human-facing handle a
    raw UUID doesn't.

- **AG17 landed 2026-09-11: `mcp/dashboard_ui.html`'s View restyled to match
  `mwanachama-wakala-studio`'s design system, via a new shared
  `mcpui` package in `mwanachama-backend-shared`.** Prompted directly by the
  user ("same feel" as the Studio web app), then explicitly widened to a
  repo-wide strategy rather than a one-off local restyle, since AG13 already
  established this repo's `mcp/` as the reference implementation other
  backend repos' own MCP dashboards will follow.
  - `mwanachama-backend-shared/mcpui/theme.css` (+ `theme.go`'s
    `//go:embed` `ThemeCSS string`) now holds the *entire* design system —
    color tokens (light + `prefers-color-scheme: dark`, each layered over
    the MCP Apps host's own `--color-*`/`--font-sans` hostContext
    variables via `var(--color-x, <wakala-fallback>)` so a real host's
    theme still wins), IBM Plex Sans/Mono + Big Shoulders Display
    typography, and every base element/utility rule (`button`, `input`,
    `table`, `dl`/`dt`/`dd`, `.tile`/`.menu`, `.form-row`, `.back`,
    `.eyebrow`, `.empty`, `.section`/`.section.active`). Turned out to
    cover 100% of what this dashboard needed — `dashboard_ui.html` itself
    carries no `<style>` block at all now, only structure/behavior.
  - `mwanachama-backend-shared` was chosen over a new standalone repo
    because it already plays this role once — `mwanachama-backend-actor`
    depends on it via `replace ... => ../mwanachama-backend-shared`, and
    its own tree already mixes unrelated concerns (`entitygraph`,
    `orgpolicy`, `orgsettings`, `events`) despite its CLAUDE.md describing
    it narrowly as "the entity-graph engine" — `mcpui` is one more
    non-entitygraph package riding the same dependency, not a new pattern.
  - `dashboard_ui.go` now embeds the raw page as `dashboardUIPage` and
    computes `dashboardUIHTML` once at package init by splicing
    `mcpui.ThemeCSS` in as a `<style>` block right after `<meta charset>`
    (`strings.Replace`, not a template) — any future page-specific CSS a
    dashboard needs can still follow in its own later `<style>` block and
    win the cascade normally.
  - Not done here, flagged for later rather than silently bundled in:
    the SEP-1865 postMessage/JSON-RPC client boilerplate (`send`,
    `callTool`, `esc`, the `ui/initialize` handshake, the menu/section
    router) in `dashboard_ui.html`'s `<script>` is equally duplicated
    per-repo protocol code, not agency-specific logic — a natural next
    candidate for the same `mwanachama-backend-shared` treatment once a
    second repo actually builds its own MCP dashboard.

- **AG18 landed 2026-09-14: every `List*` method gained real, per-module
  server-side filtering** — driven by `mwanachama-wakala-studio`'s new chat
  "Search" shortcut (pick a module, then that module's own filter fields,
  e.g. Role's Name/Description), which needed something to actually call:
  until this task, every `List*` (REST and MCP) accepted only `draft_id` (or
  `workflow_code` for `EventFlowStep`) — no field-level filtering existed
  anywhere.
  - New root-package `filters.go`: one `<Module>Filter` struct per
    searchable module (`GoalFilter`, `RoleFilter`, `RoleAssignmentFilter`,
    ...) — `Code`/`Name`-or-`Title`-or-`Text`/`Description` as
    case-insensitive substring fields, `Status`/`Label`/`SourceType` as
    exact-match fields on the handful of enum-shaped columns (`Draft`,
    `AgencyPublication`, `RoleAssignment`, `ContextSource`).
  - Two new query-builder helpers next to `scopeQuery` in `draft_impl.go`:
    `likeFilter(q, column, value)` (`LOWER(column) LIKE ?`, matching the
    existing `roleNotFoundError` idiom — portable to the SQLite this repo's
    tests run against, unlike Postgres-only `ILIKE`) and `eqFilter(q,
    column, value)` for the exact-match fields; both no-op on `""`.
  - All 17 `AgencyManager.List*` methods (every module `ResourceNames`
    lists, minus `Agency` itself — a singleton with no `List` at all —
    plus `ListDepartmentChildren`/`ListRoleAssignmentsForObjective`, which
    take the same filter as their sibling `List<Type>`) gained a trailing
    `filter <Module>Filter` parameter, each `*_impl.go` chaining the new
    helpers onto its existing query before `.Find`. `routes/*.go` reads the
    new filter fields as additional query params right alongside the
    existing `draft_id`/`workflow_code` ones; `mcp/*.go` gives each
    `ListX` tool its own named optional filter fields (`code`, `name`,
    `description`, ...) on that tool's own params struct rather than
    adding them to the shared `instanceAndDraftParams`/`instanceOnlyParams`
    every other (non-List) tool also embeds, since the fields genuinely
    differ per module.
  - Confirmed via a dedicated blast-radius check before touching the
    interface: no mock/fake implements `AgencyManager` anywhere (only the
    one real `agencyManager` struct), so this was a single-implementation
    signature change, not a multi-site one. One real external call site
    needed a source fix: `mwanachama-backend-api-shared`'s
    `mcp_tools_approval.go` calls `ListRoleAssignmentsForObjective` directly
    (K6's approval-gate `resolveAccountable`) and now passes a zero-value
    `RoleAssignmentFilter{}`. Ten of this repo's own existing tests
    (`role_test.go`, `goal_test.go`, `level_test.go`, `department_test.go`,
    `workplan_test.go`, `contextsource_test.go`, `eventflowstep_test.go`,
    `roleassignment_test.go`, `swelab_test.go`, `draft_test.go`) needed
    the same mechanical zero-value-filter-argument addition at their direct
    `mgr.ListX(...)` call sites; `routes/*_test.go` needed no changes at
    all, since those call the HTTP handler wrappers, not the interface
    method directly.
  - `ContextSource` is excluded from `mwanachama-wakala-studio`'s Search
    module list (its `List` is nested under a `WorkPlanID`, not a
    standalone agency-wide listing) even though it did get a
    `ContextSourceFilter` here — the backend filter exists for whichever
    future caller does have a `WorkPlanID` in hand.
  - Verified: `go build`/`go vet`/`go test ./...` all clean in this repo.
    Also rebuilt and retested `mwanachama-backend-api-shared`,
    `mwanachama-wakala-api`, `mwanachama-backend-api-gateway`, and
    `mwanachama-backend-api-kazi` against the changed interface (all four
    `replace` onto this repo locally) — all still build and test clean.
    `mwanachama-backend-api-gateway`'s `TestPostmanCollectionCoversEveryRoute`
    fails, but that predates this change entirely (confirmed via a clean
    `git status` in that repo before touching anything there) — unrelated
    missing Postman coverage for several pre-existing Objective/Role/
    Value/Rule routes, not something this task's unchanged route *shapes*
    (only their query params grew) could have caused.

- **AG19 landed 2026-09-14: new `mcp/catalog.go`, a static domain-type
  glossary tool.** Prompted directly by the user: `mwanachama-wakala-studio`'s
  chat had no way to answer an operator's "what is an Objective" or "how do
  I add a Role" — every `mcp/*.go` tool's `Description` explains how to call
  that one tool, not what the type conceptually is, and the real
  conceptual explanations only ever existed as `models/*.go` doc comments,
  never exposed at runtime. New `agency_list_types` tool, backed by a
  package-level `domainTypeCatalog []typeInfo` (`Name`/`WhatItIs`/
  `HowToAdd`, one entry per domain type — Agency, Department, Level, Goal,
  Objective, Role, RoleAssignment/RACI, Value, Rule, Workflow, WorkItem,
  WorkPlan, ContextSource, EventFlowStep, Draft, AgencyPublication,
  AgencySnapshot), condensed from each type's own `models/` doc comment
  rather than restating it verbatim. Takes no `ManagerResolver` — the
  catalog is static content about the data model itself, not any one
  Agency's data — following `dashboard_ui.go`'s `registerDashboardUI(server)`
  precedent for a tool that genuinely needs no manager. `name` is an
  optional case-insensitive substring filter, matching every other
  `agency_list_*` tool's own filter convention, rather than an exact-match
  lookup that would need a separate "unknown type" error path.
  `mwanachama-wakala-studio`'s `useChat.ts` system prompt gained one
  sentence telling the model to call this tool for definitional/how-to
  questions instead of answering from general knowledge — no other Studio
  change needed, since its chat already forwards every MCP tool's
  name/description straight to the LLM. `mwanachama-wakala-api` needed no
  change either, since its `mcp.go` just calls this package's
  `RegisterTools` directly. Verified: `go build`/`go vet`/`go test ./...`
  clean here and in `mwanachama-wakala-api` (both locally `replace`d),
  `tsc --noEmit` clean in `mwanachama-wakala-studio`.

- **AG20 landed 2026-09-14: `DeleteWorkflow`, cascading to its WorkItems.**
  Closes half of WKS30's (`mwanachama-wakala-studio`) known gap — its
  `WorkflowsPanel` had Edit but no Delete button because this repo had no
  `agency_delete_workflow` tool to call; Work Item Delete is still missing
  and stays out of scope here. Exact same shape as AG18's `DeleteGoal`/
  `DeleteObjective`: one transaction in `workflow_impl.go` deleting
  `WorkItem` rows `WHERE workflow_id = id` before deleting the `Workflow`
  row itself, a `mcp/goal.go` `agency_delete_workflow` tool reusing the
  already-existing `mcpWorkflowIDParams` (previously only used by Get), and
  root `workflow_test.go` (new file — Workflow had no root-level test file
  yet) covering not-found, cascade, and no-WorkItems-to-cascade cases. A
  WorkItem carries no RACI of its own (RoleAssignment is Objective-scoped
  since AG12), so — unlike `DeleteGoal`'s further cascade into
  RoleAssignments — there is nothing past WorkItem to cascade into.
  `mwanachama-wakala-studio`'s own `deleteWorkflow`/cascade-count-warning
  wiring needed one thing `DeleteGoal`/`DeleteObjective`'s Studio side
  didn't: Goal/Objective are shown live-only throughout Studio, but
  Workflow/WorkItem genuinely are edited via the standing Draft, so the
  cascade count's `agency_list_work_items` call must pass the real
  `draft_id` (whatever scope the Workflow being deleted is actually in),
  not a hardcoded live-only `''`. Verified: `go build`/`go vet`/`go test
  ./...` clean here and in `mwanachama-wakala-api` (locally `replace`d),
  `tsc --noEmit` and `vitest run` clean in `mwanachama-wakala-studio`.

- **AG21 landed 2026-09-15: closed WK17 (`mwanachama-wakala-api`'s board) —
  every `Create<Type>` manager method now owns `ID`/`CreatedAt`, and
  validates a caller-supplied `DraftID`.** WK17 was filed against
  wakala-api and named this package's `routes/` package as the fix
  location; investigating turned up a bigger, more accurate picture. All 13
  `Create<Type>` methods (`CreateAgency`, `CreateDepartment`, `CreateLevel`,
  `CreateGoal`, `CreateObjective`, `CreateRole`, `CreateRoleAssignment`,
  `CreateValue`, `CreateRule`, `CreateWorkflow`, `CreateWorkItem`,
  `CreateEventFlowStep`, `CreateWorkPlan`) shared the identical
  `if X.CreatedAt == "" { X.CreatedAt = models.NowRFC3339() }`/
  never-clear-`ID` pattern — a caller-supplied non-empty value for either
  field was written verbatim, honouring a clock-forgery vector and (since
  `gormstore`'s `BeforeCreate` hooks only mint an id `if r.ID == ""`) a
  chosen primary key. Fixing only `routes/` (as filed) would have left
  `mcp/`'s identical surface exploitable too — `mcp/goal.go`'s own
  `mcpGoalParams.ID` doc comment already claimed "required for update,
  ignored for create," which `toModel()` did not actually do until this
  fix landed. **Fixed at the one real source — inside each `Create<Type>`
  itself** (`X.ID = ""` before building the row, `X.CreatedAt =
  models.NowRFC3339()` unconditionally, dropping the `if == ""` guard) —
  this closes both `routes/` and `mcp/` in one place, and any future
  caller besides, the same way `Code`'s own unconditional
  `gormstore.NextCode` mint already worked.
  - **`DraftID` needed a materially different fix, not the same
    clear-it-outright treatment.** `issue_test.go`/`roleassignment_test.go`
    already exercise creating a row directly scoped into an existing open
    Draft as a real, tested capability (not only via `CreateDraft`'s own
    fork-copy) — clearing the field unconditionally would have broken it.
    New `validateOpenDraftID` (`draft_impl.go`, beside `findDraftRow`) is
    what the 9 Draft-scoped `Create<Type>` methods (every one above except
    Agency/Department/Level/WorkPlan, which carry no `DraftID` field at
    all) call instead: `""` always passes; a non-empty value is looked up
    via the existing `findDraftRow` and refused with the already-existing
    `ErrDraftNotFound` (unknown) or `ErrDraftNotOpen` (promoted/archived) —
    closing WK17's actual concern (a live write attached to an arbitrary
    or already-terminal Draft that `PromoteDraft`'s copy-and-cutover logic
    was never going to reach) without touching the legitimate case.
  - All 9 affected `routes/*.go` status-mapping functions
    (`goalStatusFor`, `objectiveStatusFor`, `roleStatusFor`,
    `roleAssignmentStatusFor`, `workflowStatusFor`, `workItemStatusFor`,
    `eventFlowStepStatusFor`, `valueStatusFor`, `ruleStatusFor`) gained
    `ErrDraftNotFound`→404 / `ErrDraftNotOpen`→409 arms — the same error
    path this fix introduces, not a separate change; left unmapped, both
    sentinels would have fallen through to a bare 500 "internal error"
    instead of a clean refusal.
  - New root-package `security_wk17_test.go` (caller-supplied id/created_at
    ignored; unknown/promoted/archived `draft_id` refused; an open
    `draft_id` still works) plus two HTTP-level cases added to
    `routes/goal_test.go`. Verified: `go build`/`go vet`/`go test
    ./...`/`gofmt -l .` all clean. Also rebuilt and retested
    `mwanachama-wakala-api` (locally `replace`d) — its new
    `security-03-agency-create-ignores-server-owned-fields` self-test
    (`cmd/server/selftest/security/`) replays the original WK17 curl
    reproduction over the real router and passes 7/7; that repo's
    pre-existing `security-01` doc comment, which had explicitly named this
    exact gap as real and unguarded, is updated to point at security-03
    now that it holds.
  - **Found, not fixed, along the way**: wakala-api's `make self-test` has
    one pre-existing, unrelated failure — `objective-raci`'s "a second
    Objective on the same Goal is refused" check, which predates AG14
    (2026-09-10) reversing the 1:1 Goal:Objective rule and was never
    updated after. Flagged on that repo's own board note, not touched
    here.

- **AG22 landed 2026-09-15: `Value`/`Rule` attach to Role/Goal/Objective/
  Workflow/WorkItem, not just Objective — and to more than one target at
  once.** Implements
  [AGD-003](../2.%20design/todo_details/AGD-003.md), a design
  conversation the same day prompted directly by the user ("I want to
  change Values and Rules to be attachable to Roles, Goals, Objectives,
  Workflows and WorkItems"). `ValueRow`/`RuleRow` lost `ObjectiveID`
  entirely — scoping now lives in two new join tables,
  `ValueAttachmentRow`/`RuleAttachmentRow` (`models/attachment.go`'s
  `TargetType` enum: role/goal/objective/workflow/work_item), each row
  carrying zero-to-many attachments. Zero attachments still means
  Agency-wide, same permanent-immutability rule as before.
  - **Referenced by Code, not row id, on both sides** — the one detail
    AGD-003 flagged as a consequence rather than an upfront decision:
    every attachable type (Value/Rule included) mints a brand-new row id
    on every Draft promotion (AG6's own note on this), but attaching is
    live-editable (AGD-003 decision #3) and so sits outside the fork/
    promote lifecycle entirely — an id-keyed attachment would silently
    go stale the first time anyone promoted any Draft touching either
    side. Code survives that churn unchanged (AG16), so
    `CreateValueAttachment`/`CreateRuleAttachment` resolve both the
    Value/Rule and the target to their Code at write time
    (`attachment_impl.go`'s `resolveCodeByID`/`resolveTargetCode`) and
    store that, not the id. Reads (`ListValueAttachments`, and
    `ListValues`/`ListRules`'s new target-id filters) always re-resolve
    against whatever is live right now, so a caller never sees a stale
    id. One correctness side effect worth knowing: this makes
    `CreateDraft`/`PromoteDraft`'s Value/Rule handling *simpler* than
    before, not more complex — fork-eligibility changed from a column
    test (`objective_id IS NOT NULL`) to an `EXISTS`-style Code lookup
    against the attachments table, and the old per-row `ObjectiveID`
    remap in `CreateDraft`'s copy loop is gone outright, since
    attachments are never forked or remapped at all.
  - **Ten narrow join tables vs. two polymorphic ones** — AGD-003's one
    explicitly left-open call — resolved as **two** (`ValueAttachment`,
    `RuleAttachment`, each with `TargetType`/`TargetCode`), not ten. Ten
    near-identical tables/files for what's structurally the same
    relationship shape didn't earn its keep against AGD-002's
    `ObjectiveModule` precedent once weighed against the actual
    duplication cost; two tables with a `(target_type, target_code)`
    index recovers most of the query performance a named column would
    have given for free.
  - **Existence validation** reuses each target type's own existing
    NotFound sentinel (`ErrRoleNotFound`, `ErrGoalNotFound`, ...) via
    `agencyManager.targetTable` — `CreateValueAttachment`/
    `CreateRuleAttachment` return the specific one, not a generic "target
    not found". New `ErrInvalidTargetType` for an unrecognized
    `TargetType`, `ErrDuplicateAttachment` for re-attaching the same
    (Value/Rule, target) pair.
  - **No create-time convenience** — `CreateValue`/`CreateRule` still
    take only `Text`/`DraftID`; attaching is always a separate
    `CreateValueAttachment`/`CreateRuleAttachment` call. AGD-003 left
    this open; not built, to keep Create's shape unchanged.
  - **`ValueFilter`/`RuleFilter` gained `RoleID`/`GoalID`/`ObjectiveID`/
    `WorkflowID`/`WorkItemID`** (`attachment_impl.go`'s
    `applyTargetFilters`) — the List-filtering gap AGD-003 flagged (there
    was no server-side Value/Rule filtering of any kind before this,
    not even by the old single `ObjectiveID`) is closed for all five
    target types at once. An id that resolves to no row at all yields an
    empty list, not an error — same "narrows to nothing" convention
    every other filter here follows.
  - Write-authorization (AGD-001's "only a target's Accountable may
    write") stays unbuilt, as AGD-003 anticipated — flagged again in
    `AgencyManager`'s doc comment.
  - `mcp/value_rule.go` gained `agency_attach_value`/
    `agency_list_value_attachments`/`agency_detach_value` and the Rule
    equivalents; `agency_create_value`/`agency_create_rule` lost their
    `objective_id` parameter. `routes/value.go`/`rule.go` gained the
    nested `/{id}/attachments` sub-resource
    (POST/GET, plus `DELETE .../attachments/{attachmentID}`) and
    `?role_id=`/`?goal_id=`/etc. query params on the List routes.
  - **Found and fixed along the way**: `gormstore/value_attachment.go`'s
    first cut named its composite index with a fixed string
    (`idx_value_attachment_target`) — broke immediately against
    `mwanachama-wakala-api`'s multi-instance registry, since SQLite index
    names are database-global, not per-table, and every instance's table
    set shares one physical SQLite/Postgres database. Fixed by dropping
    back to plain unnamed `gorm:"index"` per column (GORM auto-names it
    from the table name, matching every other indexed column in this
    package) — caught by `mwanachama-wakala-api`'s own
    `TestAgencyRoutes_ScopedPerInstance` and `TestMCP_
    MinimalArgumentsAcrossToolGroups`, not by this repo's own suite
    (a single-instance test harness never exercises the collision).
  - New `value_test.go`/`rule_test.go` (this repo had zero Value/Rule
    test coverage before this task, at any layer) and
    `routes/value_test.go` — attach/list/detach lifecycle, duplicate/
    invalid-target-type/target-not-found errors, multi-target-on-one-Value,
    Agency-wide permanent-immutability, and the key regression case: a
    Value/Rule attached before a fork still resolves after
    `PromoteDraft` replaces its live row with a new id.
  - Verified: `go build`/`go vet`/`go test ./...`/`gofmt -l .` all clean
    here. Rebuilt and retested `mwanachama-backend-api-shared`,
    `mwanachama-wakala-api`, `mwanachama-backend-api-gateway`, and
    `mwanachama-backend-api-kazi` (all four locally `replace`d) — all
    build clean; `mwanachama-wakala-api`'s own `mcp_test.go` needed one
    fix (`TestMCP_MinimalArgumentsAcrossToolGroups` still called
    `agency_create_value` with the now-removed `objective_id` argument)
    and its `cmd/server/selftest/selftest_value_rule.go` needed updating
    to attach via the new endpoint instead of a create-time argument —
    both fixed in that repo directly, not filed as a follow-up, since
    they were mechanical and directly caused by this change. The
    gateway's `TestPostmanCollectionCoversEveryRoute` gate — already
    failing before this task (AG18/AG20's own notes) — now reports 23
    uncovered routes, up from 17: this task's 6 new attachment routes
    (3 each for Value/Rule) land on the same pre-existing, already-filed
    gap, not a new one; not fixed here, same reasoning as AG12/AG20's own
    deferral. `mwanachama-wakala-studio` needed no change — grepped for
    any hand-written Value/Rule UI referencing `objective_id` and found
    none; Value/Rule has no dedicated Studio panel yet, so it already
    falls through the generic module fallback the CLAUDE.md's own
    AG13 cross-repo note describes.

- **AG23 landed 2026-09-15: every List/Match MCP tool now returns
  `summaryWithContext`, not bare `summary`.** Caught live in
  `mwanachama-wakala-studio`: an operator asked its chat assistant to add
  WorkItems to Workflow "W-1", the model called `agency_list_workflows`
  filtered by that code, and got back only `"Found 1 workflow(s)"` — no
  id anywhere — so it had no way to actually call
  `agency_create_work_item`. `agency_get_agency`/`agency_list_goals`/
  `agency_list_objectives`/`agency_list_roles`/`agency_list_issues` (AG13/
  AG19-era work) already used `summaryWithContext` correctly; every other
  List/Match tool — 17 of them — still used the plain `summary("Found %d
  X(s)", len(out))` one-liner that only a human reading a call log needs,
  leaving every per-item field (id included) unreachable by the model
  itself, since `mwanachama-wakala-studio`'s `toolText()` joins whatever
  `Content` text blocks a tool result carries and never falls back to
  `structuredContent` once `Content` is non-empty (confirmed correct
  there — see that repo's own note on this incident; the gap was entirely
  here, not a client-side bug).
  - Closed everywhere at once rather than just for Workflow: new
    `workflowsContextText`/`workItemsContextText`/
    `eventFlowStepsContextText` (`goal.go`), `departmentsContextText`
    (shared by `agency_list_departments`/`agency_list_department_children`,
    `orgchart.go`) /`levelsContextText` (`orgchart.go`),
    `workPlansContextText`/`workPlanMatchesContextText`/
    `contextSourcesContextText` (`workplan.go`), `roleAssignmentsContextText`
    (shared by `agency_list_role_assignments`/
    `agency_list_role_assignments_for_objective`, `raci.go`),
    `draftsContextText`/`publicationsContextText` (`agency.go`),
    `valuesContextText`/`valueAttachmentsContextText`/`rulesContextText`/
    `ruleAttachmentsContextText` (`value_rule.go`) — each follows the
    existing `rolesContextText`/`goalsContextText` shape exactly: `"There
    are no X yet."` when empty, else `"- <Code>: <label> (... id: <ID>,
    ...)"` per line, built from named fields rather than a struct/JSON
    dump (`mcp.go`'s own doc comment on why). `catalog.go`'s
    `agency_list_types` is the one List tool deliberately left alone — its
    `typeInfo` rows carry no id at all, nothing to surface.
  - `agency_list_publications` leads each line with `Tag`/`Version`, not
    ID — `agency_get_publication`/`agency_update_publication_status` both
    take `Version`, not an id, so that's the field a caller actually
    needs back.
  - Verified: `go build`/`go vet`/`go test ./...`/`gofmt -l .` all clean
    here; rebuilt and retested `mwanachama-wakala-api` (locally
    `replace`d) — clean too. Not yet re-verified live against a running
    chat — the dev server process that hit this bug was already running
    before the fix landed and needs a restart to pick it up (`go run`
    doesn't hot-reload).

- **AG24 landed 2026-09-15: `agency_design_from_spec`'s `raci` section can
  reference an already-existing Role/Objective by real id, not only a
  same-spec key.** Found while diagnosing why `mwanachama-wakala-studio`'s
  chat took ~2 minutes for "add a test role assignment": that request
  needed a new Role plus a RACI link to an Objective that already
  existed, and the composite tool's `raci[].role_key`/`objective_key`
  could only reference entities created within that same spec call —
  forcing the model back to two sequential granular calls
  (`agency_create_role` then `agency_set_raci`), each its own full LLM
  round trip. `specRACI` (`mcp/composite.go`) gained optional
  `role_id`/`objective_id` fields: set the `_key` field for an entity
  created in this same spec, or the `_id` field for one that already
  exists — the RACI loop resolves whichever is set (key takes precedence
  if both are given), erroring only if neither is. Every other spec
  section (departments' `parent_key`, work items' `workflow_key`/
  `objective_key`) has the identical key-only limitation but was left
  alone — not fixed here, flagged for later rather than silently bundled
  in. New `TestDesignFromSpec_RaciAgainstExistingObjective`
  (`mcp/mcp_test.go`) covers the exact reported shape. No client-side
  (`mwanachama-wakala-studio`) change needed — per AG19's own precedent,
  the tool's `Description` is forwarded to the model verbatim, so it was
  extended to name the new fields and when to use them instead of the
  two-call round trip. `go build`/`go vet`/`go test ./...`/`gofmt -l .`
  clean here; rebuilt and retested `mwanachama-wakala-api` (locally
  `replace`d) clean too — the only other repo importing this package's
  `mcp` subpackage.

- **AG26 landed 2026-09-16: publishing is gated by the Issue scan, and
  `PublishAgency` takes an operator tag.** Implements
  [AGD-004](documentation/2.%20design/todo_details/AGD-004.md), a design
  session the same day starting from "I want the agencies validated when
  the publish button is posted. It should tag the agency." Until this
  task `PublishAgency` validated no content at all — only that a named
  Draft was promoted and that the content hash had changed — while the
  one validation mechanism in the repo (AG20's Issue scan) was
  Draft-only and produced nothing above `warning`.
  - **The scan is now scope-agnostic and `ErrDraftIDRequired` is
    deleted.** `scanDraftIssues` → `scanIssues`, `IssueRow.DraftID` is a
    nullable `*string` like every other Draft-scoped row, and both
    `ListIssues`' delete and its read go through the existing
    `scopeQuery` rather than a bespoke `draft_id = ?`. This was the whole
    architectural knot: publish hashes *live*, the scan only knew
    *drafts*, and nothing could have run one against the other. Turned
    out to be a small change — `scanIssues` already threaded its
    `draftID` through `scopeQuery`, which has always treated `""` as
    live; the `ErrDraftIDRequired` guard was the only thing stopping it.
  - **The scan gained a small internal structure rather than growing one
    long function**: an `issueScan` struct (scope + accumulated findings)
    with one method per rule (`checkStructure`, `checkRACI`,
    `checkWorkItems`, `checkEventFlowSteps`, `checkAttachments`), which
    is what made adding five families at once readable. `rescanIssues`
    (delete-scan-reinsert, returning the findings) is shared by
    `ListIssues` and the publish gate, so both read the same rules from
    the same place rather than the gate reimplementing them.
  - **Four check families became `error` — the first producers of
    `models.IssueSeverityError`, which had been declared since AG20 and
    emitted by nothing.** Dangling references (WorkItem→missing
    Objective; RoleAssignment→missing Role/Objective; Value/Rule
    attachment whose `TargetCode` no longer resolves; EventFlowStep whose
    `workflow_code` matches no Workflow); Objective with no Accountable
    (promoted from `warning`); Agency with no Goals (new) and Goal with
    no Objectives (promoted from `info`); Workflow with no WorkItems
    (new). Nothing emits `warning` or `info` any more — the two severities
    are kept, and the gate keys on severity rather than issue count, so a
    future advisory check needs no new machinery.
  - **Attachment targets resolve in the scan's own scope, not always
    live.** Attachments are never Draft-scoped (AGD-003) but their
    targets are, and a Code survives promotion unchanged (AG16) — so
    scanning a Draft surfaces "this Draft deletes a Role a Rule is
    attached to" *before* promoting makes it true of live, which
    resolving against live could not.
  - **`PublishAgency(ctx, draftID, tag)`**: a non-empty tag must be
    unused (`ErrDuplicateTag`), `""` falls back to `v<version>` — and the
    fallback is collision-checked too, since an operator is free to name
    an earlier publication "v7" by hand. `Version` stays the machine key
    `GetPublication`/`UpdatePublicationStatus` take. The gate runs inside
    the same transaction, before hashing, and refuses with
    `ErrPublishValidationFailed` wrapping the blocking messages;
    `routes/publication.go` maps both new sentinels to 409.
  - `routes/issue.go` gained the live-scope route (`GET /issues` beside
    the existing `GET /drafts/{draftID}/issues`) — one handler serves
    both, since `r.PathValue("draftID")` reads back `""` when the pattern
    carries no such wildcard. `issueStatusFor` now has no sentinel arms
    at all and is kept only as a pair with `writeIssueErr`, matching
    every other domain file.
  - **Every publish test needed a new seed.** `seedLiveAgency` leaves its
    Objective with no Accountable, which is now blocking, so
    `publication_test.go` gained `seedPublishableAgency` (the same seed
    plus an Accountable Role) and the routes package gained its own
    equivalent. Two existing tests needed their premise changed rather
    than their assertions: `TestPublishAgency_NoDraft` changed its
    "make a change between publishes" step from `CreateGoal` (which now
    produces a blocking no-Objectives error) to an Agency-wide
    `CreateValue`, and `TestPublishAgency_RACIOnlyChangeIsAChange` now
    adds a *Consulted* assignment, since its Accountable one is part of
    the seed.
  - Worth knowing: `SeedSWELab` publishes clean on the first try, which
    is the best evidence the check set is calibrated — a realistic
    agency passes, `seedLiveAgency`'s deliberately-partial one does not.
  - Verified: `go build`/`go vet`/`go test ./...`/`gofmt -l .` all clean
    here. Rebuilt and retested `mwanachama-backend-api-shared`,
    `mwanachama-wakala-api`, `mwanachama-backend-api-gateway` and
    `mwanachama-backend-api-kazi` (all four locally `replace`d) — all
    build and test clean; **no external call site passes `PublishAgency`
    or `ListIssues` directly**, so despite being an interface change this
    needed no source fixes outside this repo, unlike AG18's. The
    gateway's `TestPostmanCollectionCoversEveryRoute` gate reports 29
    uncovered routes; exactly one of those (`GET /v1/agency/issues`) is
    this task's, landing on the same pre-existing already-filed gap
    AG18/AG20/AG22 each landed on. Same deferral.

- **AG27 landed 2026-09-16: three delete paths that were stranding rows.**
  Found while designing AG26's dangling-reference checks — the checks
  exist because these paths produced exactly the orphans they detect,
  through the ordinary UI:
  - `DeleteGoal`/`DeleteObjective` cascaded into `RoleAssignment` but
    never into `WorkItem`, even though `WorkItem.ObjectiveID` has been
    *required* since AG12 — so a surviving WorkItem was unreachable, not
    merely untagged. (`DeleteWorkflow`/AG20 already cascaded its
    WorkItems correctly; these two predate required-`ObjectiveID`
    semantics.)
  - `DeleteRole` deleted only the Role row, leaving `RoleAssignment` rows
    pointing at a dead `role_id` — while `DeleteObjective` cleans that
    same join table from the other side. Now a transaction, matching
    `DeleteObjective`'s shape.
  - The scan is the regression net, not the fix: AG26's checks still have
    to catch rows already stranded in any database that ran the old
    code, which is why `TestScan_DanglingRoleAssignment` plants its row
    by deleting the Role row directly (new `newTestManagerDB` test helper
    exposes the store for exactly that case) rather than through a
    manager call that no longer produces one.
  - `CreateWorkItem` still accepts a *nonexistent* `ObjectiveID` (it only
    rejects an empty one) — found, not fixed, since AG26's scan now
    reports it and tightening Create is its own decision. That open path
    is what `TestScan_DanglingWorkItem` uses.

- **AG28/AG29/AG30 landed 2026-09-17: a publication carries the whole
  Agency, production runs off tags, and WorkPlan/ContextSource are gone.**
  Implements [AGD-005](documentation/2.%20design/todo_details/AGD-005.md),
  a design session prompted by the user challenging a cosmetic
  `scope: 'live' | 'draft'` field in the Studio's Issues panel: *"we can
  have a tag record with a json field that has all the agency info from
  which the live/production agency can operate"*, then *"production only
  operates from the tag table/model"*. Before this, a publication stored
  **only a content hash** — nothing could answer "what was in v1", there
  was no rollback and no diff, and production read the mutable working
  rows, so publishing had no operational meaning at all. `AgencySnapshot`,
  which sounds like it solved this, is an empty stub too (`{DraftID,
  SnapshotAt}`) — two receipt-shaped records and zero snapshots.
  - **AG28 — the document and the channel model.** New
    `models/agencydocument.go` + root `document.go`'s `buildDocument`:
    `AgencyPublication.Document` is the whole Agency, serialized in
    `mwanachama-wakala-api`'s **existing import/export shape** (its
    decision #19) so publish↔import round-trips and the 52 mined templates
    are already written in it — no new format invented. Stored as JSON
    text on the row, not a driver-specific JSON column: written once, read
    whole, never queried into. Two sections the import endpoint doesn't
    itself apply are carried anyway (EventFlowSteps, the Agency's non-
    identity settings); import decodes leniently, so that's harmless.
  - **`Status` became `Channel`** — the user's own reframing, better than
    any option offered: "live" stops being a tier of rows and becomes a
    state on the tag. A channel is a name (`"production"`, `"staging"`, …)
    or `""` for archived, unique across publications — **several tags may
    be live at once on different channels**, one tag per channel.
    Publishing displaces only its own channel. New
    `GetPublicationForChannel` is the production read. The old
    `UpdatePublicationStatus`'s one-way `active → archived` rule is gone,
    replaced by `SetPublicationChannel`, which allows both directions
    because **re-channelling an archived publication is the rollback** —
    seconds, not a rebuild.
  - **The mutable `draft_id IS NULL` row set is no longer "live"** — it is
    the *working set*, the design surface. "Live" now only ever means a
    channel on a tag.
  - **AG29 — `CreateDraft` can fork from a tag.** `forkedFromType` gained
    `"publication"`, where `forkedFromID` is the **Tag**, not a row id —
    the handle an operator actually names. A publication is a document,
    not a row scope, so it is materialized whole by `document.go`'s
    `applyDocumentToDraft` rather than walked by the existing copy loops:
    fresh row ids, the document's ids remapped onto them, and **Code
    carried across verbatim** (AG16), which is what keeps EventFlowStep's
    `WorkflowCode` join and the attachment tables resolving. Attachments
    are deliberately not applied — never Draft-scoped (AGD-003), so the
    live rows already cover the copies by Code.
  - **AG30 — `WorkPlan`/`ContextSource` decommissioned outright**, code
    removed and tables dropped, on the user's call ("they are currently
    useless"). Evidence gathered first: `mwanachama-wakala-studio` already
    hid `work_plan` (`ShortcutsMenu.tsx` excluded it), `MatchWorkPlans`
    had no external caller at all, and the only non-agency Go reference is
    `mwanachama-backend-taskmanager/events.go`'s `WorkPlanID`/
    `WorkPlanCode` **event-payload fields** — a reference to the concept,
    never a call into this library; filed as W11 on that repo's board
    rather than edited from here. **`EventFlowStep` is explicitly kept**
    ("eventflow is important") — unlike WorkPlan it is Draft-scoped, in
    the content hash, and rides into the document naturally.
  - New `gormstore.DropRetiredTables`, run at the end of every `Migrate`,
    sheds the two retired tables and the retired `status` column on a
    pre-existing deployment's next boot. Driven from a
    `retiredTableSuffixes` list so the next retirement is one entry.
    Covered by tests that **seed a database with the old shape first** —
    the only case that exercises a drop at all, since a fresh database
    never creates either.
  - Every pre-AGD-005 publication lands on `ChannelArchived` with an empty
    document. They cannot be back-filled (the rows they described have
    moved on) and are readable as history only, exactly as AGD-005 says.
  - **Not done, and not filed as a row**: the consumer cutover to reading
    the tag. It needs its own pass per consumer, and almost nothing reads
    agency structure in production today — the gateway's boot-time module
    gating and K6's RACI lookup are the whole list.
  - Verified: `go build`/`go vet`/`go test ./...`/`gofmt -l .` clean here.
    All four `replace`-ing repos rebuilt and retested clean;
    `mwanachama-wakala-api`'s `make self-test` **136/136 across 12
    collections**, its `publish-validation` collection extended to 25
    checks covering the document, a second live channel, and a rollback
    end to end through the real router. The gateway's two Postman gates
    remain at exactly their prior state (29 uncovered of 373, 1 orphan) —
    AG30 removed 8 routes and their 8 Postman requests together, so
    neither number moved.

## Conventions

- **The objects are declared, not written (AGD-007, 2026-09-24).**
  `agency.blueprint.json` declares every object, field and index;
  `agency.agency.json` names what this domain calls them; the engine is
  `mwanachama-backend-shared/spec` and `/specstore`. There are no row
  structs and no `AutoMigrate` — `gormstore/` was deleted. Adding a field
  means editing the blueprint and the `models/` type together, and a
  disagreement between them fails when the manager is built rather than
  dropping a value on every write. The format's reference is
  [declared-domains.md](../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md);
  do not reintroduce the `mwanachama-backend-actor` row-struct shape here.
- **A table is `<instance>_agency_<table>`** — `agency_agency_goals`. A
  mount chooses only the instance, through `SpecFor(instance)`, and
  `Provision(db, spec)` moves a pre-spec database onto the declared names
  before creating what is missing. An instance segment carries no
  underscore and has to leave room for a 63-byte identifier, so a long one
  is refused at startup rather than truncated by Postgres.
- The column every model calls `LastUpdated` is `last_updated`, not
  `updated_at` — the store joins a declared column to a Go field by name.
- `mcp/` no longer hand-writes a tool per manager method (AG41): the
  per-entity tools come from `agency.operations.json` through
  `mwanachama-backend-shared/dispatch`, and only the composites, the type
  glossary, the inverse capability listing and the dashboard are written by
  hand. A tool's **name is declared** (`"tool": "agency_create_goal"`)
  because callers hardcode it, its **arguments are snake_case** because they
  always were, and a **listing is returned as `{"items": [...]}`** because
  MCP defines `structuredContent` as an object and a bare array fails a
  spec-strict client (AG13's live failure). Adding a tool means adding an
  operation; the per-tool conventions in `mwanachama-backend-actor`'s
  CLAUDE.md still describe what a hand-written surface should look like, and
  this repo is no longer an example of one.
- Documentation follows the workspace-wide four-phase layout
  (`1. requirements/2. design/3. implementation/4. qa`) — see
  [documentation/README.md](documentation/README.md).
- Read the source service's `agency.go`, `models.go`, `schema.go`, `drafts.go`,
  `event_flow_steps.go` before porting — they carry the domain logic.
  `raci.go` does NOT implement RACI despite its name (it's WorkPlan/
  ContextSource event-topic routing, stranded under the wrong filename by a
  2026-05-06 rename in the source repo); port it separately from the new
  RACI matrix work, under a name that matches what it does.

## Code comments

Write code with no comments. Not one-liners above a function, not section
banners, not doc comments on exported symbols, not "why" notes next to a
tricky line. A name, a type, or a smaller function carries it instead.

Anything that genuinely needs explaining goes in this repo's `documentation/`
folder, under the phase it belongs to (`1. requirements`, `2. design`,
`3. implementation`, `4. qa`) — never inline.

**Why:** inline prose drifts out of sync with the code, duplicates what
`documentation/` already owns, and buries the explanation where nobody
looking for it will search.

**How to apply:**

- New code ships without comments. If a line seems to need one, rename or
  split until it doesn't.
- Touching code that already has comments: strip the ones in the code you are
  changing. Do not sweep untouched files unless asked.
- If the reasoning matters, add or update the matching `documentation/` page
  in the same change and leave nothing behind in the source.
- Machine-read directives are not comments and stay: build tags, `//go:embed`,
  `//go:generate`, linter pragmas (`//nolint`, `// eslint-disable-next-line`,
  `// ignore:`), license headers, codegen "do not edit" banners, and generated
  files as a whole.
- Commit messages, PR descriptions, and test names carry the narration that
  used to go in comments.

This rule is repeated verbatim in every mwanachama repo's `CLAUDE.md` so that
it reaches sessions that do not load this machine's user-level config —
scheduled cloud routines, other machines, and other agent harnesses.
