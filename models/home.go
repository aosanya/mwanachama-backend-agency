package models

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
)

type HomeTargetType string

const (
	HomeTargetRecord   HomeTargetType = "record"
	HomeTargetWorkflow HomeTargetType = "workflow"
	HomeTargetBacklog  HomeTargetType = "backlog"
	HomeTargetMenu     HomeTargetType = "menu"
	HomeTargetFiles    HomeTargetType = "files"
	HomeTargetStart    HomeTargetType = "start"
	HomeTargetOngoing  HomeTargetType = "ongoing"
)

const HomeScopeRowOrder = "row.order"

const (
	HomeMaxDepth        = 5
	HomeMaxPerSection   = 40
	HomeMaxEntries      = 300
	HomeMaxKeyLength    = 60
	HomeMaxTitleLength  = 80
	HomeMaxTargetLength = 200
	HomeMaxConditions   = 5
	HomeMaxRoles        = 20
)

var homeKeyExpression = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var homeOperators = []string{"is", "is_not", "contains", "more_than", "less_than", "on_or_before", "on_or_after"}

func (t HomeTargetType) Valid() bool {
	switch t {
	case HomeTargetRecord, HomeTargetWorkflow, HomeTargetBacklog, HomeTargetMenu, HomeTargetFiles, HomeTargetStart, HomeTargetOngoing:
		return true
	}
	return false
}

func (t HomeTargetType) needsRef() bool {
	return t != HomeTargetFiles && t != HomeTargetStart && t != HomeTargetOngoing
}

type HomeValue struct {
	Literal string
	Row     string
}

func (v HomeValue) MarshalJSON() ([]byte, error) {
	if v.Row != "" {
		return json.Marshal(map[string]string{"row": v.Row})
	}
	return json.Marshal(v.Literal)
}

func (v *HomeValue) UnmarshalJSON(raw []byte) error {
	var literal string
	if err := json.Unmarshal(raw, &literal); err == nil {
		*v = HomeValue{Literal: literal}
		return nil
	}
	var placeholder struct {
		Row string `json:"row"`
	}
	if err := decodeHomeFields(raw, &placeholder, []string{"row"}, nil); err != nil || strings.TrimSpace(placeholder.Row) == "" {
		return fmt.Errorf("a condition value is a text or {\"row\": \"<question prompt>\"}")
	}
	*v = HomeValue{Row: placeholder.Row}
	return nil
}

func decodeHomeFields(raw []byte, into any, known, notBlank []string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for name := range fields {
		if !slices.Contains(known, name) {
			return fmt.Errorf("%q is not a field of this part of a home", name)
		}
	}
	for _, name := range notBlank {
		var text string
		if value, present := fields[name]; present && json.Unmarshal(value, &text) == nil && strings.TrimSpace(text) == "" {
			return fmt.Errorf("%q is present but blank; leave it out", name)
		}
	}
	return json.Unmarshal(raw, into)
}

func (c *HomeCondition) UnmarshalJSON(raw []byte) error {
	type plain HomeCondition
	var out plain
	if err := decodeHomeFields(raw, &out, []string{"question", "operator", "value"}, nil); err != nil {
		return err
	}
	*c = HomeCondition(out)
	return nil
}

func (t *HomeTarget) UnmarshalJSON(raw []byte) error {
	type plain HomeTarget
	var out plain
	if err := decodeHomeFields(raw, &out, []string{"type", "ref", "step", "scope", "where"}, []string{"ref", "step", "scope"}); err != nil {
		return err
	}
	*t = HomeTarget(out)
	return nil
}

func (n *DocumentHomeNode) UnmarshalJSON(raw []byte) error {
	type plain DocumentHomeNode
	var out plain
	if err := decodeHomeFields(raw, &out, []string{"key", "title", "titles", "icon", "permission", "roles", "page_of", "target", "entries"}, []string{"page_of", "permission"}); err != nil {
		return err
	}
	*n = DocumentHomeNode(out)
	return nil
}

type HomeCondition struct {
	Question string    `json:"question"`
	Operator string    `json:"operator"`
	Value    HomeValue `json:"value"`
}

type HomeEntry struct {
	ID          string            `json:"id"`
	DraftID     string            `json:"draft_id,omitempty"`
	ParentID    string            `json:"parent_id,omitempty"`
	Key         string            `json:"key"`
	Title       string            `json:"title"`
	Titles      map[string]string `json:"titles,omitempty"`
	Icon        string            `json:"icon,omitempty"`
	Permission  string            `json:"permission,omitempty"`
	Roles       []string          `json:"roles,omitempty"`
	PageOf      string            `json:"page_of,omitempty"`
	TargetType  HomeTargetType    `json:"target_type,omitempty"`
	Target      string            `json:"target,omitempty"`
	Step        string            `json:"step,omitempty"`
	Scope       string            `json:"scope,omitempty"`
	Conditions  []HomeCondition   `json:"conditions,omitempty"`
	Ordinality  int               `json:"ordinality"`
	CreatedAt   string            `json:"created_at"`
	LastUpdated string            `json:"last_updated"`
	Deleted     bool              `json:"deleted"`
}

