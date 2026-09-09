# mwanachama-backend-agency

A standalone Go domain library, sibling to
[mwanachama-backend-actor](../mwanachama-backend-actor), modeling an
**Agency**: departments (e.g. a software team and a mechanical team) under
one Agency, Goals/Workflows/WorkItems, and a real RACI matrix
(Responsible/Accountable/Consulted/Informed) assigning Roles to WorkItems.
Ported from the standalone `CodeValdAgency` gRPC microservice — as a plain
Go library imported by the gateway, not a network service of its own,
following `mwanachama-backend-actor`'s GORM-backed shape as closely as
possible.

Status: scoped, not yet built. See
[documentation/1. requirements/requirements.md](documentation/1.%20requirements/requirements.md)
for the decisions made so far and what's still open.
