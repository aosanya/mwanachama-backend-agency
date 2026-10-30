# A module can require others (AG80)

Member app decision 213 (owner, 2026-10-21) is a platform rule for every
agency and every template: **enabling `forms` enables `catalog` and `git`**,
because a form's answers land as records in a per-agency forms catalog and
its report is versioned in git.

## Where it is declared

`models/modules_catalog.json` gives a catalog entry an optional `requires`
list of other catalog codes. `forms` is the only entry carrying one today:
`"requires": ["catalog", "git"]`. The catalog refuses at package init a
`requires` naming a code it does not hold.

## How it is enforced

A Module reads as enabled when it is locked, or its stored flag is on, or a
workflow step in the same draft needs it (AGD-008). `withEffectiveEnabled`
(`module_impl.go`) then closes that set over `requires`
(`models.WithRequiredModules`, transitive), so every module a switched-on one
requires reads as enabled too.

Three consequences follow without any further code:

- **Switching forms on enables both.** `catalog` and `git` read as enabled in
  `ListModules`, `GetModule` and the publication document, with their own
  `explicit` flag still off.
- **Neither can be disabled while forms is on.** Switching `catalog`'s or
  `git`'s own flag off leaves them enabled for as long as forms is.
- **Importing a template that has forms enables both.** Import writes the
  template's Modules with their stored flag; enablement is then read through
  the same rule.

The rule always runs over every Module in the scope, before `ListModules`'
code/name/description filters, so a filtered listing reports the same
`enabled` an unfiltered one does.

Switching `forms` off (with no step needing it) turns `catalog` and `git` back
off unless something else enables them.

## Import creates what forms requires (AG81)

Member app decision 250 (owner, 2026-10-22): importing an agency or template
that switches on `forms` creates any forms catalog or report store it lacks.
The rule above can only report a Module row that exists, so a document
carrying a `forms` Module but no `catalog` or `git` row would otherwise land
with forms on and nothing to hold its answers or its report.

`ImportDocument` (`import.go`) therefore runs `addRequiredModules` after the
document is written into its draft. A Module counts as switched on when it reads
as enabled by the rule above (the document's flag, a step in the document, or
a lock); the switched-on codes
are closed over `requires`, and every code in that closure the draft has no
Module row for is created from its catalog entry, enabled, in the same
transaction. A row the document already carries is left as it is, and a
document that switches on nothing that requires anything gains no rows.

`mwanachama-wakala-api`'s `POST /instances/import` does not go through
`ImportDocument`: it seeds the whole module catalog into the new draft before
applying the document, so `catalog` and `git` are already present there and
read as enabled through the rule as soon as `forms` is switched on.
