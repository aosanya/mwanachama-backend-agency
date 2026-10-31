package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

type hubAgentCall struct {
	tool   string
	action string
	powers []string
}

var hubAgentCalls = []hubAgentCall{
	{"comm_chat_thread_list", "comm.chat_thread.list", []string{"chat.thread"}},
	{"comm_chat_thread_create", "comm.chat_thread.create", []string{"chat.thread"}},
	{"comm_chat_thread_mark_read", "comm.chat_thread.mark_read", []string{"chat.thread"}},
	{"comm_chat_message_list", "comm.chat_message.list", []string{"chat.post"}},
	{"comm_chat_message_post", "comm.chat_message.post", []string{"chat.post"}},
	{"forms_form_create", "forms.form.create", []string{"form.compose"}},
	{"forms_question_create", "forms.question.create", []string{"form.compose"}},
	{"forms_question_list", "forms.question.list", []string{"form.compose", "form.answers.read"}},
	{"forms_form_publish", "forms.form.publish", []string{"form.send"}},
	{"forms_answer_list_all", "forms.answer.list_all", []string{"form.answers.read"}},
	{"sources_source_list", "sources.source.list", []string{"document.search", "document.read"}},
	{"sources_chunk_list", "sources.chunk.list", []string{"document.read"}},
	{"actor_list_groups", "actor.group.list", []string{"group.read"}},
	{"actor_list_actors", "actor.actor.list", []string{"member.read"}},
	{"ai_resolve_agency_role", "ai.role.resolve_agency_role", []string{"ai.use"}},
	{"ai_list_accounts", "ai.account.list", []string{"ai.use"}},
	{"ai_record_usage", "ai.usage.record", []string{"ai.use"}},
	{"ai_check_limits", "ai.limits.check", []string{"ai.use"}},
}

var approvalToolsNoPowerGates = []string{"agency_propose_write", "agency_list_approvals", "agency_amend_approval"}

func powerNamed(t *testing.T, code string) models.Power {
	t.Helper()
	p, ok := models.LookupPower(code)
	if !ok {
		t.Fatalf("power %s is missing from the catalogue", code)
	}
	if p.Locked {
		t.Fatalf("power %s is locked to owners, so no agent role may hold it", code)
	}
	return p
}

func carries(p models.Power, action string) bool {
	for _, a := range p.Actions {
		if a == action {
			return true
		}
	}
	return false
}

func TestEveryCallTheHubAgentsMakeIsCarriedByAnUnlockedPower(t *testing.T) {
	for _, c := range hubAgentCalls {
		t.Run(c.tool, func(t *testing.T) {
			for _, code := range c.powers {
				if !carries(powerNamed(t, code), c.action) {
					t.Fatalf("power %s does not carry %s, so a Hub Steward agent linked to a role holding it is refused %s", code, c.action, c.tool)
				}
			}
		})
	}
}

func TestTheHubAgentPowersCarryExactlyTheirActions(t *testing.T) {
	want := map[string]models.Power{
		"chat.thread": {
			Code: "chat.thread", Name: "Open and keep up with chat threads", Module: "comm",
			Actions: []string{"comm.chat_thread.list", "comm.chat_thread.create", "comm.chat_thread.mark_read"},
		},
		"form.compose": {
			Code: "form.compose", Name: "Compose forms", Module: "forms",
			Actions: []string{"forms.form.create", "forms.question.create", "forms.question.list"},
		},
		"form.answers.read": {
			Code: "form.answers.read", Name: "Read the answers to a form", Module: "forms",
			Actions: []string{"forms.answer.list_all", "forms.question.list"},
		},
		"document.read": {
			Code: "document.read", Name: "Read documents", Module: "sources",
			Actions: []string{"sources.source.list", "sources.source.read", "sources.chunk.list"},
		},
		"group.read": {
			Code: "group.read", Name: "See the hub's rooms",
			Actions: []string{"actor.group.list"},
		},
		"member.read": {
			Code: "member.read", Name: "See who is in the hub",
			Actions: []string{"actor.actor.list"},
		},
	}
	for code, expected := range want {
		got, ok := models.LookupPower(code)
		if !ok {
			t.Fatalf("power %s is missing from the catalogue", code)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("%s = %+v, want %+v", code, got, expected)
		}
	}
}

