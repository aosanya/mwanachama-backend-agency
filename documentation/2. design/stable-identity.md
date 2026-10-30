# A thing's stable identity

`AgencyManager` embeds the shared `identity.Owner` (shared S56), so a host that resolves a manager per instance already holds the owner for it.

`StableIdentity(ctx, object, id)` answers the raw `agency/<instance>/<Code>` for a row, for the objects that mint a Code: `goal` (`G-n`), `objective` (`O-n`), `workflow` (`W-n`), `work_item` (`WI-n`), `role` (`R-n`), `rule` (`RU-n`), `record` (`RC-n`) and `value` (`V-n`). The instance is the spec's instance, the same one that prefixes the tables. The prefix before the dash is unique across types, so a raw names its object. A Code survives a draft fork unchanged (AG16), so the raw is the same for a role in every draft while its row id is not.

`Resolve(ctx, raw)` answers the current `(object, id)`:

- `identity.ErrInvalidRaw` for a string that does not parse;
- `identity.ErrNotOwned` for another module's raw or another instance's;
- `identity.ErrNotFound` for a prefix agency does not mint, a Code with no row, or no open draft;
- `ErrNoStableIdentity` from `StableIdentity` for an object that carries no Code (a draft, an assignment, a join row).

## Open owner question: which draft's row

Every draft carries the same Code (AGD-006: there is no live row), so one raw matches one row per draft. `Resolve` answers the row in the newest open draft (`resolvingDraft`), and a Code absent from that draft is a miss even if an older open draft or a published channel still has it. Whether it should serve the draft a channel publishes, or take the draft from the caller, is undecided; the choice lives in that one method.

Tests: `identity_owner_test.go`.