func (e HomeEntry) IsSection() bool { return e.TargetType == "" }

type HomeTarget struct {
	Type  HomeTargetType  `json:"type"`
	Ref   string          `json:"ref,omitempty"`
	Step  string          `json:"step,omitempty"`
	Scope string          `json:"scope,omitempty"`
	Where []HomeCondition `json:"where,omitempty"`
}

type DocumentHomeNode struct {
	Key        string             `json:"key"`
	Title      string             `json:"title"`
	Titles     map[string]string  `json:"titles,omitempty"`
	Icon       string             `json:"icon,omitempty"`
	Permission string             `json:"permission,omitempty"`
	Roles      []string           `json:"roles,omitempty"`
	PageOf     string             `json:"page_of,omitempty"`
	Target     *HomeTarget        `json:"target,omitempty"`
	Entries    []DocumentHomeNode `json:"entries,omitempty"`
}

func (n DocumentHomeNode) IsSection() bool { return n.Target == nil }

func ValidateHomeEntry(e HomeEntry) error {
	if e.Key == "" || len(e.Key) > HomeMaxKeyLength || !homeKeyExpression.MatchString(e.Key) {
		return fmt.Errorf("a home entry has a key of letters, digits, hyphens and underscores, at most %d long (got %q)", HomeMaxKeyLength, e.Key)
	}
	if strings.TrimSpace(e.Title) == "" {
		return fmt.Errorf("the home entry %q has a title", e.Key)
	}
	if len([]rune(e.Title)) > HomeMaxTitleLength {
		return fmt.Errorf("the title of the home entry %q is longer than %d characters", e.Key, HomeMaxTitleLength)
	}
	for language, title := range e.Titles {
		if strings.TrimSpace(language) == "" || strings.TrimSpace(title) == "" || len([]rune(title)) > HomeMaxTitleLength {
			return fmt.Errorf("the home entry %q has a translated title that is empty or longer than %d characters", e.Key, HomeMaxTitleLength)
		}
	}
	if e.Permission != "" {
		if _, found := LookupPermission(e.Permission); !found {
			return fmt.Errorf("the home entry %q names the permission %q, which is not in the catalogue", e.Key, e.Permission)
		}
	}
	if len(e.Roles) > HomeMaxRoles {
		return fmt.Errorf("the home entry %q names more than %d roles", e.Key, HomeMaxRoles)
	}
	for _, role := range e.Roles {
		if strings.TrimSpace(role) == "" {
			return fmt.Errorf("the home entry %q names a blank role", e.Key)
		}
	}
	if e.IsSection() {
		return validateHomeSection(e)
	}
	return validateHomeEntryTarget(e)
}

func validateHomeSection(e HomeEntry) error {
	if e.Target != "" || e.Step != "" || e.Scope != "" || len(e.Conditions) > 0 {
		return fmt.Errorf("the home entry %q is a section and cannot have a target", e.Key)
	}
	if len(e.PageOf) > HomeMaxTargetLength {
		return fmt.Errorf("the page_of of the home entry %q is longer than %d characters", e.Key, HomeMaxTargetLength)
	}
	return nil
}

func validateHomeEntryTarget(e HomeEntry) error {
	if e.PageOf != "" {
		return fmt.Errorf("the home entry %q opens something, so it cannot be the page of a row", e.Key)
	}
	if !e.TargetType.Valid() {
		return fmt.Errorf("the home entry %q has an unknown target type %q", e.Key, e.TargetType)
	}
	if e.TargetType.needsRef() && strings.TrimSpace(e.Target) == "" {
		return fmt.Errorf("the home entry %q opens a %s and names which one", e.Key, e.TargetType)
	}
	if !e.TargetType.needsRef() && e.Target != "" {
		return fmt.Errorf("the home entry %q opens %s and names nothing more", e.Key, e.TargetType)
	}
	if len(e.Target) > HomeMaxTargetLength || len(e.Step) > HomeMaxTargetLength {
		return fmt.Errorf("the target of the home entry %q is longer than %d characters", e.Key, HomeMaxTargetLength)
	}
	if e.TargetType == HomeTargetMenu && !strings.Contains(e.Target, "/") {
		return fmt.Errorf("the home entry %q opens a menu item and names it as section/item", e.Key)
	}
	if e.Step != "" && e.TargetType != HomeTargetWorkflow {
		return fmt.Errorf("the home entry %q names a step but does not open a workflow", e.Key)
	}
	if e.Scope != "" && (e.TargetType != HomeTargetWorkflow || e.Scope != HomeScopeRowOrder) {
		return fmt.Errorf("the home entry %q has a scope that is not %s on a workflow", e.Key, HomeScopeRowOrder)
	}
	if len(e.Conditions) > 0 && e.TargetType != HomeTargetRecord {
		return fmt.Errorf("the home entry %q narrows a list but does not open a record", e.Key)
	}
	if len(e.Conditions) > HomeMaxConditions {
		return fmt.Errorf("the home entry %q has more than %d conditions", e.Key, HomeMaxConditions)
	}
	for _, c := range e.Conditions {
		if err := validateHomeCondition(e.Key, c); err != nil {
			return err
		}
	}
	return nil
}

