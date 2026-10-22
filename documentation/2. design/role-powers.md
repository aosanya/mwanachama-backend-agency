# Role powers (AG82)

Member app decisions 263, 265, 267 and 269 (2026-10-22): what each role can
do is a list of named powers, declared in the agency definition, drafted and
published with the rest of its design.

## Where it lives

- `role.powers` (`agency.blueprint.json`, `models.Role.Powers`) is a JSON
  list of power codes. Empty declares none; an agency whose roles declare
  none keeps today's behaviour (decision 267). It rides the role's create and
  update bodies unchanged, so routes and MCP tools carry it with no new
  address.
- The vocabulary is one declared place: `models/powers_catalog.json`, read
  through `models.PowerCatalog()` and `models.LookupPower(code)`. Each entry
  has a `code`, a display `name` and `locked`.
- `models.DocumentRole.Powers` carries the list into the publication
  document, and forking a draft from a publication restores it, so powers
  version with the draft. The role list in the document is what wakala-api
  turns into held role kinds (decision 269, WK181).

## What the publish scan refuses

`checkPowers` in the Issue scan raises an `error` (so publishing is refused
with `ErrPublishValidationFailed`) when a role declares:

- a code not in the vocabulary, or
- a power locked to owners (decision 265). Locked powers are held by owners
  whatever the definition says, so no role may claim one.

## The vocabulary, and what is not yet settled

Decision 263 names powers by example ("publish a report, approve forms,
decide join requests, invite, record a promise's outcome, …") and decision
265 names the locked owner powers. The catalog holds exactly those, with
codes chosen here:

| Code | Power | Locked |
| --- | --- | --- |
| `report.publish` | Publish a report | no |
| `form.approve` | Approve forms | no |
| `join_request.decide` | Decide join requests | no |
| `member.invite` | Invite | no |
| `promise.record_outcome` | Record a promise's outcome | no |
| `agent.manage` | Add or remove agents | yes |
| `api_key.manage` | Manage API keys | yes |
| `agency.go_live` | Go live | yes |
| `agency.delete` | Delete the agency | yes |
| `power.override` | Change overrides on the powers page | yes |

Open: the full list (the "…" in decision 263) and whether these codes match
what permissions PM21 and wakala-api WK179 will enforce. The catalog is the
one file to change when that is settled. The owner, agent and member columns
of decision 264 are not agency roles and are not modelled here.