func TestTheHubAgentPowersGrantNoWriteBeyondWhatTheAgentsDo(t *testing.T) {
	forbidden := map[string][]string{
		"chat.thread":       {"comm.chat_thread.archive", "comm.chat_thread.unarchive", "comm.chat_thread.open", "comm.chat_thread.decline", "comm.chat_message.post"},
		"form.compose":      {"forms.form.publish", "forms.answer.list_all", "forms.target.create", "forms.approval.approve"},
		"form.answers.read": {"forms.form.create", "forms.question.create", "forms.results.publish"},
		"document.read":     {"sources.source.create", "sources.chunk.search"},
		"group.read":        {"actor.group.create", "actor.group.set_join_mode", "actor.assignment.create", "actor.actor.list"},
		"member.read":       {"actor.actor.create", "actor.group.list", "actor.invitation.create"},
	}
	for code, actions := range forbidden {
		p := powerNamed(t, code)
		for _, a := range actions {
			if carries(p, a) {
				t.Errorf("power %s carries %s, which no hub agent needs from it", code, a)
			}
		}
	}
}

func TestTheAIUsePowerLetsAnAgentResolveItsAgencyRoleButNotAnAIRoleByItsOwnIdOrMapOne(t *testing.T) {
	p := powerNamed(t, "ai.use")
	if !carries(p, "ai.role.resolve_agency_role") {
		t.Fatalf("ai.use = %v, want it carrying ai.role.resolve_agency_role so a linked agent can read its role's instructions", p.Actions)
	}
	for _, a := range []string{"ai.role.resolve", "ai.role.map", "ai.role.create", "ai.role.update", "ai.role.list", "ai.role.read"} {
		if carries(p, a) {
			t.Errorf("ai.use carries %s, which only the AI role managers need", a)
		}
	}
	manage := powerNamed(t, "ai.role.manage")
	if !carries(manage, "ai.role.resolve_agency_role") {
		t.Errorf("ai.role.manage lost ai.role.resolve_agency_role: %v", manage.Actions)
	}
}

func TestEarlierPowerCodesAndTheActionsOfChatPostAndDocumentSearchStayAsTheyWere(t *testing.T) {
	for _, code := range []string{
		"report.publish", "form.approve", "join_request.decide", "member.invite", "promise.record_outcome", "poll.run",
		"notice.post", "event.post", "volunteer_call.post", "form.send", "member.remove", "group.settings",
		"board_item.reply", "board_item.hide", "board.manage", "register.edit", "map.edit", "task.assign",
		"service.edit", "service.publish", "appointment.manage", "chat.post", "document.search", "asset.manage",
		"ai.account.manage", "ai.role.manage", "ai.use", "ai.account.read",
		"agent.manage", "api_key.manage", "agency.go_live", "agency.delete", "power.override",
	} {
		if _, ok := models.LookupPower(code); !ok {
			t.Errorf("power %s is gone from the catalogue; stored definitions that name it would be refused at publish", code)
		}
	}
	wantActions := map[string][]string{
		"chat.post":       {"comm.chat_message.post", "comm.chat_message.list", "comm.chat_message.read"},
		"document.search": {"sources.chunk.search", "sources.source.list", "sources.source.read"},
		"form.send":       {"forms.form.publish"},
		"ai.use":          {"ai.account.list", "ai.usage.list", "ai.usage.record", "ai.limits.check", "ai.role.resolve_agency_role", "ai.account.chat", "ai.model.list"},
	}
	for code, actions := range wantActions {
		if got := powerNamed(t, code).Actions; !reflect.DeepEqual(got, actions) {
			t.Errorf("%s actions = %v, want %v", code, got, actions)
		}
	}
}

func TestNoPowerClaimsToGateTheApprovalToolsBecauseNoActionGatesThem(t *testing.T) {
	gated := map[string]bool{}
	for _, tool := range approvalToolsNoPowerGates {
		gated[tool] = true
	}
	for _, p := range models.PowerCatalog() {
		for _, a := range p.Actions {
			if gated[strings.ReplaceAll(a, ".", "_")] {
				t.Errorf("power %s carries %s, but wakala-api checks membership, not a power, before the approval tools", p.Code, a)
			}
			if strings.HasPrefix(a, "agency.approval") {
				t.Errorf("power %s carries %s, but the approval tools have no declared action", p.Code, a)
			}
		}
	}
}

