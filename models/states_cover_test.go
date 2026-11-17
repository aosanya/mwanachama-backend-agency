package models

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestDraftStatus_OnlyAnOpenDraftCanBeArchived(t *testing.T) {
	cases := []struct {
		from, to DraftStatus
		want     bool
	}{
		{DraftStatusOpen, DraftStatusArchived, true},
		{DraftStatusOpen, DraftStatusOpen, false},
		{DraftStatusArchived, DraftStatusOpen, false},
		{DraftStatusArchived, DraftStatusArchived, false},
		{DraftStatus("promoted"), DraftStatusArchived, false},
		{DraftStatusOpen, DraftStatus("promoted"), false},
	}
	for _, c := range cases {
		if got := c.from.CanTransitionTo(c.to); got != c.want {
			t.Errorf("%q to %q = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestIssueSeverity_ValidIsExactlyTheThreeNamedSeverities(t *testing.T) {
	for _, s := range []IssueSeverity{IssueSeverityInfo, IssueSeverityWarning, IssueSeverityError} {
		if !s.Valid() {
			t.Errorf("%q must be valid", s)
		}
	}
	for _, s := range []IssueSeverity{"", "fatal", "ERROR", "Warning"} {
		if s.Valid() {
			t.Errorf("%q must be refused", s)
		}
	}
}

func TestControlVocabularies_AcceptOnlyTheirDeclaredValues(t *testing.T) {
	for _, b := range []ControlOnBreach{OnBreachBlock, OnBreachHoldForApproval, OnBreachFlag, OnBreachRunWorkflow} {
		if !b.Valid() {
			t.Errorf("on-breach %q must be valid", b)
		}
	}
	for _, s := range []ControlSeverity{SeverityLow, SeverityMedium, SeverityHigh} {
		if !s.Valid() {
			t.Errorf("severity %q must be valid", s)
		}
	}
	for _, o := range []ControlOperator{OperatorEq, OperatorNe, OperatorGt, OperatorGte, OperatorLt, OperatorLte} {
		if !o.Valid() {
			t.Errorf("operator %q must be valid", o)
		}
	}
	if ControlOnBreach("").Valid() || ControlOnBreach("ignore").Valid() {
		t.Error("an empty or unknown on-breach must be refused")
	}
	if ControlSeverity("").Valid() || ControlSeverity("critical").Valid() {
		t.Error("an empty or unknown severity must be refused")
	}
	if ControlOperator("").Valid() || ControlOperator("==").Valid() || ControlOperator("contains").Valid() {
		t.Error("an empty or unknown operator must be refused")
	}
}

func TestControl_IsEnforcedExactlyWhenItNamesAnAction(t *testing.T) {
	if (Control{Text: "advice only"}).Enforced() {
		t.Error("a control with no action must not be enforced")
	}
	if !(Control{Action: "post.entry"}).Enforced() {
		t.Error("a control naming an action must be enforced")
	}
}

func TestTargetType_ValidForControlAddsTheRiskSidesToTheFiveBaseTypes(t *testing.T) {
	for _, tt := range []TargetType{TargetTypeRole, TargetTypeGoal, TargetTypeObjective, TargetTypeWorkflow, TargetTypeWorkItem} {
		if !tt.Valid() || !tt.ValidForControl() {
			t.Errorf("%q must be valid for values, rules and controls", tt)
		}
	}
	for _, tt := range []TargetType{TargetTypeRiskPathway, TargetTypeThreat, TargetTypeConsequence} {
		if tt.Valid() {
			t.Errorf("%q must not attach a value or rule", tt)
		}
		if !tt.ValidForControl() {
			t.Errorf("%q must take a control", tt)
		}
	}
	for _, tt := range []TargetType{"", "agency", "Role"} {
		if tt.Valid() || tt.ValidForControl() {
			t.Errorf("%q must be refused everywhere", tt)
		}
	}
}

func TestRACILabelAndRiskSideAndOpensIn_RefuseUnknownValues(t *testing.T) {
	for _, l := range []RACILabel{RACILabelResponsible, RACILabelAccountable, RACILabelConsulted, RACILabelInformed} {
		if !l.Valid() {
			t.Errorf("label %q must be valid", l)
		}
	}
	if RACILabel("").Valid() || RACILabel("Responsible").Valid() || RACILabel("owner").Valid() {
		t.Error("an empty, wrongly cased or unknown RACI label must be refused")
	}
	if !RiskSideThreat.Valid() || !RiskSideConsequence.Valid() || RiskSide("").Valid() || RiskSide("barrier").Valid() {
		t.Error("only threat and consequence are risk sides")
	}
	for _, o := range []OpensIn{"", OpensInForm, OpensInChat} {
		if !o.Valid() {
			t.Errorf("opens-in %q must be valid", o)
		}
	}
	if OpensIn("video").Valid() {
		t.Error("an unknown opens-in must be refused")
	}
}

func TestOpensIn_FormIsTheDefaultSoItNormalizesToEmpty(t *testing.T) {
	if OpensInForm.Normalized() != "" {
		t.Error("form must normalize to empty")
	}
	if OpensInChat.Normalized() != OpensInChat {
		t.Error("chat must stay chat")
	}
	if OpensIn("").Normalized() != "" {
		t.Error("empty must stay empty")
	}
	if !(Workflow{OpensIn: OpensInChat}).IsChat() || (Workflow{OpensIn: OpensInForm}).IsChat() || (Workflow{}).IsChat() {
		t.Error("only a chat workflow answers IsChat")
	}
}

func TestStepKind_ValidityFormTakingAndAgencyOnlyRules(t *testing.T) {
	if len(StepKinds()) != 9 {
		t.Errorf("there are %d step kinds, want 9", len(StepKinds()))
	}
	for _, k := range StepKinds() {
		if !k.Valid() {
			t.Errorf("listed kind %q reports invalid", k)
		}
	}
	if StepKind("").Valid() || StepKind("teleport").Valid() {
		t.Error("an empty or unknown kind must be refused")
	}
	formTakers := map[StepKind]bool{StepKindFillForm: true, StepKindSendForm: true, StepKindNotify: true, StepKindYesNo: true}
	agencyOnly := map[StepKind]bool{StepKindSendForm: true, StepKindRoute: true}
	for _, k := range StepKinds() {
		if k.TakesForm() != formTakers[k] {
			t.Errorf("%q TakesForm = %v, want %v", k, k.TakesForm(), formTakers[k])
		}
		if k.AgencyOnly() != agencyOnly[k] {
			t.Errorf("%q AgencyOnly = %v, want %v", k, k.AgencyOnly(), agencyOnly[k])
		}
	}
}

func TestStepKindsAndAnswerTypes_ReturnCopiesCallersCannotCorrupt(t *testing.T) {
	kinds := StepKinds()
	kinds[0] = "tampered"
	if StepKinds()[0] == "tampered" {
		t.Error("changing the returned kinds changed the registry")
	}
	answers := AnswerTypes()
	if len(answers) != 10 {
		t.Errorf("there are %d answer types, want 10", len(answers))
	}
	answers[0] = "tampered"
	if AnswerTypes()[0] == "tampered" {
		t.Error("changing the returned answer types changed the registry")
	}
}

func TestAnswerType_OnlyChoiceTypesHaveOptions(t *testing.T) {
	for _, a := range AnswerTypes() {
		want := a == AnswerSingleChoice || a == AnswerMultiSelect
		if a.HasOptions() != want {
			t.Errorf("%q HasOptions = %v, want %v", a, a.HasOptions(), want)
		}
		if !a.Valid() {
			t.Errorf("listed answer type %q reports invalid", a)
		}
	}
	if AnswerType("").Valid() || AnswerType("signature").Valid() {
		t.Error("an empty or unknown answer type must be refused")
	}
}

func TestScenarioResult_LabelsAreReadableAndUnknownHasNone(t *testing.T) {
	labels := map[ScenarioResult]string{
		ScenarioNotRun: "Not run yet", ScenarioWaiting: "Waiting", ScenarioRunning: "Running",
		ScenarioPassed: "Passed", ScenarioFailed: "Failed", ScenarioStopped: "Stopped",
	}
	for r, want := range labels {
		if !r.Valid() || r.Label() != want {
			t.Errorf("%q valid=%v label=%q, want label %q", r, r.Valid(), r.Label(), want)
		}
	}
	if ScenarioResult("").Valid() || ScenarioResult("").Label() != "" || ScenarioResult("lost").Valid() {
		t.Error("an unknown result must be invalid and have no label")
	}
}

func TestTestRunResult_FinishedMeansPassedFailedOrStoppedAndMapsOntoAScenarioResult(t *testing.T) {
	finished := map[TestRunResult]bool{TestRunPassed: true, TestRunFailed: true, TestRunStopped: true}
	for _, r := range []TestRunResult{TestRunWaiting, TestRunRunning, TestRunPassed, TestRunFailed, TestRunStopped} {
		if !r.Valid() {
			t.Errorf("%q must be valid", r)
		}
		if r.Finished() != finished[r] {
			t.Errorf("%q Finished = %v, want %v", r, r.Finished(), finished[r])
		}
		if !r.ScenarioResult().Valid() || string(r.ScenarioResult()) != string(r) {
			t.Errorf("%q maps to scenario result %q", r, r.ScenarioResult())
		}
	}
	if TestRunResult("").Valid() || TestRunResult("not_run").Valid() || TestRunResult("").Finished() {
		t.Error("an empty result or the scenario-only not_run must not be a test run result")
	}
}

func TestReviewVocabularies_AcceptOnlyTheirDeclaredValues(t *testing.T) {
	for _, a := range ReviewAreaOrder {
		if !a.Valid() {
			t.Errorf("area %q must be valid", a)
		}
	}
	if ReviewAreaName("").Valid() || ReviewAreaName("budget").Valid() {
		t.Error("an unknown review area must be refused")
	}
	for _, s := range []FindingStatus{FindingOpen, FindingApplied, FindingDismissed} {
		if !s.Valid() {
			t.Errorf("finding status %q must be valid", s)
		}
	}
	if FindingStatus("").Valid() || FindingStatus("closed").Valid() {
		t.Error("an unknown finding status must be refused")
	}
	for _, f := range []FindingFate{FateNew, FateFixed, FateStillOpen, FateWithdrawn} {
		if !f.Valid() {
			t.Errorf("fate %q must be valid", f)
		}
	}
	if FindingFate("").Valid() || FindingFate("lost").Valid() {
		t.Error("an unknown fate must be refused")
	}
}

func TestNowRFC3339_IsFixedWidthUTCAndSortsChronologically(t *testing.T) {
	stamp := NowRFC3339()
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{9}Z$`).MatchString(stamp) {
		t.Errorf("stamp %q is not nine-digit UTC", stamp)
	}
	if strings.Compare(stamp, NowRFC3339()) > 0 {
		t.Error("a later stamp sorted before an earlier one")
	}
}

func TestCarriedKeys_AFieldIsCarriedOnlyWhenTheRequestNamedIt(t *testing.T) {
	var role Role
	if err := json.Unmarshal([]byte(`{"name":"Clerk","staff":false}`), &role); err != nil {
		t.Fatal(err)
	}
	if !role.FromRequest() {
		t.Error("a decoded role must know it came from a request")
	}
	if !role.Carries("staff") {
		t.Error("an explicit false must count as carried")
	}
	if role.Carries("description") {
		t.Error("an omitted field must not be carried")
	}
}

func TestCarriedKeys_AValueBuiltInGoCarriesEveryField(t *testing.T) {
	role := Role{Name: "Clerk"}
	if role.FromRequest() {
		t.Error("a role built in Go did not come from a request")
	}
	if !role.Carries("anything") {
		t.Error("with no request every field counts as carried")
	}
}

func TestCarriedKeys_WorkflowAndWorkItemRecordKeysAndRefuseBrokenJSON(t *testing.T) {
	var wf Workflow
	if err := json.Unmarshal([]byte(`{"name":"Intake","opens_in":"chat"}`), &wf); err != nil {
		t.Fatal(err)
	}
	if !wf.IsChat() || !wf.Carries("opens_in") || wf.Carries("cancels_order") {
		t.Errorf("workflow decoded wrongly: %+v", wf)
	}
	var item WorkItem
	if err := json.Unmarshal([]byte(`{"title":"Step"}`), &item); err != nil {
		t.Fatal(err)
	}
	if !item.Carries("title") || item.Carries("topic") {
		t.Error("work item carried keys wrong")
	}
	for _, target := range []json.Unmarshaler{&Workflow{}, &WorkItem{}, &Role{}} {
		if err := target.UnmarshalJSON([]byte(`{"name":`)); err == nil {
			t.Errorf("%T accepted truncated json", target)
		}
		if err := target.UnmarshalJSON([]byte(`[1]`)); err == nil {
			t.Errorf("%T accepted an array", target)
		}
	}
}

func TestPermissionCatalog_ReturnsACopyAndDistinctPermissionsKeepsFirstSeenOrder(t *testing.T) {
	first := PermissionCatalog()
	if len(first) == 0 {
		t.Fatal("the permission catalog is empty")
	}
	first[0].Code = "tampered"
	if PermissionCatalog()[0].Code == "tampered" {
		t.Error("changing the returned catalog changed the registry")
	}
	if got := DistinctPermissions(nil); got != nil {
		t.Errorf("nil gave %v, want nil", got)
	}
	got := DistinctPermissions([]string{"b", "a", "b", "c", "a"})
	if strings.Join(got, ",") != "b,a,c" {
		t.Errorf("distinct = %v, want b,a,c", got)
	}
	if got := DistinctPermissions([]string{}); got == nil || len(got) != 0 {
		t.Errorf("empty gave %v, want an empty non-nil list", got)
	}
}
