# Compute settings of an agency

Status: written for ISS-53 and not yet reviewed by the owner. A sentence that starts with "Decided" is in the decisions pages. A sentence that starts with "Proposed" is a starting point and is not decided. "Recommendation" is the page's advice on an open question and decides nothing. All names and figures are invented.

End goal: the owner can see in one page where each agency chooses its currency's decimal places, its rounding rule, the digits kept in a division that does not end and the most rows one batch may hold, who may change them and what happens to a publication that already exists when one changes.

The engine-side rules are decisions 13 to 21 in `mwanachama-backend-compute/documentation/1. requirements/decisions.md`. The steps-side rules are in [compute-decisions.md](../1.%20requirements/compute-decisions.md). The library takes the first three settings as values it is given and makes no choice for an agency (`mwanachama-backend-compute/documentation/3. implementation/numbers.md`). The fourth is checked by the host before a batch starts ([compute-step.md](compute-step.md), section 6).

## 1. The four settings

| Setting | Source decision | What it controls | Proposed default |
|---|---|---|---|
| Decimal places of the currency | 15 | How many places every amount, rounding and share has | 2 |
| Rounding rule | 14, 20 | How a fraction is rounded where a formula names no rule, and the list the builder is offered | half up |
| Digits kept in a division | 21 | The digits kept inside a calculation before the final rounding, when a division does not end | 20 |
| Rows one batch may hold | 16, 17 | The largest calculation batch; a batch over it is refused whole before any row is worked out | 10,000 |

Decided: each of the four is set by the agency and not by the library. For the rounding rule a formula may still name its own rule where it rounds a result (decisions 14 and 20); the agency's value is the rule where it names none. The defaults are proposed, except that half up is the decided rule where none is chosen (decision 14).

## 2. Where they live

Proposed: all four live in one object, `compute`, in the agency's definition, and the object is stored as the `settings` of one module row in the draft, the way any module's settings are stored ([module-settings.md](module-settings.md)).

Why that place: the nearest existing home for an agency's setting already does what these four need.

- The values are edited in a draft, checked by the Issue scan at publish, and published and versioned with everything else.
- They ride in the publication document and enter the content hash, so a settings-only change is a new version.
- A draft forked from a publication tag comes back with them.

What would have to be built (not built here): a catalog entry for a `compute` module that is locked and always on, so an agency can never be left without values, and a checker that returns one plain sentence for each wrong value. The checker is the host's, as it is for every module ([module-settings.md](module-settings.md), "Who checks them").

Alternative, not proposed: fields on the agency row itself. It avoids a module that is always on, but it needs a new column, a new hash input and a new fork path for each field, and the module route already carries all three.

Open: whether the owner wants the object in the module's settings or on the agency row.

## 3. Decimal places

- Stored: `compute.places`, a whole number.
- Who may change it: whoever may edit the agency definition in a draft, and the change goes live only when the draft is published with the Issue scan passing. Proposed: a locked permission, so only the agency's owner role can change it (see section 8).
- Proposed default: 2.
- What a change does to publications already made: nothing. A publication is never edited and keeps the value it was published with. The new value applies from the version that is published after the change. A run that already started keeps the settings of the publication it started on (Recommendation; open).

Posted ledger entries never change, because the ledger is append-only and a correction is a reversing entry (decision 6). A currency's places also decide the smallest unit a share is cut down to (decision 13), so changing them after money has been posted changes what a share means. Recommendation: refuse a change of places once the agency has posted any entry, and say so in a plain sentence. This is open.

Example: Lakeside Savings Circle keeps its money in a currency with 2 places. A total of 100.00 shared among 3 members gives 33.33 each. The left-over of 0.01 stays in the account the payout came from. If an agency with 3 places shared 100.000 the same way, each share is 33.333 and the left-over is 0.001.

## 4. Rounding rule

- Stored: `compute.rounding`, one name from the list the library offers.
- Who may change it: as for the places.
- Proposed default: half up. The list offered to a builder wherever a result is a number is proposed as down, up, half up, half down and half even (decision 20).
- What a change does to publications already made: nothing to a publication that exists. A formula that names its own rule keeps it whatever the agency default is. Only a formula that names no rule follows the agency's value, from the version published after the change.