func TestARoleMayDeclareEveryHubAgentPowerAndStillPublish(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, role := seedPublishableAgency(t, mgr, ctx)
	role.Powers = []string{
		"ai.use", "chat.post", "chat.thread", "form.compose", "form.send", "form.answers.read",
		"document.search", "document.read", "group.read", "member.read",
	}
	if _, err := mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	pub, err := mgr.PublishAgency(ctx, o.DraftID, "hub-agent-1", "")
	if err != nil {
		t.Fatalf("PublishAgency refused a role holding the hub agent powers: %v", err)
	}
	if len(pub.Document.Roles) != 1 || !reflect.DeepEqual(pub.Document.Roles[0].Powers, role.Powers) {
		t.Fatalf("document roles = %+v, want powers %v", pub.Document.Roles, role.Powers)
	}
}

func TestThePowerListServesTheHubAgentPowers(t *testing.T) {
	mgr := newTestManager(t)
	powers, err := mgr.ListPowers(context.Background())
	if err != nil {
		t.Fatalf("ListPowers: %v", err)
	}
	have := map[string]bool{}
	for _, p := range powers {
		have[p.Code] = true
	}
	for _, code := range []string{"chat.thread", "form.compose", "form.answers.read", "document.read", "group.read", "member.read"} {
		if !have[code] {
			t.Errorf("GET /powers does not list %s", code)
		}
	}
}

type siblingOperations struct {
	Operations map[string]struct {
		Action string `json:"action"`
		Tool   string `json:"tool"`
	} `json:"operations"`
}

func toolNamesByAction(t *testing.T, path string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the operations file: %v", err)
	}
	var ops siblingOperations
	if err := json.Unmarshal(raw, &ops); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	out := map[string]string{}
	for _, op := range ops.Operations {
		tool := op.Tool
		if tool == "" {
			tool = strings.ReplaceAll(op.Action, ".", "_")
		}
		out[op.Action] = tool
	}
	return out
}

func TestEveryActionOfTheHubAgentPowersIsDeclaredByItsModuleUnderTheToolNameTheAgentsCall(t *testing.T) {
	declared := map[string]string{}
	for _, file := range []string{
		"testdata/operations/comm.operations.json",
		"testdata/operations/forms.operations.json",
		"testdata/operations/sources.operations.json",
		"testdata/operations/actor.operations.json",
	} {
		for action, tool := range toolNamesByAction(t, file) {
			declared[action] = tool
		}
	}
	for _, c := range hubAgentCalls {
		if strings.HasPrefix(c.action, "ai.") {
			continue
		}
		tool, ok := declared[c.action]
		if !ok {
			t.Errorf("no module declares %s, which %s names", c.action, c.tool)
			continue
		}
		if tool != c.tool {
			t.Errorf("%s is served as tool %s, but the agents call %s", c.action, tool, c.tool)
		}
	}
	for _, code := range []string{"chat.thread", "form.compose", "form.answers.read", "document.read", "group.read", "member.read"} {
		for _, a := range powerNamed(t, code).Actions {
			if _, ok := declared[a]; !ok {
				t.Errorf("power %s lists %s, which no module declares", code, a)
			}
		}
	}
}

func TestTheAIActionsTheGatewayAndTheAgentsUseAreDeclaredByTheAIModule(t *testing.T) {
	declared := toolNamesByAction(t, "testdata/operations/ai.operations.json")
	for _, c := range hubAgentCalls {
		if !strings.HasPrefix(c.action, "ai.") {
			continue
		}
		tool, ok := declared[c.action]
		if !ok {
			t.Errorf("the ai module declares no %s", c.action)
			continue
		}
		if tool != c.tool {
			t.Errorf("%s is served as tool %s, want %s", c.action, tool, c.tool)
		}
	}
	for _, a := range powerNamed(t, "ai.use").Actions {
		if _, ok := declared[a]; !ok {
			t.Errorf("ai.use lists %s, which the ai module does not declare", a)
		}
	}
}
