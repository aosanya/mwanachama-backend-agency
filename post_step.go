package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

var ErrAccountNotFound = errors.New("mwanachamaagency: account not found")

const (
	postDirectionDebit  = "debit"
	postDirectionCredit = "credit"
)

type AccountDirectory interface {
	HasAccount(ctx context.Context, draftID, account string) (bool, error)
}

var accountDirectoryHolder = struct {
	sync.RWMutex
	directory AccountDirectory
}{}

func SetAccountDirectory(directory AccountDirectory) {
	accountDirectoryHolder.Lock()
	defer accountDirectoryHolder.Unlock()
	accountDirectoryHolder.directory = directory
}

func currentAccountDirectory() AccountDirectory {
	accountDirectoryHolder.RLock()
	defer accountDirectoryHolder.RUnlock()
	return accountDirectoryHolder.directory
}

func init() {
	registerPostStep()
}

func registerPostStep() {
	RegisterStepKind(models.StepKindPost, StepKindHooks{
		Shape:      postShape,
		References: postReferences,
		Scan:       postScan,
	})
	for _, kind := range stepKindsOtherThanPost() {
		RegisterStepKind(kind, StepKindHooks{Shape: postFieldsOnAnotherKind})
	}
}

func stepKindsOtherThanPost() []models.StepKind {
	others := []models.StepKind{""}
	for _, kind := range models.StepKinds() {
		if kind != models.StepKindPost {
			others = append(others, kind)
		}
	}
	return others
}

func hasPostFields(wi models.WorkItem) bool {
	return wi.AmountStepID != "" || wi.Account != "" || wi.CounterAccount != "" || wi.Direction != "" || wi.Reason != ""
}

func postFieldsOnAnotherKind(wi models.WorkItem) error {
	if hasPostFields(wi) {
		return invalidStep("an amount step, an account, a counter account, a direction and a reason belong to a post step, not to a %s step", kindWord(wi.Kind))
	}
	return nil
}

func kindWord(kind models.StepKind) string {
	if kind == "" {
		return "step without a kind"
	}
	return string(kind)
}

func postShape(wi models.WorkItem) error {
	if wi.Function != "" || len(wi.Bindings) > 0 || wi.ResultName != "" || wi.OnFailure != nil || wi.Hold != nil {
		return invalidStep("a post step takes no calculating fields: it posts the result of a calculating step")
	}
	if wi.Tool != "" {
		return invalidStep("a post step names no tool")
	}
	if wi.Direction != "" && wi.Direction != postDirectionDebit && wi.Direction != postDirectionCredit {
		return invalidStep("the direction of a post step is %s or %s, not %q", postDirectionDebit, postDirectionCredit, wi.Direction)
	}
	if wi.Reason != "" && strings.TrimSpace(wi.Reason) == "" {
		return invalidStep("the reason of a post step is written in words, not left blank")
	}
	if wi.Account != "" && wi.Account == wi.CounterAccount {
		return invalidStep("a post step debits one account and credits another, so %q cannot be both sides", wi.Account)
	}
	return nil
}

func postReferences(ctx context.Context, lookup StepLookup, wi models.WorkItem) error {
	if wi.AmountStepID != "" {
		if err := checkAmountStep(ctx, lookup, wi); err != nil {
			return err
		}
	}
	return checkAccountsExist(ctx, wi)
}

func checkAmountStep(ctx context.Context, lookup StepLookup, wi models.WorkItem) error {
	if wi.ID != "" && wi.AmountStepID == wi.ID {
		return invalidStep("a post step cannot take its amount from itself")
	}
	amount, found, err := lookup.StepOf(ctx, wi.DraftID, wi.AmountStepID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: the amount step %s is not a step of this draft", ErrWorkItemNotFound, wi.AmountStepID)
	}
	switch {
	case amount.WorkflowID != wi.WorkflowID:
		return invalidStep("the amount step %q belongs to another workflow", amount.Title)
	case amount.Ordinality >= wi.Ordinality:
		return invalidStep("the amount step %q is not earlier than this step", amount.Title)
	case amount.Kind != models.StepKindCompute:
		return invalidStep("the amount step %q is not a calculating step: only a calculating step works out an amount", amount.Title)
	}
	return nil
}

func checkAccountsExist(ctx context.Context, wi models.WorkItem) error {
	directory := currentAccountDirectory()
	if directory == nil {
		return nil
	}
	for _, account := range []string{wi.Account, wi.CounterAccount} {
		if account == "" {
			continue
		}
		ok, err := directory.HasAccount(ctx, wi.DraftID, account)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: the agency has no account %q", ErrAccountNotFound, account)
		}
	}
	return nil
}

func postScan(report StepScanReport, wi models.WorkItem, steps []models.WorkItem) {
	if wi.AmountStepID == "" {
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q is a post step with no amount source: it names no calculating step", wi.Code, wi.Title)
	} else {
		scanAmountStep(report, wi, steps)
	}
	for _, missing := range missingPostFields(wi) {
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q is a post step with no %s", wi.Code, wi.Title, missing)
	}
	scanAccountsExist(report, wi)
}

func missingPostFields(wi models.WorkItem) []string {
	var missing []string
	if wi.Account == "" {
		missing = append(missing, "account")
	}
	if wi.CounterAccount == "" {
		missing = append(missing, "counter account")
	}
	if wi.Direction == "" {
		missing = append(missing, "direction")
	}
	if strings.TrimSpace(wi.Reason) == "" {
		missing = append(missing, "reason")
	}
	return missing
}

func scanAmountStep(report StepScanReport, wi models.WorkItem, steps []models.WorkItem) {
	amount, found := stepByID(steps, wi.AmountStepID)
	switch {
	case !found:
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q posts the result of step %s, which does not exist", wi.Code, wi.Title, QuoteIfUnsafe(wi.AmountStepID))
	case amount.WorkflowID != wi.WorkflowID:
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q posts the result of step %s %q, which belongs to another workflow", wi.Code, wi.Title, amount.Code, amount.Title)
	case amount.Ordinality >= wi.Ordinality:
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q posts the result of step %s %q, which is not earlier than it", wi.Code, wi.Title, amount.Code, amount.Title)
	case amount.Kind != models.StepKindCompute:
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q posts the result of step %s %q, which is not a calculating step", wi.Code, wi.Title, amount.Code, amount.Title)
	}
}

func scanAccountsExist(report StepScanReport, wi models.WorkItem) {
	directory := currentAccountDirectory()
	if directory == nil {
		return
	}
	for _, account := range []string{wi.Account, wi.CounterAccount} {
		if account == "" {
			continue
		}
		ok, err := directory.HasAccount(context.Background(), wi.DraftID, account)
		if err == nil && !ok {
			report.Add(models.IssueSeverityError, "work_item", wi.ID,
				"Work item %s %q posts to account %q, which the agency does not have", wi.Code, wi.Title, account)
		}
	}
}

func stepByID(steps []models.WorkItem, id string) (models.WorkItem, bool) {
	for _, step := range steps {
		if step.ID == id {
			return step, true
		}
	}
	return models.WorkItem{}, false
}
