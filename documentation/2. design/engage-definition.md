# Engage in the definition (AG85)

Member app decisions 346–348, on the shapes AG82 (role permissions) and AG83
(module settings) built.

## The module

`engage` is an entry in `models/modules_catalog.json`, switched off and not
locked, like every optional module. Every Draft is seeded with it, the
operator turns it on or off there, and the switch rides the publication
document's `modules` with the rest of the definition (decision 281). It
declares no settings and no `requires`.

## The posting permissions

Decision 348 makes posting a permission per kind, ticked per role. The vocabulary
gains one permission per kind of `engage`'s `post.kind`:

| Code | Permission | Locked |
| --- | --- | --- |
| `notice.post` | Post a notice | no |
| `event.post` | Post an event or meeting | no |
| `volunteer_call.post` | Post a volunteer call | no |

Each carries `"module": "engage"` (`models.Permission.Module`), naming the module
the permission belongs to. A role declares them in `role.permissions`; the Issue scan
refuses nothing new, and they publish and fork with the role (decision 267).

## Left open

- Whether a role may declare an engage permission while engage is switched off.
  Today the scan accepts it; the host simply has nothing mounted to gate.
- Whether engage requires `comm`, since replies are threads in a group's room
  (decision 349). Not declared until the owner says so.
- Whether these codes match what wakala-api and permissions will enforce
  (the same question AG82 left for its own codes). Decision 270's owner and
  agent fallback is the host's, not modelled here.
