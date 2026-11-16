# Updates keep the fields a request does not carry

Hub issue AG116. The console's update body for a workflow leaves out `ordinality`, `event_flows`, `opens_in` and `cancels_order`, and its body for a step leaves out `tool`. The update operations write the whole decoded struct, so saving blanked them. Two probes filed on 2026-11-02 (79d27ca) showed it and stayed red until now.

## How it works

`models.Role` already remembered which JSON keys a request carried (d44235a, hub b2d0ca7d). That memory is now one unexported type, `carriedKeys` in `models/carried.go`, embedded in `Role`, `Workflow` and `WorkItem`. Each of the three has an `UnmarshalJSON` that decodes as before and then records the keys present in the body.

- `Carries(field)` is true for a key the body held, and true for every field when the value was not decoded from JSON, so a caller that builds the struct in code replaces the whole row as it always did.
- `FromRequest()` is true only for a decoded body.
- A key sent empty (`"event_flows": ""`, `"ordinality": 0`, `"tool": ""`) is carried and clears the field.

`UpdateWorkflow` keeps `ordinality`, `event_flows`, `opens_in` and `cancels_order` from the stored row when they are not carried, before the shape is validated. `UpdateWorkItem` keeps `tool` the same way. The other fields of both are unchanged: a body still replaces them.

## Tests

- `TestUpdateWorkflow_TheConsolesBodyKeepsItsOrdinalityAndEventFlows` and `TestUpdateWorkItem_TheConsolesBodyKeepsItsTool` were the red tests and are green.
- `update_clears_sent_fields_test.go` holds the guards that a sent empty value clears, that a body built in code still replaces the row, and that `opens_in` and `cancels_order` are kept.

## The test that stays red

`routes` `TestAG74_OpenHole_AttachEditDetachRewritesAnAgencyWideValue` is not a stale expectation. It asserts the invariant AG74 filed (attach, edit, detach must not rewrite an agency-wide value) and the code does not hold it. The board (AG74, AG107), AGD-011 and `4. qa/README.md` record that it stays red until AG107 deletes `rule` and `value`. It was not changed.
