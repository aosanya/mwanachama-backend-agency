package models

import (
	"slices"
	"testing"
)

var ruleReadActions = []string{
	"taskmanager.rule.read",
	"taskmanager.rule.list",
	"taskmanager.rule.issue_list",
	"taskmanager.rule.list_for_issue",
	"taskmanager.rule.file_list",
	"taskmanager.rule.produced_list",
	"taskmanager.rule.overlap_list",
	"taskmanager.backlog.rules_list",
	"taskmanager.backlog.decisions_list",
	"taskmanager.backlog.questions_list",
}

var ruleWriteActions = []string{
	"taskmanager.rule.create",
	"taskmanager.rule.update",
	"taskmanager.rule.set_status",
	"taskmanager.rule.supersede",
	"taskmanager.rule.answer",
	"taskmanager.rule.link",
	"taskmanager.rule.unlink",
	"taskmanager.rule.file_add",
	"taskmanager.rule.file_remove",
	"taskmanager.backlog_tag.add",
}

func roleMayCall(permissions []string, action string) bool {
	for _, code := range permissions {
		p, ok := LookupPermission(code)
		if ok && slices.Contains(p.Actions, action) {
			return true
		}
	}
	return false
}

func TestPermissionCatalog_RuleReadCarriesEveryRuleAndBacklogListRead(t *testing.T) {
	requirePermissionCarries(t, "rule.read", ruleReadActions...)
}

func TestPermissionCatalog_RuleEditCarriesEveryRuleWriteAndTheBacklogTagAdd(t *testing.T) {
	requirePermissionCarries(t, "rule.edit", ruleWriteActions...)
}

func TestPermissionCatalog_RuleReadAndRuleEditAreGrantableTaskmanagerPermissions(t *testing.T) {
	for _, code := range []string{"rule.read", "rule.edit"} {
		p, ok := LookupPermission(code)
		if !ok {
			t.Fatalf("catalogue lacks %s", code)
		}
		if p.Locked {
			t.Errorf("%s is locked, want it givable to a role", code)
		}
		if p.Module != "taskmanager" {
			t.Errorf("%s module = %q, want taskmanager", code, p.Module)
		}
	}
}

func TestPermissionCatalog_RuleReadCarriesNoWriteAndRuleEditNoRead(t *testing.T) {
	read, _ := LookupPermission("rule.read")
	edit, _ := LookupPermission("rule.edit")
	for _, a := range ruleWriteActions {
		if slices.Contains(read.Actions, a) {
			t.Errorf("rule.read carries the write %s", a)
		}
	}
	for _, a := range ruleReadActions {
		if slices.Contains(edit.Actions, a) {
			t.Errorf("rule.edit carries the read %s", a)
		}
	}
}

func TestPermissionCatalog_ARoleHoldingTheRulePermissionsMayCallEveryNewAction(t *testing.T) {
	held := []string{"rule.read", "rule.edit"}
	for _, a := range slices.Concat(ruleReadActions, ruleWriteActions) {
		if !roleMayCall(held, a) {
			t.Errorf("a role holding %v is refused %s", held, a)
		}
	}
}

func TestPermissionCatalog_ARoleWithoutTheRulePermissionsIsRefusedEveryNewAction(t *testing.T) {
	for name, held := range map[string][]string{
		"none":               {},
		"task permissions":   {"task.read", "task.edit", "task.assign"},
		"backlog and cases":  {"backlog.manage", "case.vote"},
		"read of rules only": {"rule.read"},
		"edit of rules only": {"rule.edit"},
	} {
		for _, a := range slices.Concat(ruleReadActions, ruleWriteActions) {
			want := slices.Contains(ruleReadActions, a) && slices.Contains(held, "rule.read") ||
				slices.Contains(ruleWriteActions, a) && slices.Contains(held, "rule.edit")
			if got := roleMayCall(held, a); got != want {
				t.Errorf("role with %s: may call %s = %v, want %v", name, a, got, want)
			}
		}
	}
}

func TestPermissionCatalog_NoOtherPermissionCarriesTheNewRuleActions(t *testing.T) {
	for _, p := range PermissionCatalog() {
		if p.Code == "rule.read" || p.Code == "rule.edit" {
			continue
		}
		for _, a := range slices.Concat(ruleReadActions, ruleWriteActions) {
			if slices.Contains(p.Actions, a) {
				t.Errorf("%s also carries %s", p.Code, a)
			}
		}
	}
}
