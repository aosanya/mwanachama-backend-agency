// Package mwanachamaagency models an Agency as a declarative document — an
// org chart, a strategy (Mission/Vision/Goals), and a RACI matrix — rather
// than a live operational store. Ported from the standalone CodeValdAgency
// gRPC/ArangoDB microservice onto GORM-backed relational storage,
// following mwanachama-backend-actor's shape as closely as possible. See
// documentation/1. requirements/requirements.md for the full decision
// record and documentation/3. implementation/todo.md for build status.
//
// Layout so far:
//   - models/    — domain types (Agency, Draft, Goal, Workflow, WorkItem)
//   - gormstore/ — GORM row structs, row<->domain conversion, migration
//   - doc.go (this file), tables.go — table-name/migrate wrappers
//
// No CRUD manager or edit-lifecycle behavior exists yet — see
// documentation/3. implementation/todo.md for what's next.
package mwanachamaagency
