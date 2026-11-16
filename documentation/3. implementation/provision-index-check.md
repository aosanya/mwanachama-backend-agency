# Provision gives back a declared index an existing table lacks

Hub issue MUN317 (found by MUN302 on Postgres 17). The once-per-run send rule (hub MUN169) is the unique index `send_key_unique` on `record_row`.

## Cause

`Provision` skips `spec.Migrate` when `upToDate` says the database is ready. `upToDate` checked the tables, the columns and the indexes of `workflow_objective` only. `spec.migrateTable` adds a column in its own transaction before it creates the indexes, so a stop between the two (a crash, a killed pod, a lock timeout, a failed `CREATE UNIQUE INDEX`) leaves the column and no index, and every later `Provision` answered nil without building it. Two API processes could then both send one step per run, with nothing to say so.

## Rule

`upToDate` asks `indexesHeld` for every declared object, not for `workflow_objective` alone. A table that lacks any declared index sends `Provision` through `spec.Migrate`, whose `create index if not exists` builds it. A database that holds every index still runs no migration: `TestProvision_ASecondProvisionOfAnUpToDateDatabaseRunsNoMigration` counts the create statements of a second `Provision` and expects none.

## Rows that already break the index

Rows holding one non-empty `send_key` more than once cannot take the unique index. `Provision` now returns that error on every call, so the agency answers 500 on every request and says why, where it used to be quietly without the rule. This change does not touch stored rows. Choosing what to do with the duplicates (blank the later keys, retire the later rows, or repair by hand) is the owner's: it changes stored member records. It is a Blocked todo on the hub issue.

## Proof

`provision_send_key_index_test.go`: dropping only `send_key_unique` and provisioning again gives the index back; with two rows of one key and no index, `Provision` answers an error on both attempts; a second `Provision` of an up-to-date database runs no create statement. The first two fail on the master before the change.

## Hub copy

The hub copy (`projects/hub-steward/code/wakala-api/modules/agency`) does not have this change and needs it in `provision.go` until ISS-105 deletes the copy.
