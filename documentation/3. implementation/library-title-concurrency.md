# Library parts created or renamed at the same moment

Hub issue MUN698 (found by MUN234b). Hazards, threats and consequences are the draft's library parts, and a part's title is free once per draft (once per hazard for a consequence).

## Cause

`createPart` and `CreateConsequence` read the draft's rows to see whether the title was taken, then inserted, as two steps with nothing between them holding the answer. Two creates at the same moment both read "free" and both inserted. The same pair one after the other gave 201 then 409. The rename path (`updatePart`, `UpdateConsequence`) read inside a transaction that started as a read and then wrote, so on SQLite two renames at the same moment answered the raw `database is locked` error (SQLITE_BUSY) instead of one conflict.

No unique index can carry the rule: titles are compared through `plaintext.SameNameIgnoringCase`, which folds case and Unicode form, and an index compares bytes.

## Rule

Two creates, two renames, or a create and a rename to one title at the same moment behave like the same calls one after the other: one succeeds and every other answers `ErrLibraryTitleTaken` (409) naming the title. Nobody sees a lock error.

## How

- One mutex per manager, `libraryTitles`, is held across the title check and the write in `createPart`, `updatePart`, `CreateConsequence` and `UpdateConsequence`. The manager is pooled per agency in the API, so every request of one agency passes through the same mutex.
- A unique violation or busy answer from the write is still mapped to `ErrConflict` by `classifyConflict`.

## Limit

The mutex holds inside one server process. Two server processes on one database can still race on a title. Closing that needs a database-side lock (on Postgres `pg_advisory_xact_lock` on the draft and kind, as the custody consent publish takes) and was not built here because it cannot be exercised without a Postgres database; it is an owner question.

## Proof

`library_part_concurrent_test.go` sends 8 creates of one title at once, 12 rounds, for each of hazard, threat and consequence; 8 renames of different parts to one title, 12 rounds; and a create and a rename to one title, 12 rounds, on a SQLite file in WAL mode with 8 connections. Exactly one call succeeds each round, every other answers `ErrLibraryTitleTaken` or `ErrConflict`, and the list grows by one. Before the fix the creates stored two parts and the renames answered `database is locked`.

## A flake that was in the test, not the store

The consequence case of the simultaneous creates failed about one run in five with `a consequence needs a hazard_code`. The store was right. A consequence needs a hazard, the test helper `sharedHazard` in `library_parts_test.go` makes one titled "Shared hazard" when the draft has none, and in that case all 8 callers made it at the same moment: one won, seven were refused with the title conflict, and the old helper swallowed the refusal and returned an empty code, so seven consequences were created with no hazard.

`sharedHazard` now waits, for at most 5 seconds, until the winner's hazard is listed and returns its code, so every simultaneous caller gets the one code. `TestSharedHazard_EightCallersOnADraftWithNoHazardAllGetTheOneCode` fails against the old helper in 9 of 10 runs and passes 30 of 30 now, and the simultaneous creates test passes 30 of 30. A test that creates a part which needs another part should seed that part before the callers start, or use this helper.

## Hub copy

The hub copy (`projects/hub-steward/code/wakala-api/modules/agency`) does not have this change and needs it until ISS-105 deletes the copy. Until then `scripts/hub-copy-diff.sh` shows differences in `agency.go`, `library_part_impl.go` and `consequence_impl.go`.
