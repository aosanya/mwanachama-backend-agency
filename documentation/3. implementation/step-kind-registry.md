# Step kind registry

Issue ISS-52. This page records the seam that lets the calculating, posting and holding steps each add their own checks without editing the shared step files again. It builds only the seam. The kinds fill it in other issues: refusing a bad calculating step is ISS-60, the failure choice is ISS-61, the posting and holding steps are ISS-63, carrying their fields through copies is ISS-66 and the scenario warning is ISS-70.

## What it is

One file, `step_kind_registry.go`, holds two registries filled by an `init` in a kind's own file.

| Registry | Key | Holds |
|---|---|---|
| Step kind | a `models.StepKind` | one `StepKindHooks`, each hook optional |
| Issue checks | none | a list of `IssueCheck`, kind-independent, run once per scan |

A kind adds itself with `RegisterStepKind(kind, hooks)`. A kind-independent check adds itself with `RegisterIssueCheck(check)`. A kind with nothing registered behaves exactly as before, and a kind that is registered is also accepted as a valid kind by the shape check.

## The hooks and where each is called

| Hook | Runs | Called from |
|---|---|---|
| `Shape(wi)` | when a step is saved, after every existing shape check; a refusal is wrapped as `ErrInvalidWorkItem` | `validateStepShape` in `workitemstep_impl.go` |
| `References(ctx, lookup, wi)` | when a step is saved, after every existing reference check; `lookup.StepOf` reads a sibling step of the draft | `validateStepReferences` in `workitemstep_impl.go` |
| `Scan(report, wi, steps)` | once per step of the kind in each Issue scan; `report.Add` adds a finding | `checkWorkItems` in `issue_impl.go` |
| `Remap(wi, ids)` | for each step when a draft is forked, after the existing remaps; `ids` holds the workflow, role and step id maps | `forkScope` in `draft_impl.go` |
| `ToDocument(wi, out)` | when the publication document is built | `buildDocument` in `document.go` |
| `FromDocument(in, row, ids)` | when a document is applied to a draft, that is on import and on a fork from a tag | `applyDocumentToDraft` in `document.go` |
| `Hash(wi)` | when the content hash is built; the records it returns are added to the step's records, and a kind with nothing to add returns none | `contentHashFor` in `publication_impl.go` |

The kind-independent checks run once per scan from `checkRegistered`, placed in the scan list after `checkWorkItems`. Each gets the whole list of steps of the draft and the same `report`.

`import.go` has no step code of its own: an import goes through `document.go`, so the import read is `FromDocument`.

## How to add a kind

1. Add the kind to `models.StepKinds()` and its fields to the model, the document and `agency.blueprint.json`, as ISS-48 did.
2. In a new file for the kind, write the hooks the kind needs and register them in an `init` with `RegisterStepKind`.
3. Test with the registry cleared first: `ClearStepKindRegistry` exists in the test build only, so a test registers its own hooks and clears them in a cleanup.

Rules a hook follows:

- A `Hash` hook adds a record only when the step has the kind's fields, so the content hash of an agency that has none of them does not move.
- A hook never edits a field another kind owns.
- `StepHashRecord(type, fields)` builds a hash record in the same form as the shared ones.

## What is checked

`step_kind_registry_test.go` registers a fake kind and shows:

- an unregistered kind is still refused as an unknown kind;
- a registered kind that passes its checks saves, and its shape check and its reference check each refuse a save;
- its finding appears once in the scan, and a kind-independent check runs once per scan;
- a fork runs its remap, the document carries its fields and an import reads them back;
- its hash record changes the content hash, and the same content hashes the same.

The fake kind carries its extra data in fields that already exist on a step, because the fields of the real kinds belong to ISS-48.

## Baseline

With nothing registered every hook is a no-op, so the existing tests are the baseline. Before the change and after it, `go vet ./...` is clean and the suite gives the same results, except for two tests that already failed or flickered before the change and are not touched by it:

| Test | Before and after |
|---|---|
| `TestAG74_OpenHole_AttachEditDetachRewritesAnAgencyWideValue` in `routes` | fails; it records a known open hole |
| `TestLibraryPartTitles_SimultaneousCreatesOfOneTitleStoreOnePart` in the root package | fails in about three runs of four on a clean copy of the previous commit and with the change, a concurrent create against an in-memory database |

A publication hashed before the change cannot be compared with one after it by value, because ids are minted at random in every test. The guarantee is by construction: `hashRecordsOfRegisteredKind` returns nothing for a kind with no hook, so the records of an agency made only of older kinds are the same list as before.

## Open

Nothing is open on the seam. The shapes of the hooks are this page's proposal and can change while no kind uses them.