func validateHomeCondition(key string, c HomeCondition) error {
	if strings.TrimSpace(c.Question) == "" {
		return fmt.Errorf("a condition of the home entry %q names no question", key)
	}
	known := false
	for _, operator := range homeOperators {
		known = known || operator == c.Operator
	}
	if !known {
		return fmt.Errorf("a condition of the home entry %q has the operator %q, which is not one of %s", key, c.Operator, strings.Join(homeOperators, ", "))
	}
	if c.Value.Row == "" && strings.TrimSpace(c.Value.Literal) == "" {
		return fmt.Errorf("a condition of the home entry %q has no value", key)
	}
	if len(c.Value.Literal) > HomeMaxTargetLength {
		return fmt.Errorf("a condition of the home entry %q has a value longer than %d characters", key, HomeMaxTargetLength)
	}
	return nil
}

func UsesHomeRow(e HomeEntry) bool {
	if e.Scope != "" {
		return true
	}
	for _, c := range e.Conditions {
		if c.Value.Row != "" {
			return true
		}
	}
	return false
}

func HomeEntryOfNode(n DocumentHomeNode) HomeEntry {
	e := HomeEntry{Key: n.Key, Title: n.Title, Titles: n.Titles, Icon: n.Icon, Permission: n.Permission, Roles: n.Roles, PageOf: n.PageOf}
	if n.Target != nil {
		e.TargetType, e.Target, e.Step, e.Scope, e.Conditions = n.Target.Type, n.Target.Ref, n.Target.Step, n.Target.Scope, n.Target.Where
	}
	return e
}

func ValidateHomeDocument(nodes []DocumentHomeNode) error {
	seen := map[string]bool{}
	pages := map[string]bool{}
	total := 0
	var walk func(level int, inPage bool, nodes []DocumentHomeNode) error
	walk = func(level int, inPage bool, nodes []DocumentHomeNode) error {
		if len(nodes) == 0 {
			return nil
		}
		if level > HomeMaxDepth {
			return fmt.Errorf("the home goes deeper than %d levels", HomeMaxDepth)
		}
		if len(nodes) > HomeMaxPerSection {
			return fmt.Errorf("a home section holds more than %d entries", HomeMaxPerSection)
		}
		for _, n := range nodes {
			total++
			if total > HomeMaxEntries {
				return fmt.Errorf("the home holds more than %d entries", HomeMaxEntries)
			}
			if n.Target != nil && n.Entries != nil {
				return fmt.Errorf("the home entry %q has both a target and entries", n.Key)
			}
			row := HomeEntryOfNode(n)
			if err := ValidateHomeEntry(row); err != nil {
				return err
			}
			if UsesHomeRow(row) && !inPage {
				return fmt.Errorf("the home entry %q uses the tapped row outside a row page", n.Key)
			}
			if n.PageOf != "" {
				if pages[n.PageOf] {
					return fmt.Errorf("the record %q has more than one row page", n.PageOf)
				}
				pages[n.PageOf] = true
			}
			if seen[n.Key] {
				return fmt.Errorf("the home key %q is used twice", n.Key)
			}
			seen[n.Key] = true
			if err := walk(level+1, inPage || n.PageOf != "", n.Entries); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(1, false, nodes)
}

func HomeRoleCodes(nodes []DocumentHomeNode) []string {
	var out []string
	for _, n := range nodes {
		out = append(out, n.Roles...)
		out = append(out, HomeRoleCodes(n.Entries)...)
	}
	return out
}

func NestHomeEntries(rows []HomeEntry) []DocumentHomeNode {
	byParent := map[string][]HomeEntry{}
	for _, row := range rows {
		byParent[row.ParentID] = append(byParent[row.ParentID], row)
	}
	var build func(parentID string) []DocumentHomeNode
	build = func(parentID string) []DocumentHomeNode {
		children := byParent[parentID]
		sort.SliceStable(children, func(i, j int) bool { return children[i].Ordinality < children[j].Ordinality })
		var out []DocumentHomeNode
		for _, row := range children {
			node := DocumentHomeNode{Key: row.Key, Title: row.Title, Titles: row.Titles, Icon: row.Icon, Permission: row.Permission, Roles: row.Roles, PageOf: row.PageOf}
			if row.IsSection() {
				node.Entries = build(row.ID)
			} else {
				node.Target = &HomeTarget{Type: row.TargetType, Ref: row.Target, Step: row.Step, Scope: row.Scope, Where: row.Conditions}
			}
			out = append(out, node)
		}
		return out
	}
	return build("")
}
