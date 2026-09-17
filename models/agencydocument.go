package models

// AgencyDocument is a whole Agency serialized into one value — what an
// [AgencyPublication] carries, and what production reads to operate
// (AGD-005). Nothing running resolves the working rows; it resolves the
// document on the tag published to its channel.
//
// The shape deliberately mirrors mwanachama-wakala-api's
// `POST /instances/import` document field-for-field, so a publication can
// be fed straight back to import and the 52 mined agency templates are
// already written in it. Two additions that import does not itself apply:
// EventFlowSteps (kept when WorkPlan/ContextSource were retired — AGD-005
// #7) and the Agency's own non-identity settings. Import decodes leniently,
// so those extra sections pass through it harmlessly.
//
// Every reference between sections is by the id the row carried when it
// was published, with Code alongside it. An importer may resolve either;
// Code is the one that survives a promotion unchanged (AG16).
type AgencyDocument struct {
	Agency           DocumentAgency            `json:"agency"`
	Levels           []DocumentLevel           `json:"levels"`
	GroupTypes       []DocumentGroupType       `json:"group_types"`
	Roles            []DocumentRole            `json:"roles"`
	Goals            []DocumentGoal            `json:"goals"`
	Objectives       []DocumentObjective       `json:"objectives"`
	Workflows        []DocumentWorkflow        `json:"workflows"`
	WorkItems        []DocumentWorkItem        `json:"work_items"`
	RoleAssignments  []DocumentRoleAssignment  `json:"role_assignments"`
	Values           []DocumentStatement       `json:"values"`
	Rules            []DocumentStatement       `json:"rules"`
	ValueAttachments []DocumentAttachment      `json:"value_attachments"`
	RuleAttachments  []DocumentAttachment      `json:"rule_attachments"`
	Modules          []DocumentModule          `json:"modules"`
	ObjectiveModules []DocumentObjectiveModule `json:"objective_modules"`
	EventFlowSteps   []EventFlowStep           `json:"event_flow_steps"`
}

type DocumentAgency struct {
	Name                         string `json:"name"`
	Description                  string `json:"description"`
	Enabled                      bool   `json:"enabled"`
	DefaultFailurePipelineBudget int    `json:"default_failure_pipeline_budget"`
	InactivityTimeoutSeconds     int    `json:"inactivity_timeout_seconds"`
	EventFlows                   string `json:"event_flows"`
}

type DocumentLevel struct {
	ID    string `json:"id"`
	Code  string `json:"code"`
	Name  string `json:"name"`
	Depth int    `json:"depth"`
}

// DocumentGroupType is a published GroupType — the kinds of node this
// Agency's org uses ("Department", "Ward"), not the nodes themselves. The
// nodes are operational and live in mwanachama-backend-actor's Group tables
// (AGD-017, which replaced the Department section this document used to
// carry).
type DocumentGroupType struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Singular string `json:"singular"`
	Plural   string `json:"plural"`
	LevelID  string `json:"level_id"`
}

type DocumentRole struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DocumentGoal struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
}

type DocumentObjective struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	GoalID      string `json:"goal_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
}

type DocumentWorkflow struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
	EventFlows  string `json:"event_flows"`
}

type DocumentWorkItem struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	WorkflowID  string `json:"workflow_id"`
	ObjectiveID string `json:"objective_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
	Prompt      string `json:"prompt"`
}

type DocumentRoleAssignment struct {
	RoleID      string `json:"role_id"`
	ObjectiveID string `json:"objective_id"`
	Label       string `json:"label"`
}

type DocumentStatement struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Text string `json:"text"`
}

// DocumentAttachment carries both the id and the Code of each end, matching
// the import shape's own both-accepted convention (AG22 stores Codes; an
// export written by id resolves through the same maps).
type DocumentAttachment struct {
	ValueID    string `json:"value_id,omitempty"`
	ValueCode  string `json:"value_code,omitempty"`
	RuleID     string `json:"rule_id,omitempty"`
	RuleCode   string `json:"rule_code,omitempty"`
	TargetType string `json:"target_type"`
	TargetCode string `json:"target_code"`
}

type DocumentModule struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Enabled bool   `json:"enabled"`
}

type DocumentObjectiveModule struct {
	ObjectiveID string `json:"objective_id"`
	ModuleID    string `json:"module_id"`
}
