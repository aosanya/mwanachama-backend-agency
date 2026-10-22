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

A Module's `Enabled` was already derived rather than stored (AGD-002): an
unlocked Module is on exactly when an Objective in the same scope links it.
`deriveEnabled` (`module_impl.go`) widens that rule: it takes the directly
enabled codes in a scope and closes them over `requires`
(`models.WithRequiredModules`, transitive), so every module a directly
enabled one requires reads as enabled too.

Three consequences follow without any further code:

- **Enabling forms enables both.** Linking an Objective to `forms` makes
  `catalog` and `git` read as enabled in `ListModules`, `GetModule` and the
  publication document.
- **Neither can be disabled while forms is on.** There is no stored flag to
  turn off; removing `catalog`'s or `git`'s own links leaves them enabled for
  as long as anything links `forms`.
- **Importing a template that has forms enables both.** Import writes the
  template's Modules and Objective links; enablement is then read through the
  same derivation.

The derivation always runs over every Module in the scope, before
`ListModules`' code/name/description filters, so a filtered listing reports
the same `Enabled` an unfiltered one does.

Turning `forms` off (its last link removed) turns `catalog` and `git` back
off unless something else enables them.

## Import creates what forms requires (AG81)

Member app decision 250 (owner, 2026-10-22): importing an agency or template
that switches on `forms` creates any forms catalog or report store it lacks.
The derivation above can only report a Module row that exists, so a document
carrying a `forms` Module but no `catalog` or `git` row would otherwise land
with forms on and nothing to hold its answers or its report.

`ImportDocument` (`import.go`) therefore runs `addRequiredModules` after the
document is written into its draft. A Module counts as switched on when the
document marks it `enabled` or an Objective links it; the switched-on codes
are closed over `requires`, and every code in that closure the draft has no
Module row for is created from its catalog entry, enabled, in the same
transaction. A row the document already carries is left as it is, and a
document that switches on nothing that requires anything gains no rows.

`mwanachama-wakala-api`'s `POST /instances/import` does not go through
`ImportDocument`: it seeds the whole module catalog into the new draft before
applying the document, so `catalog` and `git` are already present there and
read as enabled through the derivation as soon as `forms` is linked.
