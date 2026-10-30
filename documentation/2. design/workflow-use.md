# Default workflows and a workflow's use (HS385)

`agency_list_default_workflows` (`GET /workflows/defaults`) lists the
workflows every agency has built in. Cancel is the first: code `cancel`, name
"Cancel", description "At once", `default: true`. It holds until an agency adds
a cancel workflow of its own; it is not a row, so it is never edited or
removed.

Every workflow the agency reports, from `agency_list_workflows`,
`agency_get_workflow` and the default list, carries:

- `in_use`: an order is under way on it.
- `orders_under_way`: how many.
- `order_versions`: the distinct versions those orders are on, oldest first.
  Orders keep the version they started on, so this can hold two versions while
  the workflow is already published at a newer one.
- `version`: the version the production channel is published at, 0 before the
  first publish. A default carries none.

These are derived, not stored. The orders belong to the task manager, which
the agency cannot reach, so the manager reads them through `WithRunLister`,
which the host sets (the same way `WithGroupLister` supplies groups). A run is
matched to a workflow by the workflow's code, the `workflow_code` the order
was started with. A host that sets none reports nothing in use, and a source
that fails fails the call.