Example: 2.345 rounded to 2 places is 2.35 under half up, 2.34 under half even, 2.34 under down and 2.35 under up. Lakeside's Treasurer sees the same 2.35 on every machine for the same inputs.

Open: whether an agency can change its own default from half up (decision 14 says half up stays the rule when none is chosen; decision 20 does not say the agency may change that default). Recommendation: yes, because decision 20 gives the rounding rule to the agency, and the change is an ordinary published version with its own content hash.

## 5. Digits kept in a division

- Stored: `compute.division_digits`, a whole number.
- Who may change it: as for the places.
- Proposed default: 20.
- What a change does to publications already made: nothing to a publication that exists. The same setting and the same input always give the same figure (decision 21), so a new value gives a different figure only from the version published after the change.

Example: 1 divided by 3 is kept to 20 digits, 0.33333333333333333333, and rounded to 2 places afterwards as the formula says, 0.33.

Open: the range an agency may choose from. Recommendation: from the currency's places up to a ceiling of 30, the longest decimal the language page accepts, so the setting can never be smaller than the amounts it serves or larger than a value the engine would refuse.

## 6. Rows one batch may hold

- Stored: `compute.batch_rows`, a whole number of 1 or more.
- Who may change it: as for the places.
- Proposed default: 10,000, the figure the batch tests use (`2. design/batch-evaluation.md`).
- What a change does to publications already made: nothing to a run that already started; it keeps the limit it started with. A run that starts after the new version is published meets the new limit. A batch over the limit is refused whole before any row is worked out, with the code `too-many-rows`, and no row is calculated and nothing partial comes back.

Example: Lakeside's month-end run covers 12,500 members and the limit is 10,000. A single call of 12,500 rows to the library is refused whole, with no row worked out. Whether the platform or the builder cuts the run into calculation batches of at most 10,000 is open (below).

Open: the ceiling above which no agency may set it, and who cuts a larger run into several calculation batches, the platform or the builder (steps decision 11). Recommendation for the ceiling: the platform owner sets one, proposed 100,000, because the library holds a batch's results in memory. Recommendation for the cutting: the platform, with the confirmer still confirming the whole run once (steps decision 4), so that a builder never has to draw the splitting.

## 7. What the Issue scan says

Proposed sentences, one for each wrong value, so a person can fix it without reading code:

| Wrong value | Sentence |
|---|---|
| places below 0 or above the currency's largest | The decimal places must be a whole number from 0 to 10. |
| a rounding rule not on the list | The rounding rule must be one of: down, up, half up, half down, half even. |
| division digits smaller than the places | The digits kept in a division must be at least the decimal places. |
| batch rows below 1 or above the ceiling | The most rows in one batch must be from 1 to the ceiling set by the platform. |

The upper bound of 10 places comes from the language page's limit on the places of a decimal literal and is proposed. A missing value is not an error: the proposed default is used and the scan says so as a note.

## 8. Who may change the settings

Decided: nothing here is changed in a live agency directly. Every change is a draft change, published with everything else, and the Issue scan gates publishing ([module-settings.md](module-settings.md)).

Proposed: a locked permission for changing compute settings, so that it stays with the agency's owner role and is not handed to a builder by accident. This page does not define the permission code.

Open: whether a permission of its own is wanted, or whether the existing permission to edit the draft is enough.

## 9. Open questions

These are copied from ISS-53 and the pages above. None is decided here.

1. The rounding list: down, up, half up, half down and half even is proposed.
2. Whether an agency can change its default rounding from half up (section 4, recommendation: yes).
3. The 20-digit default for a division that does not end, and the range an agency may choose from (section 5).
4. The ceiling above which no agency may set the batch limit; 10,000 is the proposed default (section 6).
5. Who cuts a larger run into several calculation batches: the platform or the builder (section 6).
6. The direction of the cut for a negative total: the library cuts toward zero for now, which gives -33.33 each and a left-over of -0.01 for -100.00 among 3, and cutting toward negative infinity would give -33.34 and a left-over of +0.02 (`4. qa/exact-money.md`, open question 2).
7. Where the four settings are stored: module settings or the agency row (section 2).
8. Whether a change of places is refused once an entry has been posted (section 3).
9. Whether a run already started keeps the settings of its publication (sections 3 to 6).
10. Whether a permission of its own is wanted (section 8).
