# Hub copy diff

`mwanachama-backend-agency` is the custodian of the agency module. The hub (`projects/hub-steward/code/wakala-api/modules/agency`) holds a copy made on 2026-11-04 and changed it afterwards. Every change made in that copy exists here, so the two differ by the import path only (hub issue AG115, the agency half of ISS-105).

## Command

```
scripts/hub-copy-diff.sh [path-to-hub-modules-agency]
```

The default path is `../projects/hub-steward/code/wakala-api/modules/agency`, relative to this repository's parent folder. The exit code is 1 when the trees differ, 0 when they do not; test files only here keep it at 1. The script and `scripts/hubdiff` are the ones of `mwanachama-backend-shared`, adapted to this module's package folders.

## What it does

1. Copies the `*.go` and `*.json` files of the hub folder and of this repository, at the top level and in `mcp`, `models` and `routes`, into two temporary folders. `scripts`, `testdata` and `.git` are left out.
2. Rewrites the hub's imports: `wakalaserver/modules/shared/` becomes `github.com/aosanya/mwanachama-backend-shared/` and `wakalaserver/modules/agency` becomes `github.com/aosanya/mwanachama-backend-agency`.
3. Runs every Go file on both sides through `scripts/hubdiff`, which parses it, drops all comments and prints it with `gofmt`, so comments and layout do not show as differences.
4. Runs `diff -r backend hub`.

## Result of 2026-11-16

Every non-test file is identical after the rewrite, JSON included, and no file exists only in the hub. What remains is `Only in backend` lines, all test files, and one `diff`, in a test file:

| Difference | Reason |
|---|---|
| Only here: test files the hub copy never carried (229 lines) | The hub copy of 2026-11-04 was made with a subset of the tests. The repository kept its own. They test code that is identical in both. |
| `routes/review_author_test.go` holds `TestARecordRowIsSignedByTheCallerNotTheBody` here | The red test written for hub 6422ddbe (the author of a record row is the caller); the hub commit tested it through the API. |
| Only here: `record_row_send_key_test.go`, `names_plaintext_test.go`, `kept_codes_test.go`, `record_row_stamp_prompts_test.go`, `workitem_role_filter_test.go`, the `models` catalog tests | The red tests written for hub 743a64ef, da99df0a and 98579506, d6ab2174, dde9355c 7836db1d cbb613e6, 12a4290d and the catalog commits; the hub commits tested them through the API or not at all. |

## The hub commits carried over

One commit here per hub commit group, each naming its hub hashes: 73c451de, 53b12775, f11b6fef (catalog group C); 5fc39fa7, bda4d379, f837351c, d135d26d, f19ca9a9, 8478cb43, ee60ebae, 5063fbe7 (group E); dde9355c, 7836db1d, cbb613e6 (approval limit and stamp); 15f49936 (routes cover tests); 5018e13b, 4446bfd0, 0eaa3c45, b1c21fed, a8a7efab (group D); d6ab2174 (kept codes); da99df0a, 98579506 (plaintext names); 743a64ef (send key); 3d11d3b8 (duplicate row per member); 12a4290d (role filter); 6422ddbe, 8899379e (signed by the caller); 20319cd1, ebd187e0 (post kinds); d571f6c4 (mcp cover tests); eb7cd7eb, 85246ae2, e12d5112, e8cbd7c4 (catalog group B); b2d0ca7d (partial role update); c80433b5, d96ca8dd, f09856e0 (task permissions); 83963ee8, 36cebd57, d0ddefd7, 92c070dd, e6728d3e (catalog group A). The hub commit a7d64d10 is the copy-in itself.

The functions a commit touched lost their comments (the repository rule is no comments in code); comments in code a commit did not touch remain until that code is next changed. The two catalog JSON files of the last group were taken whole from the hub after the diff showed only those commits' lines, because this repository already carried its own `assetmanager` description. The hub's wording is the one kept.

## Failures present before this work

Three tests fail on a clean export of this repository at 4e47fa0, before any hub change, and still fail: `TestUpdateWorkflow_TheConsolesBodyKeepsItsOrdinalityAndEventFlows`, `TestUpdateWorkItem_TheConsolesBodyKeepsItsTool` and, in `routes`, `TestAG74_OpenHole_AttachEditDetachRewritesAnAgencyWideValue`. None is touched by a hub commit.

## Changes made here that the hub copy does not have

- Library parts created or renamed at the same moment (hub issue MUN698): `agency.go`, `library_part_impl.go`, `consequence_impl.go`; see `library-title-concurrency.md`.
- A table that lacks a declared index is given it by `Provision` (hub issue MUN317): `provision.go`; see `provision-index-check.md`.
