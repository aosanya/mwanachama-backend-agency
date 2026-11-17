package mwanachamaagency

import "github.com/aosanya/mwanachama-backend-agency/models"

func ClearStepKindRegistry() {
	stepKindRegistry.Lock()
	defer stepKindRegistry.Unlock()
	stepKindRegistry.kinds = map[models.StepKind]StepKindHooks{}
	stepKindRegistry.checks = nil
}
