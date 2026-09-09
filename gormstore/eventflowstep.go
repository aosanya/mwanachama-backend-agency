package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EventFlowStepRow is the GORM row for a [models.EventFlowStep]. DraftID
// is nullable (NULL for live), same convention as GoalRow/WorkflowRow.
type EventFlowStepRow struct {
	ID                 string  `gorm:"primaryKey"`
	DraftID            *string `gorm:"index"`
	Code               string  `gorm:"index"`
	WorkflowCode       string  `gorm:"index"`
	Step               string
	Name               string
	Description        string
	StepType           string
	TriggerTopic       string
	TriggerPublisher   string
	Consumer           string
	HandlerCode        string `gorm:"index"`
	EmitsTopics        string
	OnErrorEmitsTopics string
	Ordinality         int
	CreatedAt          string
	UpdatedAt          string
	Deleted            bool
}

func (r *EventFlowStepRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// EventFlowStepToRow converts a domain EventFlowStep to its row shape,
// stamping UpdatedAt.
func EventFlowStepToRow(s models.EventFlowStep) EventFlowStepRow {
	return EventFlowStepRow{
		ID:                 s.ID,
		DraftID:            StringToNullable(s.DraftID),
		Code:               s.Code,
		WorkflowCode:       s.WorkflowCode,
		Step:               s.Step,
		Name:               s.Name,
		Description:        s.Description,
		StepType:           s.StepType,
		TriggerTopic:       s.TriggerTopic,
		TriggerPublisher:   s.TriggerPublisher,
		Consumer:           s.Consumer,
		HandlerCode:        s.HandlerCode,
		EmitsTopics:        s.EmitsTopics,
		OnErrorEmitsTopics: s.OnErrorEmitsTopics,
		Ordinality:         s.Ordinality,
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          models.NowRFC3339(),
		Deleted:            s.Deleted,
	}
}

// EventFlowStepFromRow converts a row back to the domain EventFlowStep.
func EventFlowStepFromRow(r EventFlowStepRow) models.EventFlowStep {
	return models.EventFlowStep{
		ID:                 r.ID,
		DraftID:            nullableToString(r.DraftID),
		Code:               r.Code,
		WorkflowCode:       r.WorkflowCode,
		Step:               r.Step,
		Name:               r.Name,
		Description:        r.Description,
		StepType:           r.StepType,
		TriggerTopic:       r.TriggerTopic,
		TriggerPublisher:   r.TriggerPublisher,
		Consumer:           r.Consumer,
		HandlerCode:        r.HandlerCode,
		EmitsTopics:        r.EmitsTopics,
		OnErrorEmitsTopics: r.OnErrorEmitsTopics,
		Ordinality:         r.Ordinality,
		CreatedAt:          r.CreatedAt,
		LastUpdated:        r.UpdatedAt,
		Deleted:            r.Deleted,
	}
}
