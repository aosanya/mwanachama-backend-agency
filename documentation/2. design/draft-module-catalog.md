# A draft started from nothing holds the module catalog

Member app decision 464, board row AG91.

Provisioning seeds every entry of `models.ModuleCatalog` into an agency's
first working draft. A draft created with `CreateDraft(ctx, description, "",
"")` copies nothing, because it has no source to fork from, and before AG91 it
held no modules either. A module is switched on by setting the flag on the
draft's own copy of it (or by a workflow step that needs it, AGD-008), and an
unpublished agency is gated on its modules, so such a draft could never
switch one on.

`CreateDraft` with no source now seeds the catalog into the new draft through
`seedModuleCatalog`, the same function provisioning's `backfillModules` runs,
inside the draft's own transaction. A draft forked from another draft or from
a publication is unchanged: it copies the modules its source holds.

## What a caller does differently

The catalog codes already exist in a fresh draft, so `CreateModule` with one
of them answers `ErrDuplicateModule`. A caller that used to create the
catalog itself reads the seeded rows with `ListModules` instead.
`SeedSWELab` does so, and the module tests read the seeded row rather than
creating it. A code outside the catalog is still created with
`CreateModule`.

`mwanachama-wakala-api`'s import (`instance_import.go`'s `seedModules`) still
creates every catalog module in the draft it starts from nothing, and needs
the same change there.
