# Module mounts in the definition (HS271)

Hub decision 80 (2026-10-29): an agency runs a module as one or more named
mounts, each with its own records. A farmers' group runs Inputs and Harvest
side by side; a school a Uniforms market and a Bus market. This page is the
agency definition's half: where the names live and the rules they keep. The
host's half (building a manager per mount) is `mwanachama-wakala-api` HS272.

## Where the names live

Each draft's `module` row carries `mounts`, a JSON list of
`{name, key, icon}`, and `icon`, the module's default icon key.

- `name` is what the agency calls the mount. Unique within the module, ignoring
  case. One line, no padding, at most 60 characters. It reaches tool text, so
  control characters are refused.
- `key` is the mount the module's records live in, the second argument of
  `SpecForMount(instance, mount)`. A key is assigned by the manager and never
  by the caller. A caller may only repeat a key the module already has, so a
  rename keeps the records and nobody can point a name at another mount's
  tables.
- `icon` is optional and must be one of the keys in `models.ModuleIcons`, the
  set the app and the studio kits both ship (`grid`, `chat`, `bell`, `user`,
  `file`, `calendar`, `clock`, `globe`, `wrench`, `lock`, `search`, `wheat`,
  `bike`, `check`, `alert`, `send`). A mount with no icon takes the module's;
  a module with none shows the first letter of its name, as agency cards and
  chat rooms do.

`modules_catalog.json` keeps its one on/off flag per code and gains three
facts per module: `icon`, `mount` (the key the module runs under today) and
`mountable` (the definition accepts several names). A module with no `mount`
has no named mount at all and accepts none.

## Nothing moves

- A module with nothing stored reads as one mount: the catalog's `mount` as
  its key, the module's name as its name. Existing rows, databases and
  publications are untouched, and a publication of an agency that named
  nothing carries no `mounts`, so its content hash is unchanged.
- The first new entry with no key takes the module's `mount` key when no kept
  entry holds it, so naming the first market Inputs keeps the records already
  in `market`. Later entries get a key made from the name (`Get a ride`
  becomes `get-a-ride`, `-2` on a clash).
- An empty list clears the names and returns to the default mount.
- Dropping a name drops it from the list, not its records. Naming it again
  with the same key reconnects them.

## Surface

`SetModuleMounts(ctx, id, mounts)` writes the whole list on a module in an open
draft. Route `PUT /modules/{module_id}/mounts`, action
`agency.module.set_mounts`, tool `agency_set_module_mounts`. `GetModule` and
`ListModules` return the effective list in `mounts` and the icon in `icon`.
Refusals are `ErrInvalidModuleMounts` (400): a repeated or empty name, a name
over the limit or with control characters, an icon outside the set, a key the
module does not hold, more than 16 entries, or more names than the module can
take.

Mounts ride in the publication document as `modules[].mounts`, enter the
content hash when present, and come back on fork and import. Import refuses a
document whose mounts break these rules.

## Which modules can be mounted by name

Checked 2026-10-29 in each repo and in `mwanachama-wakala-api`'s registry. A
library "takes a name" when its blueprint has `SpecForMount(instance, mount)`.
The shared spec hashes the mount into every table name, and the modules that
write raw SQL name tables through the store, so a second mount gets its own
tables.

| Module | Library takes a name | wakala-api today | Catalog `mountable` | Verdict |
|---|---|---|---|---|
| catalog | yes, `SpecFor(slug, mount)` | `CatalogManager(ctx, id, mount)`, a name is required | yes | Works now. |
| market | yes | `MarketManager(ctx, id)` on the fixed mount `market` | yes | Ready here. Waits on HS272 (host) and HS273 (proof). |
| voting | yes | `PollManager(ctx, id)` on the fixed mount `polls` | no | Needs its own row in wakala-api. |
| taskmanager | yes | `TaskManager(ctx, id)` on the fixed mount `board`; the workflow engine runs on that one board | no | Needs its own row: a board per mount is an engine question. |
| accounting | yes | `LedgerRepository(ctx, id)` on the default mount | no | Needs its own row in wakala-api. |
| engage | yes | not built by the registry at all | no | Needs the host to build it first, then a row. |
| git | no | one repository store per agency | no | Library work first. |
| custody | no | not built by the registry | no | Library work first. |
| digitaltwin | no | not built by the registry | no | Library work first. |
| actor | yes | the one directory, mount `directory` | no | One by nature: the people of an agency. Never. |
| comm | yes | `CommManager(ctx, id)` | no | One by nature: rooms and addresses reach every member. Never. |
| forms | yes | `FormManager(ctx, id)`, and it needs catalog and git | no | One by nature. The catalog beneath it can be named. |
| agency | yes | `AgencyManager(ctx, id)` | no | The definition itself. Never. |
| auth | yes | the global identity | no | One by nature. Never. |

A module that is `mountable: false` but has a `mount` accepts exactly one name,
so an agency can rename it and give it an icon.

## Rows to file

Own row each: voting, taskmanager, accounting (host builds a manager per mount
and flips `mountable`); engage (host builds it first); git, custody and
digitaltwin (library gains `SpecForMount` first). No row for actor, comm,
forms, agency and auth: they are one per agency by nature, and the catalog
under forms is already nameable.

Modules the host mounts that are not in the catalog (`sources`, `assets`, `ai`)
are not switched on by an agency and carry no names here.

## Not settled

- The words an agency sets for each mount (what a poster is called, the unit)
  are the module's settings, not mounts.
- Renaming keeps the key, so a name is a label. Moving records between mounts is
  not offered.
