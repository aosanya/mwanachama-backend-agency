// Package mwanachamaagency models an Agency as a declarative document — an
// org chart, a strategy (Goals/Objectives, Values/Rules), and a RACI
// matrix — rather than a live operational store.
//
// Its objects are declared, not written: agency.blueprint.json names every
// object, field and index, agency.agency.json names what this domain calls
// them, and the storage engine is mwanachama-backend-shared/spec and
// /specstore. See documentation/2. design/todo_details/AGD-007.md for the
// conversion record and mwanachama-backend-shared/documentation/2. design/
// declared-domains.md for the format.
//
// Layout:
//   - agency.blueprint.json — the module's objects, declared once
//   - agency.agency.json    — this domain's own names for them
//   - models/               — the Go types those objects are carried in
//   - blueprint.go, provision.go, store.go — loading, provisioning, storage
//   - *_impl.go             — the behaviour, which is all that is hand-written
//   - routes/, mcp/         — the HTTP and MCP surfaces
package mwanachamaagency
