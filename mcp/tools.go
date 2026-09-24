package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

const instanceArg = "instance_id"

var operations = sync.OnceValues(func() (*dispatch.Spec, error) {
	return dispatch.Parse(mwanachamaagency.Operations())
})

var carriers = map[string]string{
	"agency":             "Agency",
	"draft":              "Draft",
	"goal":               "Goal",
	"objective":          "Objective",
	"workflow":           "Workflow",
	"work_item":          "WorkItem",
	"event_flow_step":    "EventFlowStep",
	"role":               "Role",
	"role_assignment":    "RoleAssignment",
	"module":             "Module",
	"objective_module":   "ObjectiveModule",
	"value":              "Value",
	"value_attachment":   "ValueAttachment",
	"rule":               "Rule",
	"rule_attachment":    "RuleAttachment",
	"issue":              "Issue",
	"agency_publication": "AgencyPublication",
}

var storeOwned = map[string]bool{
	"Module.enabled":  true,
	"Draft.status":    true,
	"Issue.severity":  true,
	"Issue.message":   true,
	"Issue.entity_id": true,
}

func fields() (map[string]dispatch.FieldDoc, error) {
	b, err := mwanachamaagency.Blueprint()
	if err != nil {
		return nil, err
	}
	out := map[string]dispatch.FieldDoc{}
	for role, carrier := range carriers {
		o, ok := b.Object(role)
		if !ok {
			return nil, fmt.Errorf("agency tools: the blueprint declares no %q", role)
		}
		for _, f := range o.Fields {
			key := carrier + "." + f.Name
			out[key] = dispatch.FieldDoc{
				Description: f.Description,
				Required:    f.Required,
				ReadOnly:    f.Primary || f.Type == spec.TypeTimestamp || f.Immutable || f.Name == "deleted" || storeOwned[key],
				Values:      f.Values,
			}
		}
	}
	return out, nil
}

func toolsFor(m mwanachamaagency.AgencyManager) ([]dispatch.Tool, error) {
	s, err := operations()
	if err != nil {
		return nil, err
	}
	f, err := fields()
	if err != nil {
		return nil, err
	}
	return dispatch.Tools(s, dispatch.Deps{Manager: m, Errors: sentinels, Fields: f})
}

func registerDeclaredTools(server *mcp.Server, resolve ManagerResolver) error {
	shapes, err := toolsFor(mwanachamaagency.ManagerShape())
	if err != nil {
		return err
	}
	for _, shape := range shapes {
		schema, err := withInstance(shape.InputSchema)
		if err != nil {
			return fmt.Errorf("agency tools: %s: %w", shape.Name, err)
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        shape.Name,
			Title:       shape.Title,
			Description: shape.Description,
			InputSchema: schema,
		}, declaredHandler(shape.Name, resolve))
	}
	return nil
}

func withInstance(schema json.RawMessage) (json.RawMessage, error) {
	var doc map[string]any
	if err := json.Unmarshal(schema, &doc); err != nil {
		return nil, err
	}
	properties, ok := doc["properties"].(map[string]any)
	if !ok {
		properties = map[string]any{}
		doc["properties"] = properties
	}
	properties[instanceArg] = map[string]any{
		"type":        "string",
		"description": "which Agency this operates on, as resolved by the mounting host",
	}

	required, _ := doc["required"].([]any)
	doc["required"] = append(required, instanceArg)

	return json.Marshal(doc)
}

func declaredHandler(name string, resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, map[string]any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, any, error) {
		instance, _ := in[instanceArg].(string)
		delete(in, instanceArg)

		m, err := resolve(ctx, instance)
		if err != nil {
			return nil, nil, err
		}
		tools, err := toolsFor(m)
		if err != nil {
			return nil, nil, err
		}
		tool, ok := byName(tools, name)
		if !ok {
			return nil, nil, fmt.Errorf("agency tools: %s is no longer declared", name)
		}

		args, err := json.Marshal(in)
		if err != nil {
			return nil, nil, err
		}
		out, err := tool.Invoke(ctx, args)
		if err != nil {
			return nil, nil, err
		}
		if out == nil {
			return summary("Done."), nil, nil
		}
		return summaryWithContext(headline(out), describe(out)), structured(out), nil
	}
}

// structured keeps a listing an object. MCP defines structuredContent as a
// JSON object, and a bare array fails a spec-strict client while passing a
// lenient one — the live failure AG13 found.
func structured(out any) any {
	if reflect.ValueOf(out).Kind() == reflect.Slice {
		return map[string]any{"items": out}
	}
	return out
}

func byName(tools []dispatch.Tool, name string) (dispatch.Tool, bool) {
	for _, t := range tools {
		if t.Name == name {
			return t, true
		}
	}
	return dispatch.Tool{}, false
}

func headline(out any) string {
	v := reflect.ValueOf(out)
	if v.Kind() == reflect.Slice {
		return fmt.Sprintf("Found %d.", v.Len())
	}
	return "Done."
}

// describe is what a model reads to act on an answer. The go-sdk fills
// StructuredContent from the returned value either way, but a client that
// joins the text content and never looks at the structure would otherwise
// see a count with no id in it — see AG23, which this generalizes.
func describe(out any) string {
	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Slice {
		return line(v)
	}
	if v.Len() == 0 {
		return "There are none yet."
	}
	lines := make([]string, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		lines = append(lines, "- "+line(v.Index(i)))
	}
	return strings.Join(lines, "\n")
}

func line(v reflect.Value) string {
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Sprint(v.Interface())
	}

	var head []string
	for _, name := range []string{"Code", "Tag", "Version"} {
		if s := text(v, name); s != "" {
			head = append(head, s)
			break
		}
	}
	for _, name := range []string{"Title", "Name", "Text", "Message", "Label", "Description"} {
		if s := text(v, name); s != "" {
			head = append(head, fmt.Sprintf("%q", s))
			break
		}
	}

	var tail []string
	for _, name := range []string{"ID", "DraftID", "Channel", "Severity", "TargetCode"} {
		if s := text(v, name); s != "" {
			tail = append(tail, strings.ToLower(columnish(name))+": "+s)
		}
	}

	if len(head) == 0 && len(tail) == 0 {
		return "(no identifying fields)"
	}
	if len(tail) == 0 {
		return strings.Join(head, " ")
	}
	return strings.TrimSpace(strings.Join(head, " ") + " (" + strings.Join(tail, ", ") + ")")
}

func text(v reflect.Value, name string) string {
	f := v.FieldByName(name)
	if !f.IsValid() {
		return ""
	}
	switch f.Kind() {
	case reflect.String:
		return f.String()
	case reflect.Int, reflect.Int64:
		return fmt.Sprint(f.Int())
	default:
		return ""
	}
}

func columnish(field string) string {
	if field == "ID" {
		return "id"
	}
	return strings.ReplaceAll(field, "ID", " id")
}
