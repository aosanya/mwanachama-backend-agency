package routes

import (
	"fmt"
	"sort"
	"sync"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

type Route = httpwire.Route

var operations = sync.OnceValues(func() (*dispatch.Spec, error) {
	return dispatch.Parse(mwanachamaagency.Operations())
})

var sentinels = map[string]error{
	"ErrAgencyNotFound":           mwanachamaagency.ErrAgencyNotFound,
	"ErrAgencyExists":             mwanachamaagency.ErrAgencyExists,
	"ErrAmbiguousLookup":          mwanachamaagency.ErrAmbiguousLookup,
	"ErrCannotEditLive":           mwanachamaagency.ErrCannotEditLive,
	"ErrConflict":                 mwanachamaagency.ErrConflict,
	"ErrDraftNotFound":            mwanachamaagency.ErrDraftNotFound,
	"ErrDraftNotOpen":             mwanachamaagency.ErrDraftNotOpen,
	"ErrDraftRequired":            mwanachamaagency.ErrDraftRequired,
	"ErrDuplicateAccountable":     mwanachamaagency.ErrDuplicateAccountable,
	"ErrDuplicateAttachment":      mwanachamaagency.ErrDuplicateAttachment,
	"ErrDuplicateModule":          mwanachamaagency.ErrDuplicateModule,
	"ErrDuplicateObjectiveModule": mwanachamaagency.ErrDuplicateObjectiveModule,
	"ErrDuplicateTag":             mwanachamaagency.ErrDuplicateTag,
	"ErrEventFlowStepNotFound":    mwanachamaagency.ErrEventFlowStepNotFound,
	"ErrGoalNotFound":             mwanachamaagency.ErrGoalNotFound,
	"ErrInvalidForkSource":        mwanachamaagency.ErrInvalidForkSource,
	"ErrLimitExceeded":            mwanachamaagency.ErrLimitExceeded,
	"ErrInvalidLookup":            mwanachamaagency.ErrInvalidLookup,
	"ErrInvalidModule":            mwanachamaagency.ErrInvalidModule,
	"ErrInvalidRACILabel":         mwanachamaagency.ErrInvalidRACILabel,
	"ErrInvalidRole":              mwanachamaagency.ErrInvalidRole,
	"ErrInvalidRule":              mwanachamaagency.ErrInvalidRule,
	"ErrInvalidTargetType":        mwanachamaagency.ErrInvalidTargetType,
	"ErrInvalidValue":             mwanachamaagency.ErrInvalidValue,
	"ErrInvalidWorkItem":          mwanachamaagency.ErrInvalidWorkItem,
	"ErrModuleNotFound":           mwanachamaagency.ErrModuleNotFound,
	"ErrNoChangesDetected":        mwanachamaagency.ErrNoChangesDetected,
	"ErrObjectiveModuleNotFound":  mwanachamaagency.ErrObjectiveModuleNotFound,
	"ErrObjectiveNotFound":        mwanachamaagency.ErrObjectiveNotFound,
	"ErrPublicationNotFound":      mwanachamaagency.ErrPublicationNotFound,
	"ErrPublishValidationFailed":  mwanachamaagency.ErrPublishValidationFailed,
	"ErrRationaleRequired":        mwanachamaagency.ErrRationaleRequired,
	"ErrRoleAssignmentNotFound":   mwanachamaagency.ErrRoleAssignmentNotFound,
	"ErrRoleNotFound":             mwanachamaagency.ErrRoleNotFound,
	"ErrRuleAttachmentNotFound":   mwanachamaagency.ErrRuleAttachmentNotFound,
	"ErrRuleNotFound":             mwanachamaagency.ErrRuleNotFound,
	"ErrValueAttachmentNotFound":  mwanachamaagency.ErrValueAttachmentNotFound,
	"ErrValueNotFound":            mwanachamaagency.ErrValueNotFound,
	"ErrWorkItemNotFound":         mwanachamaagency.ErrWorkItemNotFound,
	"ErrWorkflowNotFound":         mwanachamaagency.ErrWorkflowNotFound,
	"ErrInvalidMeasure":           mwanachamaagency.ErrInvalidMeasure,
	"ErrInvalidRecord":            mwanachamaagency.ErrInvalidRecord,
	"ErrInvalidRecordRow":         mwanachamaagency.ErrInvalidRecordRow,
	"ErrRecordNotFound":           mwanachamaagency.ErrRecordNotFound,
	"ErrRecordRowRefused":         mwanachamaagency.ErrRecordRowRefused,
	"ErrInvalidModuleMounts":      mwanachamaagency.ErrInvalidModuleMounts,
	"ErrScenarioNotFound":         mwanachamaagency.ErrScenarioNotFound,
	"ErrInvalidScenario":          mwanachamaagency.ErrInvalidScenario,
	"ErrScenarioActorNotFound":    mwanachamaagency.ErrScenarioActorNotFound,
	"ErrDuplicateScenarioActor":   mwanachamaagency.ErrDuplicateScenarioActor,
	"ErrScenarioRunning":          mwanachamaagency.ErrScenarioRunning,
	"ErrTestRunNotFound":          mwanachamaagency.ErrTestRunNotFound,
	"ErrTestRunFinished":          mwanachamaagency.ErrTestRunFinished,
	"ErrUsedByScenario":           mwanachamaagency.ErrUsedByScenario,
}

func Build(m mwanachamaagency.AgencyManager) ([]Route, error) {
	s, err := operations()
	if err != nil {
		return nil, err
	}
	return dispatch.Dispatch(s, dispatch.Deps{Manager: m, Errors: sentinels})
}

func Routes(m mwanachamaagency.AgencyManager) []Route {
	out, err := Build(m)
	if err != nil {
		panic(fmt.Sprintf("agency routes: %v", err))
	}
	return out
}

func Shape() []Route {
	s, err := operations()
	if err != nil {
		panic(fmt.Sprintf("agency routes: %v", err))
	}
	names := make([]string, 0, len(s.Operations))
	for name := range s.Operations {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]Route, 0, len(names))
	for _, name := range names {
		op := s.Operations[name]
		out = append(out, Route{Method: op.Method, Path: s.Base + op.Path, Action: op.Action})
	}
	return out
}
