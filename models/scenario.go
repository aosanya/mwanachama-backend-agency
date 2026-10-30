package models

type ScenarioResult string

const (
	ScenarioNotRun  ScenarioResult = "not_run"
	ScenarioRunning ScenarioResult = "running"
	ScenarioPassed  ScenarioResult = "passed"
	ScenarioFailed  ScenarioResult = "failed"
	ScenarioStopped ScenarioResult = "stopped"
)

var scenarioResultLabels = map[ScenarioResult]string{
	ScenarioNotRun:  "Not run yet",
	ScenarioRunning: "Running",
	ScenarioPassed:  "Passed",
	ScenarioFailed:  "Failed",
	ScenarioStopped: "Stopped",
}

func (r ScenarioResult) Valid() bool {
	_, known := scenarioResultLabels[r]
	return known
}

func (r ScenarioResult) Label() string { return scenarioResultLabels[r] }

type ActorKind string

const (
	ActorKindRole   ActorKind = "role"
	ActorKindAgent  ActorKind = "agent"
	ActorKindSystem ActorKind = "system"
)

const SystemActorName = "System"

type Scenario struct {
	ID          string          `json:"id"`
	DraftID     string          `json:"draft_id,omitempty"`
	Name        string          `json:"name"`
	Summary     string          `json:"summary"`
	WorkflowID  string          `json:"workflow_id"`
	Result      ScenarioResult  `json:"result"`
	LastRunID   string          `json:"last_run_id"`
	CreatedAt   string          `json:"created_at"`
	LastUpdated string          `json:"last_updated"`
	Workflow    string          `json:"workflow,omitempty" spec:"-"`
	Rules       []string        `json:"rules,omitempty" spec:"-"`
	Actors      []ScenarioActor `json:"actors,omitempty" spec:"-"`
	Inputs      []ScenarioInput `json:"inputs,omitempty" spec:"-"`
	Blocks      []ScenarioBlock `json:"blocks,omitempty" spec:"-"`
	Thens       []ScenarioThen  `json:"thens,omitempty"`
}

type ScenarioThen struct {
	StepID string `json:"step_id"`
	Then   string `json:"then"`
}

type ScenarioActor struct {
	ID          string    `json:"id"`
	ScenarioID  string    `json:"scenario_id"`
	DraftID     string    `json:"draft_id,omitempty"`
	Name        string    `json:"name"`
	Kind        ActorKind `json:"kind"`
	RoleID      string    `json:"role_id"`
	Ordinality  int       `json:"ordinality"`
	CreatedAt   string    `json:"created_at"`
	LastUpdated string    `json:"last_updated"`
	RoleName    string    `json:"role_name,omitempty" spec:"-"`
	BuiltIn     bool      `json:"built_in,omitempty" spec:"-"`
}

type ScenarioInput struct {
	ID          string `json:"id"`
	ScenarioID  string `json:"scenario_id"`
	DraftID     string `json:"draft_id,omitempty"`
	StepID      string `json:"step_id"`
	Field       string `json:"field"`
	Value       string `json:"value"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
}

type BlockField struct {
	Prompt string `json:"prompt"`
	Free   bool   `json:"free"`
	Value  string `json:"value,omitempty"`
}

type ScenarioBlock struct {
	StepID string       `json:"step_id"`
	Code   string       `json:"code"`
	Title  string       `json:"title"`
	Kind   StepKind     `json:"kind"`
	Action string       `json:"action"`
	Actor  string       `json:"actor"`
	Then   string       `json:"then,omitempty"`
	Fields []BlockField `json:"fields,omitempty"`
}

type TestRunResult string

const (
	TestRunRunning TestRunResult = "running"
	TestRunPassed  TestRunResult = "passed"
	TestRunFailed  TestRunResult = "failed"
	TestRunStopped TestRunResult = "stopped"
)

func (r TestRunResult) Valid() bool {
	return r == TestRunRunning || r == TestRunPassed || r == TestRunFailed || r == TestRunStopped
}

func (r TestRunResult) Finished() bool {
	return r == TestRunPassed || r == TestRunFailed || r == TestRunStopped
}

func (r TestRunResult) ScenarioResult() ScenarioResult { return ScenarioResult(r) }

type RuleCheck struct {
	Code string `json:"code"`
	Held bool   `json:"held"`
}

type RunStep struct {
	StepID   string      `json:"step_id"`
	Step     string      `json:"step"`
	Actor    string      `json:"actor"`
	PlayedBy string      `json:"played_by,omitempty"`
	Did      string      `json:"did"`
	Recorded string      `json:"recorded"`
	Rules    []RuleCheck `json:"rules,omitempty"`
	Tokens   int         `json:"tokens"`
	Failed   bool        `json:"failed,omitempty"`
	Expected string      `json:"expected,omitempty"`
	Found    string      `json:"found,omitempty"`
}

type TestRun struct {
	ID           string        `json:"id"`
	ScenarioID   string        `json:"scenario_id"`
	DraftID      string        `json:"draft_id,omitempty"`
	DraftVersion string        `json:"draft_version"`
	Result       TestRunResult `json:"result"`
	StopReason   string        `json:"stop_reason"`
	Tokens       int           `json:"tokens"`
	Steps        []RunStep     `json:"steps,omitempty"`
	StartedAt    string        `json:"started_at"`
	FinishedAt   string        `json:"finished_at"`
}

type DocumentScenarioActor struct {
	Name   string `json:"name"`
	RoleID string `json:"role_id"`
}

type DocumentScenarioInput struct {
	StepID string `json:"step_id"`
	Field  string `json:"field"`
	Value  string `json:"value"`
}

type DocumentScenario struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Summary    string                  `json:"summary"`
	WorkflowID string                  `json:"workflow_id"`
	Actors     []DocumentScenarioActor `json:"actors,omitempty"`
	Inputs     []DocumentScenarioInput `json:"inputs,omitempty"`
	Thens      []ScenarioThen          `json:"thens,omitempty"`
}
