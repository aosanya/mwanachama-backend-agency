# Home entries

An agency declares the rows of the app's Home, as a tree, including a page for each row of a record. This page is what the agency stores and what its three tools answer. How the app draws them is the hub's design page.

## The declaration

An agency file may carry a top-level `home`: a list of nodes in the order a person should see them. A node is a section (it has `entries`, further nodes), an entry (it has a `target`) or a row page (a section with `page_of`). Sections nest, mixed with entries, up to 5 levels: Home's own rows are level 1.

| Field | Meaning |
|---|---|
| `key` | The handle: letters, digits, hyphen and underscore, at most 60. Unique in a draft. |
| `title` | What the row says, at most 80 characters. |
| `titles` | The title in other languages, a map of language code to text, each at most 80 characters and never blank. |
| `icon` | An icon name the app draws. Not checked here: the set belongs to the app. |
| `permission` | A code of the permission catalogue. |
| `roles` | Role codes of the agency. Each must be a role of the same draft or file. |
| `page_of` | On a section: a record code. The section is the page of one row of that record and is not listed. One section per record code in a draft. |
| `target` | `{ "type", "ref", "step", "scope", "where" }` on an entry. |

Target types: `record` (`ref` a record code), `workflow` (`ref` a workflow code), `backlog` (`ref` a name or id), `menu` (`ref` as `section/item`), `files`, `start` and `ongoing` (no `ref`). `step` is a work item code and only on a workflow. `scope` is `row.order` and only on a workflow inside a row page. `where` is at most 5 conditions on a record, each `{ "question", "operator", "value" }` with an operator of `is`, `is_not`, `contains`, `more_than`, `less_than`, `on_or_before` or `on_or_after`, and a value that is a text or `{ "row": "<question prompt>" }`, the answer of the tapped row, only inside a row page. A section holds at most 40 nodes, a draft at most 300, a `ref` at most 200 characters.

## What is stored

Each node is one row of the draft-scoped object `home_entry`: `parent_id` (empty on Home itself), `key`, `title`, `titles`, `icon`, `permission`, `roles`, `page_of`, `target_type`, `target`, `step`, `scope`, `conditions` (the `where`), `ordinality` (the place among its siblings, from 0, in the declared order). A section has an empty `target_type`. A key is unique among a draft's rows, and so is a non-empty `page_of`.

## The operations

| Action | Tool | Route | Arguments |
|---|---|---|---|
| `agency.home_entry.create` | `agency_create_home_entry` | `POST /home-entries` | one row: `draft_id`, `parent_id?`, `key`, `title`, the optional fields above (`conditions` for `where`) |
| `agency.home_entry.list` | `agency_list_home_entries` | `GET /home-entries` | `draft_id` |
| `agency.home.set` | `agency_set_home` | `PUT /home` | `draft_id`, `entries`: the whole nested Home |

A create puts the row last in its section. `agency_set_home` replaces every row of the draft in one transaction with the nested list sent, under the same refusals as import; a refusal keeps the old Home, and an empty list clears it. There is no per-row change or remove. Refusals: `ErrInvalidHomeEntry` (400) for a broken row, a repeated key, a repeated row page, a role the draft does not hold, a parent that is an entry, a sixth level, a full section or draft, or a tapped-row value outside a row page; `ErrHomeEntryNotFound` (404) for a parent that does not exist; `ErrDraftNotOpen` (409) once the draft is archived.

## Where it travels

- **Import** of an agency document writes the `home` nested into rows. A document that breaks a limit or any refusal above is refused whole with `ErrInvalidHomeEntry`, and no draft is made.
- **Fork from a draft** copies every row with a new id, each into its copied parent.
- **Publish** writes `home` nested into the publication's document. The content hash counts every row by key, parent key, order and fields and not by row id, so setting the same Home again is no change and adding, retitling or moving a row is.
- **Fork from a publication** reads the nested `home` back into rows.
- A document without `home` imports, forks and publishes as before. A reader that does not know `home` ignores it.

`models.NestHomeEntries` turns a draft's rows into the nested form, ordered by `ordinality`.

## Not here

Whether a `record`, `workflow` or `step` target exists, and whether a `where` question is one of the record's, is not checked when a row is written: the reader of the published version drops a target that is gone. The Issue scan does not report a dangling target. A backlog is the instance's own data and cannot be checked here.
