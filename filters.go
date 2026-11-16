package mwanachamaagency

type GoalFilter struct {
	Code        string `query:"code"`
	Title       string `query:"title"`
	Description string `query:"description"`
}
type ObjectiveFilter struct {
	Code        string `query:"code"`
	Title       string `query:"title"`
	Description string `query:"description"`
}
type WorkflowFilter struct {
	Code        string `query:"code"`
	Name        string `query:"name"`
	Description string `query:"description"`
}
type WorkItemFilter struct {
	Code        string   `query:"code"`
	Title       string   `query:"title"`
	Description string   `query:"description"`
	RoleIDs     []string `query:"-"`
}
type DraftFilter struct {
	Description string `query:"description"`
	Status      string `query:"status"`
}
type PublicationFilter struct {
	Tag     string `query:"tag"`
	Channel string `query:"channel"`
}
type RecordFilter struct {
	Code       string `query:"code"`
	Name       string `query:"name"`
	WorkflowID string `query:"workflow_id"`
}
type RiskFilter struct {
	Code  string `query:"code"`
	Title string `query:"title"`
}
type RiskPathwayFilter struct {
	RiskID string `query:"risk_id"`
	Side   string `query:"side"`
	Title  string `query:"title"`
}
type MeasureFilter struct {
	ObjectiveID string `query:"objective_id"`
}
type RecordRowFilter struct {
	RecordID   string `query:"record_id"`
	RecordCode string `query:"record_code"`
	OrderID    string `query:"order_id"`
}
type EventFlowStepFilter struct {
	Code        string `query:"code"`
	Name        string `query:"name"`
	Description string `query:"description"`
}
type RoleFilter struct {
	Code        string `query:"code"`
	Name        string `query:"name"`
	Description string `query:"description"`
}
type RoleAssignmentFilter struct {
	Label      string `query:"label"`
	WorkItemID string `query:"-"`
	RoleID     string `query:"role_id"`
}
type WorkflowObjectiveFilter struct {
	WorkflowID  string `query:"workflow_id"`
	ObjectiveID string `query:"objective_id"`
}
type ModuleFilter struct {
	Code        string `query:"code"`
	Name        string `query:"name"`
	Description string `query:"description"`
	Enabled     *bool  `query:"enabled"`
}
type ValueFilter struct {
	Code        string `query:"code"`
	Text        string `query:"text"`
	RoleID      string `query:"role_id"`
	GoalID      string `query:"goal_id"`
	ObjectiveID string `query:"-"`
	WorkflowID  string `query:"workflow_id"`
	WorkItemID  string `query:"work_item_id"`
}
type RuleFilter struct {
	Code        string `query:"code"`
	Text        string `query:"text"`
	RoleID      string `query:"role_id"`
	GoalID      string `query:"goal_id"`
	ObjectiveID string `query:"-"`
	WorkflowID  string `query:"workflow_id"`
	WorkItemID  string `query:"work_item_id"`
}
type IssueFilter struct {
	Message    string `query:"message"`
	Severity   string `query:"severity"`
	EntityType string `query:"entity_type"`
}
