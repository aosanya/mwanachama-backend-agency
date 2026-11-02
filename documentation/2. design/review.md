# Review an agency (HS465-HS468)

The hub's design is [agency-review.md](../../../projects/meta-global-ai-hackathon-community-hubs/documentation/2.%20design/agency-review.md). This page records what the agency backend does about it and where it differs.

## The objects

`review`, `review_area`, `review_finding` and `review_comment` are declared in `agency.blueprint.json` (HS465). Reviews belong to a draft and are never copied into a publication.

A finding's `then` list is stored in the column `then_steps` (Go field `ThenSteps`), because `then` is a reserved word and the migration writes column names unquoted. Everywhere else it is `then`: the tool and the HTTP body take and give an ordered array of `{tool, arguments}`, where `arguments` is a JSON object. The finding's own `arguments` is stored as JSON in the column of that name.

## The seven operations

All seven are declared in `agency.operations.json` and call a manager method; no hand-written tool exists. Every tool also takes `instance_id` from the connection, like the others.

| Tool | HTTP | Arguments | Returns |
|---|---|---|---|
| `agency_start_review` | `POST /reviews` | `draft_id`, optional `started_by`, optional `area` | `{review, areas}` |
| `agency_get_review` | `GET /review` | `review_id`, or `draft_id` for the latest | `{review, areas}`; with `draft_id` each part is the newest run that covered it; `{review: null}` when the draft has none |
| `agency_list_reviews` | `GET /reviews` | `draft_id` | `{items: [review]}`, newest first, no parts |
| `agency_set_finding_status` | `PATCH /review-findings/{finding_id}` | `finding_id`, `status` | the finding with its `comments` |
| `agency_comment_finding` | `POST /review-findings/{finding_id}/comments` | `finding_id`, `body`, optional `author` | the comment |
| `agency_review_input` | `GET /reviews/{review_id}/input` | `review_id`, `area` | `{previous: [finding]}` from the newest earlier review that covered the part |
| `agency_record_review_area` | `PUT /reviews/{review_id}/areas/{area}` | `review_id`, `area`, `status`, `summary`, `failure_reason`, `tokens`, `findings` | `{area, review_status}` |

A finding is `ReviewFinding` (with its `ordinal`) plus `comments`. An area is `ReviewArea` plus `findings`.

Differences from the hub's table: a listing is `{items}` like every listing in this repo, not `{reviews}`; `started_by` and `author` are optional arguments because the MCP connection carries no caller here; a finding also carries `review_id`.

## Starting a run

`StartReview` needs an open draft, refuses with the sentence `A review of this draft is already running.` while any review of the draft is `running`, sets `previous_review_id` to the latest earlier review, and writes one `reading` area per part.

Two starts at once for one draft cannot both pass the check: a review carries `running`, true from its start until its last part is recorded, and the unique index `running_unique` on `draft_id` where `running` is true refuses a second one. The index cannot test the `status` column, because an index condition here can only ask whether a column is empty, true or false, so `running` mirrors `status` being `running`. `StartReview` reads a unique violation as `ErrReviewRunning`, the same sentence as the check. Reviews written before the column existed hold no value in it and are not covered by the index; the check on `status` still refuses a start beside them.

`StartReview(ctx, draftID, startedBy, area string)` takes an optional `area`. Empty reads every part the hub reads. A part name (`profile`, `goals`, `objectives`, `roles_agents`, `workflows`, `risks`, `tests`) writes only that one `reading` area; an unknown name, or one the hub does not read for the draft, is refused with `ErrInvalidReview`. The other parts are left as they were: they keep the findings, notes and dismissals of the last run that covered them.

### Which earlier part a part follows

Because a review may hold one part, "the previous review" no longer says what a part follows. `coveringArea` in `review_cover_impl.go` walks `previous_review_id` from a review back to the newest review that holds the part. `ReviewInput` reads its `previous` findings from that part, and a carried finding must belong to that same part, not merely to the previous review.

### Reading a draft

`GetReview` by `draft_id` answers the latest review as `review` and, for each part the hub reads, the part from the newest review that held it, ordered as below. A part being read by a running area-only run shows `reading` from that run while the others show their last results. When every run covered every part this is the latest review exactly as before. `GetReview` by `review_id` still answers that review alone.

Which parts are read is decided in one place, `AllReviewAreas` in `review_areas.go`, which returns all six. The farmers' agency should leave out `objectives`; the repo has no agency kind yet, so the hub passes its own function with `WithReviewAreas` when it has one.

## Recording a part

`RecordReviewArea` runs in one transaction that first locks the review row, so six reviewers finishing together cannot both see the other as still reading. It refuses a part already recorded, a status other than `done` or `failed`, negative tokens, an untitled finding, a `fate` outside the four, and a `then` step with no tool.

Carry-over, for a finding with `carried_from_id`:

| Earlier status | Fate now | Status written |
|---|---|---|
| `open` | any | `open` |
| `applied` | `still_open` | `open` |
| `applied` | `fixed`, `withdrawn` | `applied` |
| `dismissed` | any, `reopen` true | `open` |
| `dismissed` | any, `reopen` false | `dismissed` |

The earlier finding must belong to the previous review. A carried finding cannot have fate `new`. A finding without `carried_from_id` is written `open`, fate `new` when none is given.

The area's tokens are added to the review. When no area is left `reading` the review becomes `done` and `finished_at` is written, even if every area failed; `failed` on a review is never written by these tools.

## What the Workflows part checks

The Workflows reviewer reads every workflow of the draft: steps in a sensible order, a person at each wait, time limits, and a name that starts with a verb that says what the person does (`Create group order`, `Buy now`, `Approve a poster`; not `Group order`, `Group sale`, `Pooled input order`). A name that does not is raised as a finding whose Apply is `agency_update_workflow` changing only `name`, to the workflow's own words with a verb in front. Standard: workflow names start with a verb.

## Order

Areas are returned in the order profile, goals, objectives, roles_agents, workflows, risks, tests. Findings are returned in the order the reviewer wrote them: each stores its `ordinal`, counting from 1 within its part, set when the part is recorded after a repeated title has been folded into its first place. Findings recorded before the column existed have no ordinal and are returned by title, then id. Comments are returned oldest first.
