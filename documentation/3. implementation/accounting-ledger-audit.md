# Accounting ledger audit

This page lists what `mwanachama-backend-accounting` actually builds of its ledger, judged from the code and its tests and not from the requirements. Each row is built, partly built or missing, with the file that shows it (read on 2026-11-17).

| Item | Status | Evidence | What the code does |
|---|---|---|---|
| 1. Accounts | built | `ledger_store.go` `OpenAccount`, `GetAccount`, `ListAccounts`, `CloseAccount`; test `TestOpenAccountIsLazyAndIdempotent` in `ledger_test.go` | An account has a category, one of five types and a holder id. It opens on first use, no operation edits it except closing, and cannot close while its balance is not zero (`ErrNonZeroBalance`). |
| 2. Entries, double-entry, append-only | built | `ledger_store.go` `postIn`; `ledger.go` `checkEntryBeyondSpec`; `accounting.operations.json` (no update or delete operation); every entry field `immutable` in `accounting.blueprint.json`; test `TestEntryIsAppendOnlyCorrectedByReversal` | An entry debits exactly one account and credits exactly one other for a positive amount, and both accounts must exist and be open. Nothing in the module changes or removes an entry. The database itself does not block an update, only the module's missing code path does. |
| 3. Reversal by `reverses_entry_id` | partly built | `ledger_store.go` `postIn`; unique index `reverses_once` in `provision.go`; tests `TestPostRefusesASecondReversalOfTheSameEntry`, `TestPostRefusesUnknownReversesEntryID`, `TestAReversalKeepsItsOriginalsDimensionsSoADimensionBalanceNetsToZero` | A reversal must name an existing entry, can be made once (`ErrAlreadyReversed`, backed by a database index) and copies its original's dimensions. Nothing checks that it swaps the two accounts or repeats the amount, so a "reversal" of 500 for 50 in the same direction is accepted. |
| 4. Balance as a fold | built | `ledger.go` `balanceOf`; `ledger_store.go` `balanceIn`, `Balance`; tests `TestBalanceOfDebitsAddAndCreditsSubtract`, `TestPostingMovesTheBalancesOfBothAccounts` | Balance is debits minus credits over every entry touching the account, computed on each read. It loads all of the account's entries into Go and adds them there. |
| 5. One account per member | partly built | `accounting.blueprint.json` unique index `identity` on `category` plus `holder_id`; `ledger_store.go` `accountByIdentity`; tests `TestOpenAccountIsLazyAndIdempotent`, `TestOpenAccountKeysOnCategoryAndHolderTogether` | A second open for the same category and holder returns the first account. "Member" is not a concept here: `holder_id` is an opaque string and `category` is open, so one account per member holds only if the caller always opens members under one category. No test uses a member. |
| Extra: provenance (function code, published version, input fingerprint) | missing | `models/ledger.go` `Entry` | None of the three exists. The nearest are `document_kind`, `document_id` and `actor_id`, which say what document and who caused the posting, not which function, version or input. |
| Extra: idempotency or send key | partly built | `ledger_store.go` `entryOfSameDocumentLine`; tests in `ledger_replay_test.go` | There is no key field. A repeat is recognised when document kind, document id, both accounts, amount, `reverses_entry_id` and `settles_entry_id` all match, and the first entry is returned. It is a read inside the transaction, not a unique index. |
| Extra: direction (debit or credit) | built | `models/ledger.go` `Entry.DebitAccountID`, `CreditAccountID`; `checkEntryBeyondSpec` | Direction is which account sits on which side. The amount is never signed. |
| Extra: reason text | partly built | `models/ledger.go` `Entry.Narration`; blueprint `narration` | One optional free-text field. It is not required, not even on a reversal, and only the legacy-adoption test reads one back. |
| Extra: stored balance that could drift | none found | `models/ledger.go` (`Account`, `Entry`, `Budget`), `accounting.blueprint.json` | No balance field exists on any object. `Budget.Amount` is a target, and `Variance` is computed on read. |

## Gaps

1. A reversal is not checked against its original: accounts not swapped, amount not equal. Suggested issue: "Refuse a reversal that does not mirror its original entry".
2. "One account per member" depends on every caller using one category for members. Suggested issue: "Fix the member account category and test one account per member".
3. An entry carries no function code, published version or input fingerprint. Suggested issue: "Carry function code, published version and input fingerprint on an entry".
4. There is no send key, and the replay match is a lookup without a unique index. Two real payments of 50 against a balance of 500, on the same document, accounts and settled charge, collapse into one. Suggested issue: "Add a send key to an entry with a unique index".
5. The reason on an entry is optional, including on a reversal. Suggested issue: "Require a reason on a reversal".
6. The balance loads every entry of the account into Go instead of summing in the database. Suggested issue: "Fold the account balance in the database".

## Tests run

`go test -count=1 ./...` in `mwanachama-backend-accounting`, in-memory SQLite only, no Postgres needed: the root package and `routes` passed, `models` and `scripts/hubdiff` have no tests. `git status` there was clean before and after.
