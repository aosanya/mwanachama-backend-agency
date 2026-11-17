package mwanachamaagency

import (
	"context"
	"errors"
	"sync"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

type StepIDMaps struct {
	Workflows map[string]string
	Roles     map[string]string
	Steps     map[string]string
}

type StepLookup interface {
	StepOf(ctx context.Context, draftID, stepID string) (models.WorkItem, bool, error)
}

type StepScanReport interface {
	Add(severity models.IssueSeverity, entityType, entityID, format string, args ...any)
}

type StepKindHooks struct {
	Shape        func(wi models.WorkItem) error
	References   func(ctx context.Context, lookup StepLookup, wi models.WorkItem) error
	Scan         func(report StepScanReport, wi models.WorkItem, steps []models.WorkItem)
	Remap        func(wi *models.WorkItem, ids StepIDMaps)
	ToDocument   func(wi models.WorkItem, out *models.DocumentWorkItem)
	FromDocument func(in models.DocumentWorkItem, row *models.WorkItem, ids StepIDMaps)
	Hash         func(wi models.WorkItem) []string
}

type IssueCheck func(report StepScanReport, steps []models.WorkItem) error

var stepKindRegistry = struct {
	sync.RWMutex
	kinds  map[models.StepKind]StepKindHooks
	checks []IssueCheck
}{kinds: map[models.StepKind]StepKindHooks{}}

func RegisterStepKind(kind models.StepKind, hooks StepKindHooks) {
	stepKindRegistry.Lock()
	defer stepKindRegistry.Unlock()
	if existing, ok := stepKindRegistry.kinds[kind]; ok {
		hooks = chainStepKindHooks(existing, hooks)
	}
	stepKindRegistry.kinds[kind] = hooks
}

func chainStepKindHooks(first, second StepKindHooks) StepKindHooks {
	return StepKindHooks{
		Shape:        chainShape(first.Shape, second.Shape),
		References:   chainReferences(first.References, second.References),
		Scan:         chainScan(first.Scan, second.Scan),
		Remap:        chainRemap(first.Remap, second.Remap),
		ToDocument:   chainToDocument(first.ToDocument, second.ToDocument),
		FromDocument: chainFromDocument(first.FromDocument, second.FromDocument),
		Hash:         chainHash(first.Hash, second.Hash),
	}
}

func chainShape(first, second func(models.WorkItem) error) func(models.WorkItem) error {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(wi models.WorkItem) error {
		if err := first(wi); err != nil {
			return err
		}
		return second(wi)
	}
}

func chainReferences(first, second func(context.Context, StepLookup, models.WorkItem) error) func(context.Context, StepLookup, models.WorkItem) error {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(ctx context.Context, lookup StepLookup, wi models.WorkItem) error {
		if err := first(ctx, lookup, wi); err != nil {
			return err
		}
		return second(ctx, lookup, wi)
	}
}

func chainScan(first, second func(StepScanReport, models.WorkItem, []models.WorkItem)) func(StepScanReport, models.WorkItem, []models.WorkItem) {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(report StepScanReport, wi models.WorkItem, steps []models.WorkItem) {
		first(report, wi, steps)
		second(report, wi, steps)
	}
}

func chainRemap(first, second func(*models.WorkItem, StepIDMaps)) func(*models.WorkItem, StepIDMaps) {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(wi *models.WorkItem, ids StepIDMaps) {
		first(wi, ids)
		second(wi, ids)
	}
}

func chainToDocument(first, second func(models.WorkItem, *models.DocumentWorkItem)) func(models.WorkItem, *models.DocumentWorkItem) {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(wi models.WorkItem, out *models.DocumentWorkItem) {
		first(wi, out)
		second(wi, out)
	}
}

func chainFromDocument(first, second func(models.DocumentWorkItem, *models.WorkItem, StepIDMaps)) func(models.DocumentWorkItem, *models.WorkItem, StepIDMaps) {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(in models.DocumentWorkItem, row *models.WorkItem, ids StepIDMaps) {
		first(in, row, ids)
		second(in, row, ids)
	}
}

func chainHash(first, second func(models.WorkItem) []string) func(models.WorkItem) []string {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(wi models.WorkItem) []string {
		return append(first(wi), second(wi)...)
	}
}

func RegisterIssueCheck(check IssueCheck) {
	stepKindRegistry.Lock()
	defer stepKindRegistry.Unlock()
	stepKindRegistry.checks = append(stepKindRegistry.checks, check)
}

func StepHashRecord(typeName string, fields map[string]any) string {
	return hashRecord(typeName, fields)
}

func hooksOf(kind models.StepKind) (StepKindHooks, bool) {
	stepKindRegistry.RLock()
	defer stepKindRegistry.RUnlock()
	hooks, ok := stepKindRegistry.kinds[kind]
	return hooks, ok
}

func registeredIssueChecks() []IssueCheck {
	stepKindRegistry.RLock()
	defer stepKindRegistry.RUnlock()
	return append([]IssueCheck(nil), stepKindRegistry.checks...)
}

func stepKindRegistered(kind models.StepKind) bool {
	_, ok := hooksOf(kind)
	return ok
}

func shapeOfRegisteredKind(wi models.WorkItem) error {
	hooks, ok := hooksOf(wi.Kind)
	if !ok || hooks.Shape == nil {
		return nil
	}
	err := hooks.Shape(wi)
	if err == nil || errors.Is(err, ErrInvalidWorkItem) {
		return err
	}
	return invalidStep("%s", err.Error())
}

type managerStepLookup struct {
	m  *agencyManager
	tx *gorm.DB
}

func (l managerStepLookup) StepOf(ctx context.Context, draftID, stepID string) (models.WorkItem, bool, error) {
	q := scopeQuery(l.tx.WithContext(ctx).Table(l.m.table(roleWorkItem)), draftID).Where("id = ?", stepID)
	found, err := list[models.WorkItem](l.m, q, roleWorkItem)
	if err != nil || len(found) == 0 {
		return models.WorkItem{}, false, err
	}
	return found[0], true, nil
}

func (m *agencyManager) referencesOfRegisteredKind(ctx context.Context, tx *gorm.DB, wi models.WorkItem) error {
	if hooks, ok := hooksOf(wi.Kind); ok && hooks.References != nil {
		return hooks.References(ctx, managerStepLookup{m: m, tx: tx}, wi)
	}
	return nil
}

type scanReport struct{ s *issueScan }

func (r scanReport) Add(severity models.IssueSeverity, entityType, entityID, format string, args ...any) {
	r.s.add(severity, entityType, entityID, format, args...)
}

func (s *issueScan) scanRegisteredKinds(steps []models.WorkItem) {
	report := scanReport{s: s}
	for _, wi := range steps {
		if hooks, ok := hooksOf(wi.Kind); ok && hooks.Scan != nil {
			hooks.Scan(report, wi, steps)
		}
	}
}

func (s *issueScan) checkRegistered() error {
	checks := registeredIssueChecks()
	if len(checks) == 0 {
		return nil
	}
	steps, err := list[models.WorkItem](s.m, s.scoped(roleWorkItem), roleWorkItem)
	if err != nil {
		return err
	}
	report := scanReport{s: s}
	for _, check := range checks {
		if err := check(report, steps); err != nil {
			return err
		}
	}
	return nil
}

func remapRegisteredKind(wi *models.WorkItem, ids StepIDMaps) {
	if hooks, ok := hooksOf(wi.Kind); ok && hooks.Remap != nil {
		hooks.Remap(wi, ids)
	}
}

func documentOfRegisteredKind(wi models.WorkItem, out *models.DocumentWorkItem) {
	if hooks, ok := hooksOf(wi.Kind); ok && hooks.ToDocument != nil {
		hooks.ToDocument(wi, out)
	}
}

func rowOfRegisteredKind(in models.DocumentWorkItem, row *models.WorkItem, ids StepIDMaps) {
	if hooks, ok := hooksOf(in.Kind); ok && hooks.FromDocument != nil {
		hooks.FromDocument(in, row, ids)
	}
}

func hashRecordsOfRegisteredKind(wi models.WorkItem) []string {
	if hooks, ok := hooksOf(wi.Kind); ok && hooks.Hash != nil {
		return hooks.Hash(wi)
	}
	return nil
}
