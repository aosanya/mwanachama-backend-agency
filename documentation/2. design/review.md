# Review an agency (HS465-HS468)

The hub's design is [agency-review.md](../../../projects/meta-global-ai-hackathon-community-hubs/documentation/2.%20design/agency-review.md). This page records what the agency backend does about it and where it differs.

## The objects

`review`, `review_area`, `review_finding` and `review_comment` are declared in `agency.blueprint.json` (HS465). Reviews belong to a draft and are never copied into a publication.

A finding's `then` list is stored in the column `then_steps` (Go field `ThenSteps`), because `then` is a reserved word and the migration writes column names unquoted. Everywhere else it is `then`: the tool and the HTTP body take and give an ordered array of `{tool, arguments}`, where `arguments` is a JSON object. The finding's own `arguments` is stored as JSON in the column of that name.

## The seven operations

All seven are declared in `agency.operations.json` and call a manager method; no hand-written tool exists. Every tool also takes `instance_id` from the connection, like the others.

| Tool | HTTP | Arguments | Returns |
|---|---|---|---|
| `agency_start_review` | `POST /reviews` | `draft_id`, optional `started_by` | `{review, areas}` |
| `agency_get_review` | `GET /review` | `review_id`, or `draft_id` for the latest | `{review, areas}`; `{review: null}` when the draft has none |
| `agency_list_reviews` | `GET /reviews` | `draft_id` | `{items: [review]}`, newest first, no parts |
| `agency_set_finding_status` | `PATCH /review-findings/{finding_id}` | `finding_id`, `status` | the finding with its `comments` |
| `agency_comment_finding` | `POST /review-findings/{finding_id}/comments` | `finding_id`, `body`, optional `author` | the comment |
| `agency_review_input` | `GET /reviews/{review_id}/input` | `review_id`, `area` | `{previous: [finding]}` |
| `agency_record_review_area` | `PUT /reviews/{review_id}/areas/{area}` | `review_id`, `area`, `status`, `summary`, `failure_reason`, `tokens`, `findings` | `{area, review_status}` |

A finding is `ReviewFinding` plus `comments`. An area is `ReviewArea` plus `findings`.

Differences from the hub's table: a listing is `{items}` like every listing in this repo, not `{reviews}`; `started_by` and `author` are optional arguments because the MCP connection carries no caller here; a finding also carries `review_id`.

## Starting a run

`StartReview` needs an open draft, refuses with the sentence `A review of this draft is already running.` while any review of the draft is `running`, sets `previous_review_id` to the latest earlier review, and writes one `reading` area per part.

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

## Order

Areas are returned in the order profile, goals, objectives, roles_agents, workflows, risks, tests. Findings are returned by title, then id, because a finding stores no position. Comments are returned oldest first.
