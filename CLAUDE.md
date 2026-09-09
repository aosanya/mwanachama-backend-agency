# CLAUDE.md

Guidance for Claude Code (claude.ai/code) working in this repository.

## Project: mwanachama-backend-agency

A standalone Go library, sibling to
[`mwanachama-backend-actor`](../mwanachama-backend-actor), modeling an
Agency: departments under one Agency, Goals/Workflows/WorkItems, and a RACI
matrix (Responsible/Accountable/Consulted/Informed) assigning Roles to
WorkItems. Module path `github.com/aosanya/mwanachama-backend-agency` (not
yet scaffolded). Ported from the standalone `CodeValdAgency` gRPC
microservice at `/Users/tony/Agents/workdir/CodeVald-AIProject/CodeValdAgency` —
see [documentation/1. requirements/requirements.md](documentation/1.%20requirements/requirements.md)
for the full decision record from the 2026-09-09 scoping session.

Scoped 2026-09-09. No code yet — this session produced the requirements
record and the initial task board; the actual port from `CodeValdAgency`,
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
- `Draft` ports `CodeValdAgency`'s `AgencyDraft` as a header/pointer type
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
  the domain, like `CodeValdAgency`'s own `agency.go` — not copying actor's
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
  grepped `CodeValdAgency/raci.go` directly and confirmed it's genuinely
  all WorkPlan/`ContextSource` CRUD with zero RACI logic (every
  RACI-labeled hit is in generated proto code) — matching requirements.md's
  original read. AG8 does not need a closer look at it.
- **AG5 landed 2026-09-09: `Department`/`Level`, the org chart.** Net
  new — `CodeValdAgency` has no department concept at all. Modeled on
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

- **AG7 landed 2026-09-09: `SeedNeoLabHQ`, an exported seed function** (not
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

## Conventions

- Follow `mwanachama-backend-actor`'s shape as closely as possible: `models/`
  (one file per domain type), `gormstore/` (row structs, `BeforeCreate` UUID
  minting, `XToRow`/`XFromRow`, `Migrate`/`TableNames`), a `routes/` package,
  and root `doc.go`/`tables.go`/`*_impl.go` files. Read
  `mwanachama-backend-actor/CLAUDE.md` first — it documents these decisions
  in detail (multi-tenancy dropped, recursive tree walks stay Go loops not
  CTEs, `group_impl.go`/`hierarchy_impl.go` as the pattern for this repo's
  department modeling).
- Documentation follows the workspace-wide four-phase layout
  (`1. requirements/2. design/3. implementation/4. qa`) — see
  [documentation/README.md](documentation/README.md).
- Read `CodeValdAgency`'s `agency.go`, `models.go`, `schema.go`, `drafts.go`,
  `event_flow_steps.go` before porting — they carry the domain logic.
  `raci.go` does NOT implement RACI despite its name (it's WorkPlan/
  ContextSource event-topic routing, stranded under the wrong filename by a
  2026-05-06 rename in the source repo); port it separately from the new
  RACI matrix work, under a name that matches what it does.
