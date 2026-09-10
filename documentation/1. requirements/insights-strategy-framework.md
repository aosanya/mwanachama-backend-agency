# Insights — strategy/goal ontology reference

Background research, not a scope decision. Captured 2026-09-10 while
scoping the Goals→Objectives→Measures extension raised against
`mwanachama-wakala-studio`'s draft-canvas UI (its RACI whiteboard sketch
noted "Goals have Objectives, Objectives have Measures" — this repo's
current [`models/goal.go`](../../models/goal.go) has neither). Recorded
here before any model change so the reasoning has a source to point back
to.

## Source

An enterprise-architecture ontology diagram (attributed "© 2020 Henrik von
Scheel"), plus a strategy-scope breakdown supplied alongside it. Both are
one person's framing of how strategy, goals, and execution relate — useful
as a checklist of concepts to weigh against this repo's model, not a
spec to port wholesale.

## Ontology as diagrammed

Five layers, each entity annotated with the framework's own step numbers
(1–6) it participates in:

- **Strategy** — `Enterprise Strategy` *defines* `Organizational Area
  Strategy` *defines* `Organizational Group Strategy`; each also *sets
  direction for* its matching Competency-layer entity
  (`Organization`/`Organizational Area`/`Organizational Group`).
  `Forces (external/internal)` *forces the direction of*, and `Drivers
  (external/internal)` *drives the direction of*, `Organizational Area
  Strategy`.
- **Purpose & Goal** — `Objectives` *realize* `Goals`, which *realize*
  `Organizational Group Strategy`. `Objectives` are *measured by* `Measure
  (KPIs, PPIs & SPIs)`, which is *recorded by* `Report`; `Report` is also
  *designed to* `Objectives` and *record[s] for* `Organizational Group
  Strategy`.
- **Competency** — `Organization` is *composed by* `Organizational Area`,
  composed by `Organizational Group`. `Organizational Function` and `Role`
  *execute*/*perform* work and are *executed by*/*performed by* `Measure`
  and `Business Process`; `Role`'s purpose is *defined by*
  `Organizational Group`.
- **Service** — `Business Service`, *delivered by* `Role` and delivering/
  delivered by `Business Process`.
- **Process** — `Business Process`, *executed by* `Organizational
  Function`/`Role`, *measured by* `Measure`.

Read top-to-bottom, the chain is: **Strategy sets direction → Goals realize
strategy → Objectives realize goals → Measures/Reports track objectives →
Roles and Processes are what actually execute against all of it.**

## Strategy breakdown (supplied alongside the diagram)

A strategy, in this framing, decomposes into three groups:

- **Strategic Positioning** — Understand/Analyze/Develop; Market & Brand;
  Purpose (Mission & Vision); Forecast & Plan; Control & Monitor; Culture;
  Capabilities; Policies/Rules/Guidelines; Strategic Choices (what not to
  do, where to differentiate, what to focus on).
- **Strategy Execution** — Report & Measure; Portfolio/Project/Programs;
  Maturity & Change Management; Best Practices; Services (workflows &
  processes); Execution/Delivery/Processing.
- **Strategy Governance** — Oversight & Audits; Maturity & Change
  Management; Continuous Improvements.

`Mission`/`Vision` sit inside Strategic Positioning here — consistent with
this repo's existing `Agency.Mission`/`Agency.Vision` fields
([`models/agency.go`](../../models/agency.go)) already being adjacent to
`Goal`.

## Relevance to this repo

- `models/goal.go`'s `Goal` is currently flat (`Title`/`Description`/
  `Ordinality`) — it plays the role the diagram splits into two: `Goals`
  (realizing strategy) and `Objectives` (realizing goals, and the thing
  actually measured).
- Nothing in this repo models `Measure`/KPI or `Report` today. The
  whiteboard's "objectives have measures" is this ontology's `Objectives
  measured by Measure` edge.
- `Role` already exists here (AG6, RACI) and lines up with the diagram's
  Competency-layer `Role` — no gap there.
- The diagram's `Business Process`/`Business Service`/`Organizational
  Function` layers have no equivalent in this repo yet; `Workflow`/
  `WorkItem` are this repo's closest existing concepts but were scoped
  (requirements.md decision #6) as declarative plans, not the
  execution/delivery layer the diagram describes. Whether they should
  absorb `Business Process` or stay separate is an open question, not
  decided here.

## Open questions for future scoping

None of the following are decided — they're what a future Goal/Objective/
Measure scoping session needs to answer, now that this reference exists:

1. Does `Goal` split into `Goal` + `Objective` as two linked types, or does
   `Goal` gain an `Objectives []Objective` shape in place?
2. Is `Measure` its own type (with a KPI/PPI/SPI kind field), or a plain
   field on `Objective`?
3. Is `Report` in scope at all yet, or deferred until something actually
   needs to read measures back out?
4. Does any of this interact with the Draft lifecycle (AG4) the way
   `Goal`/`Workflow`/`WorkItem` already do, or is it always-live like
   `Department`/`Level` (AG5)?
