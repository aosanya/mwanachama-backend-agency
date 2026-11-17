# What the host and the accounting repo must do for held figures

This repo stores and scans the hold; it does not hold, rework or post. The host
(wakala-api, the service that runs agency workflows) writes and moves the held
figure, and the accounting repo keeps the ledger. Design:
[held-figures.md](../2.%20design/held-figures.md) (ISS-22). Cases:
[held-figures-cases.md](../4.%20qa/held-figures-cases.md), H1 to H11. What the
accounting repo builds today: [accounting-ledger-audit.md](accounting-ledger-audit.md).

Provisional: the calculating step ([compute-step.md](../2.%20design/compute-step.md)),
its failure choice ([compute-step-failure.md](../2.%20design/compute-step-failure.md)),
the posting step ([posting-step.md](../2.%20design/posting-step.md)) and every name
the design page proposes are not accepted, so each sentence leaning on one contains
that word. This page lists the work; it files no issue. The lead files them after
the owner has seen the list.

Example throughout: Lakeside Savings Circle, the workflow Quick loan interest, and
Wanjiru Kamau, who owes 3,500.00, pays 200.00 late and is approved at 132.50
instead of the 140.00 held on Monday.

## What this repo stores and scans

1. Provisional: `hold` on a `compute` step (`{"expires_after_days":14}`) and
   `held_step_id` on a `yes_no` or `post` step, in `agency.blueprint.json`,
   `models.WorkItem` and `models.DocumentWorkItem`, left out when empty (design
   section 3).
2. The save refusals of section 7: cases H7, H8, H9.
3. The Issue scan errors of section 8, so a hold nothing can release is not
   published: cases H6, H11j, H11k.
4. Fork remaps `held_step_id`; a publication carries both fields; the hash gains
   `WorkItemHold` only for a step that has one; an older document imports with
   nothing invented (section 10): cases H11a to H11k.
5. Provisional: if the owner answers open question 1 as proposed, the held row is
   this repo's object `held_figure`, with the operations of section 9 and the four
   states of section 2. If not, none of that is stored here.

## What wakala-api does

What it does today is not known beyond a search. Searching the `*.go` files of
`mwanachama-wakala-api` found no `held_figure`, `HeldFigure`, `held_step_id` or
`WorkItemHold`, and no step kind `compute` or `post` (the one `"post"` hit,
`internal/testruns/endpoint.go`, is a market tool mapping). Provisional: what the
host does at a calculating step today is not shown by the repos, and this page does
not say.

1. Provisional: when a case reaches a `compute` step with a `hold`, write the held
   row in state `held` (design section 1) and keep nothing under `result_name`.
   Cases H1, H2.
2. Provisional: make the posting step wait while the figure is `held`: nothing
   reaches the ledger or the member's figures. Cases H1 to H4.
3. Provisional: list the figure in the releasing role's open steps with its inputs
   and the days left.
4. When the approver opens it, compare the fingerprint of the inputs now with
   `inputs_fingerprint`. Same: show it, release it unchanged. Different: call the
   compute repo with the function and version held, never a newer one, and today's
   inputs; show both figures and one sentence for each input that moved. Cases H1,
   H2. Provisional: how the fingerprint is worked out is open.
5. Approve by fingerprint. Refuse the old figure with `ErrHeldFigureMoved` (409),
   also when inputs move between opening and pressing approve, and answer with the
   new figure and the reason. Case H2.
6. On approve, record the rework if there is one, move the row to `released` and
   post once through the posting step, with the function, version and fingerprint
   of the figure approved and the note "Reworked at approval; held 140.00 on
   Monday". Cases H1, H2. A figure that is not money goes to the record of figures
   (open question 7).
7. Send back needs a reason (`ErrInvalidHeldFigure`, 400, if blank). The row becomes
   `rejected` and is kept; nothing is posted. Case H3.
8. Apply expiry on every read and move: a `held` row past `expires_at` becomes
   `expired` and is kept, and approve and send back return `ErrHeldFigureExpired`
   (409). No timer. Case H4. Open question 2.
9. Refuse a caller without the releasing step's role or offered group with
   `ErrHeldFigureRefused` (403). Case H10. Open question 3.
10. List one step's held figures in one run, 100 to a page, at most 200, totals
    first. Release in batches of at most 500, one member at a time, one outcome each
    (`released`, `moved`, `already_released`, `refused`). A moved figure stays `held`
    until a second call names the new fingerprint. Case H5. Open question 9.
11. Restart safely: releasing a `released` figure returns the first result and writes
    nothing, and the send key keeps the ledger to one entry, so a batch cut off after
    1,000 is sent again whole. Case H5. Provisional: the key's shape is open.
12. Provisional: when the rework fails, follow the step's failure choice, leave the
    old figure not approvable and the row `held` (open question 4).
13. Keep `hold` and `held_step_id` when the reviewer renames steps
    (`reviewruns/workflows.go`). Provisional: whether it keeps unknown fields today
    is not checked here.

The host must not post a figure whose inputs have moved, recompute with a newer
version (open question 10), edit `figure` or `inputs`, or delete a `rejected` or
`expired` row.

## What the accounting repo must do

The audit's gaps, as the accounting repo's work. None is made or filed here.

1. Provisional: carry the function code, version, fingerprint and the rework note on
   an entry. Audit gap 3, cases H1, H2.
2. Provisional: a send key with a unique index, so a repeated release writes one
   entry. Audit gap 4, case H5.
3. Provisional: keep amounts to 2 places (an entry's amount is a whole number today)
   and give a balance for each day, which `balance_by_day` needs (design open
   question 6; posting-step.md section 12, questions 5 and 8).

## Not here

- The posting step's work: [posting-step-host.md](posting-step-host.md).
- The calculating step's work: [compute-step-host.md](compute-step-host.md).
- The formula language and `loan.daily_interest`: the compute repo.
