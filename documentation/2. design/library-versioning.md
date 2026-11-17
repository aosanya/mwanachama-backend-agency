# Library versions in a publication

The compute repo's page [library-versioning.md](../../../mwanachama-backend-compute/documentation/2.%20design/library-versioning.md) owns the rules. This page records what the agency backend does about them. The owner decided on 2026-11-17: an agency that already published keeps the version it published with until its builder upgrades, and a fix ships as a new version.

The agency has no formula field yet. This page is the design the agency follows when formulas land. Every name marked proposal below is not decided. All names and figures are invented.

The loan used throughout: 12,000 at 12% a year over 24 months. `fin.pmt@1` gives 564 and `fin.pmt@2` gives 564.88.

## What the publication document carries

A list of library calls, one entry per call, each with three parts:

| Part | Meaning |
|---|---|
| `function` | The library function, such as `fin.pmt`. |
| `version` | The whole-number version, such as `1`. |
| `where` | The place the call sits: a workflow, step or record field, named by its code and not its id, because a fork mints new ids. |

The JSON key is `library_calls`. The key name is a proposal.

- The list enters the content hash only when it is not empty, so a publication with no formula hashes as it did before.
- It comes back on fork and on import. Import refuses a document whose call has no version, or names a place that does not exist in the document.
- A version changed on one call is a change to publish: `ErrNoChangesDetected` is not returned for it.

Lakeside Savings Circle's March publication, 2 calls to `fin.pmt` version 1 (invented):

```json
"library_calls": [
  {"function": "fin.pmt", "version": 1, "where": "workflow:loan-request/step:show-payment"},
  {"function": "fin.pmt", "version": 1, "where": "record:arrears/field:monthly-due"}
]
```

## Saving a formula in a draft

A call that names no version is refused. The error is `ErrLibraryVersionMissing` (400), a proposal. The sentence names the function and lists the versions the library holds:

> `fin.pmt needs a version. Available: fin.pmt@1, fin.pmt@2.`

Hillside Seed Buyers saves `fin.pmt(12000, 0.01, 24)` and is refused. It saves `fin.pmt@2(12000, 0.01, 24)` and the save succeeds.

## What the Issue scan reports

The scan reads the draft's library calls against the library's version list. Like AGD-004, `PublishAgency` is gated on `error` severity only.

| Call | Severity | Sentence | Publish |
|---|---|---|---|
| A version that does not exist | `error` | `Error: fin.pmt@7 does not exist. Latest is fin.pmt@2. Publish refused.` | Refused. |
| A known-wrong version | `warning` | `Warning: this publication calls known-wrong fin.pmt@1 in 1 place. Replacement: fin.pmt@2. Publish anyway or move first.` | Goes through. |
| A version older than the newest | `info` | `fin.pmt@2 is available.` | Goes through. |

The `info` sentence is a proposal: the compute page names the notice but gives its text only on the review, below. A call that is known-wrong is reported once, as the `warning`, and not also as `info`.

Riverbend Market Women published in April with `fin.pmt@1`. In August the library marks it known-wrong. Riverbend publishes a change to an unrelated screen, sees the warning and publishes anyway. Its April publication keeps giving 564.

## The notice on the review

Proposal. A review is never copied into a publication, so the notice is computed when the review is read. It takes the draft's library calls and the library's version list, and nothing about it is stored: no row, no column, no field in the document.

- Each notice carries the release note of the newer version: what changed, for whom it matters, and one differing result.
- A known-wrong version is listed first, above the ordinary notices.
- Each notice offers the move described below.

Example for Lakeside in July:

> `fin.pmt@2` is available. Changed: payment rounded to the nearest 0.01, not down to a whole number. Matters for: any agency that shows or collects a loan payment. Example: 12,000 at 12% a year over 24 months, was 564, now 564.88.

Example for Riverbend in August, listed first:

> `fin.pmt@1` is known wrong. Use `fin.pmt@2`. Loan payments of 12,000 at 12% a year over 24 months are 564, not 564.88.

## Moving to a newer version

One step. It is an edit of the draft that rewrites the version in every call of the function in that draft, and records nothing else. Lakeside's draft holds 2 calls to `fin.pmt@1`; the move makes both `fin.pmt@2`.

The move is published like any other change, so the content hash differs and the result is a new publication with its own tag. No published agency changes until that publish. Lakeside's March publication keeps giving 564 and stays available.

## What the agency never does

- It never rewrites the versions in a published document.
- It never asks the library to edit or remove a version.

## Open for the owner

Can a known-wrong version ever be made to stop running (for example when a result is unsafe), and who may decide that?

This is not decided here. Until it is, a known-wrong version keeps running and only warns.
