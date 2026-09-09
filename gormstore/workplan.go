package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkPlanRow is the GORM row for a [models.WorkPlan].
type WorkPlanRow struct {
	ID string `gorm:"primaryKey"`

	Code             string `gorm:"index"`
	Name             string
	Description      string
	TriggerTopic     string
	PayloadCondition string
	Instructions     string

	AgentID        string
	AgentCode      string
	FunctionCode   string
	FunctionParams string
	HandlerService string

	Enabled    bool
	Ordinality int

	SuccessEvent      string
	FailureEvent      string
	OnFailurePipeline string
	StepTimeout       string

	ReviewStepType     string
	ReviewTriggerTopic string
	ReviewSuccessTopic string
	ReviewFailureTopic string

	CreatedAt string
	UpdatedAt string
	Deleted   bool
}

func (r *WorkPlanRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// WorkPlanToRow converts a domain WorkPlan to its row shape, stamping
// UpdatedAt.
func WorkPlanToRow(w models.WorkPlan) WorkPlanRow {
	return WorkPlanRow{
		ID:                 w.ID,
		Code:               w.Code,
		Name:               w.Name,
		Description:        w.Description,
		TriggerTopic:       w.TriggerTopic,
		PayloadCondition:   w.PayloadCondition,
		Instructions:       w.Instructions,
		AgentID:            w.AgentID,
		AgentCode:          w.AgentCode,
		FunctionCode:       w.FunctionCode,
		FunctionParams:     w.FunctionParams,
		HandlerService:     w.HandlerService,
		Enabled:            w.Enabled,
		Ordinality:         w.Ordinality,
		SuccessEvent:       w.SuccessEvent,
		FailureEvent:       w.FailureEvent,
		OnFailurePipeline:  w.OnFailurePipeline,
		StepTimeout:        w.StepTimeout,
		ReviewStepType:     w.ReviewStepType,
		ReviewTriggerTopic: w.ReviewTriggerTopic,
		ReviewSuccessTopic: w.ReviewSuccessTopic,
		ReviewFailureTopic: w.ReviewFailureTopic,
		CreatedAt:          w.CreatedAt,
		UpdatedAt:          models.NowRFC3339(),
		Deleted:            w.Deleted,
	}
}

// WorkPlanFromRow converts a row back to the domain WorkPlan.
func WorkPlanFromRow(r WorkPlanRow) models.WorkPlan {
	return models.WorkPlan{
		ID:                 r.ID,
		Code:               r.Code,
		Name:               r.Name,
		Description:        r.Description,
		TriggerTopic:       r.TriggerTopic,
		PayloadCondition:   r.PayloadCondition,
		Instructions:       r.Instructions,
		AgentID:            r.AgentID,
		AgentCode:          r.AgentCode,
		FunctionCode:       r.FunctionCode,
		FunctionParams:     r.FunctionParams,
		HandlerService:     r.HandlerService,
		Enabled:            r.Enabled,
		Ordinality:         r.Ordinality,
		SuccessEvent:       r.SuccessEvent,
		FailureEvent:       r.FailureEvent,
		OnFailurePipeline:  r.OnFailurePipeline,
		StepTimeout:        r.StepTimeout,
		ReviewStepType:     r.ReviewStepType,
		ReviewTriggerTopic: r.ReviewTriggerTopic,
		ReviewSuccessTopic: r.ReviewSuccessTopic,
		ReviewFailureTopic: r.ReviewFailureTopic,
		CreatedAt:          r.CreatedAt,
		LastUpdated:        r.UpdatedAt,
		Deleted:            r.Deleted,
	}
}
