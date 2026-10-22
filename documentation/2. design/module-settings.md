# Module settings in the definition (AG83)

Member app decisions 281 and 282 (2026-10-22): a module's settings values
belong to the agency's definition, so they are edited in a draft, checked at
publish, and published and versioned with everything else.

## Where the values live

Each draft's `module` row carries `settings`, one JSON object keyed by setting
name. Empty means no values were given. `SetModuleSettings(ctx, id, raw)`
writes it on a module in an open draft; anything but a JSON object (or empty /
`null`, which clears it) is refused with `ErrInvalidModuleSettings`. The value
is stored as given, not reformatted.

The values ride in the publication document as `modules[].settings`, enter the
content hash (so a settings-only change is a new version), and come back when a
draft is forked from a publication tag or imported from a document.

## Who checks them

This library does not know any module's settings shape; the module does
(shared's S64, being built separately). So the manager takes the check from the
host:

```go
type SettingsChecker func(ctx context.Context, module string, settings json.RawMessage) []string

mgr, err := NewAgencyManager(db, spec, WithSettingsChecker(check))
```

The issue scan calls the checker once for every **enabled** module in the
scanned draft, with the module's catalog code and its settings (possibly
empty, so a module can report a required value that is missing). Each string
it returns becomes one `error`-severity issue on that module
(`entity_type: "module"`), which blocks publishing like any other error. A
module that is switched off is not asked: nothing will run it.

No checker means no check. A host that has not wired S64 yet publishes
exactly as before.

## Not built here

- No REST route or MCP tool for `SetModuleSettings` yet; the manager method is
  the surface, for the host (wakala-api WK185) to expose.
- The override layer (S64's per-launch values over the definition's) is not
  this library's: only the definition layer is stored here.
