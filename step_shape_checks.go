package mwanachamaagency

import (
	"errors"
	"sync"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

var stepShapeChecks = struct {
	sync.RWMutex
	checks []func(wi models.WorkItem) error
}{}

func RegisterStepShapeCheck(check func(wi models.WorkItem) error) {
	stepShapeChecks.Lock()
	defer stepShapeChecks.Unlock()
	stepShapeChecks.checks = append(stepShapeChecks.checks, check)
}

func sharedShapeOfRegisteredKinds(wi models.WorkItem) error {
	stepShapeChecks.RLock()
	checks := append([]func(wi models.WorkItem) error(nil), stepShapeChecks.checks...)
	stepShapeChecks.RUnlock()
	for _, check := range checks {
		if err := check(wi); err != nil {
			if errors.Is(err, ErrInvalidWorkItem) {
				return err
			}
			return invalidStep("%s", err.Error())
		}
	}
	return nil
}
