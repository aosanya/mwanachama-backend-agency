# Voting in the definition (AG84)

Member app decisions 341–343 (2026-10-22), on the shapes AG82 (powers,
[role-powers.md](role-powers.md)) and AG83 (module settings,
[module-settings.md](module-settings.md)) set. Nothing new was built for
voting: it is one entry in each of the two catalogs.

## On or off

`voting` is an entry in `models/modules_catalog.json`, off by default and not
locked, so every draft carries a voting module row and the definition switches
it on the way it switches any module on (linking it to an Objective). Switched
off, the host does not mount it (decision 284). It requires no other module.

## Default poll settings

The defaults live in the voting module row's `settings` object, published,
hashed and forked like any module's. The keys are the ones
`mwanachama-backend-voting` declares in its blueprint's `settings` block:
`mode` (`one`, `up_to`, `quadratic`), `max_choices`, `budget`,
`change_allowed` and `per_group` (`once`, `per_group`). Agency does not know
those keys: the host's `SettingsChecker` asks voting whether the values are
valid, so a refused value is a blocking publish issue. A key left out takes
voting's own default.

## Run a poll

`poll.run` ("Run a poll") is an unlocked entry in `models/powers_catalog.json`,
ticked per role in `role.powers` (decision 342). It covers creating, closing
and publishing polls. That owners and agents hold it when no role does
(decision 270), and that a leader's power reaches the groups inside theirs
(decision 266), are the host's and permissions' to apply (PM23, WK207), not
this module's.
