package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type reviewAuthors struct {
	mwanachamaagency.AgencyManager
	startedBy, commentedBy string
}

func (r *reviewAuthors) StartReview(_ context.Context, _, by, _ string) (models.ReviewView, error) {
	r.startedBy = by
	return models.ReviewView{}, nil
}

func (r *reviewAuthors) CommentFinding(_ context.Context, _, _, by string) (models.ReviewComment, error) {
	r.commentedBy = by
	return models.ReviewComment{}, nil
}

func routeFor(t *testing.T, manager mwanachamaagency.AgencyManager, action string) Route {
	t.Helper()
	routes := Routes(manager, func(context.Context) string { return "the-session" })
	for _, route := range routes {
		if route.Action == action {
			return route
		}
	}
	t.Fatalf("no route for %s", action)
	return Route{}
}

func post(handler http.HandlerFunc, body string, pathValues map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range pathValues {
		req.SetPathValue(k, v)
	}
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func TestAReviewIsStartedByTheCallerNotTheBody(t *testing.T) {
	manager := &reviewAuthors{}
	route := routeFor(t, manager, "agency.review.start")
	post(route.Handler, `{"draft_id":"d1","started_by":"forged-clerk"}`, nil)
	if manager.startedBy != "the-session" {
		t.Errorf("the review names %q as its starter; want the caller's own id the-session, never the body's forged-clerk", manager.startedBy)
	}
}

func TestAFindingCommentIsWrittenByTheCallerNotTheBody(t *testing.T) {
	manager := &reviewAuthors{}
	route := routeFor(t, manager, "agency.review_comment.create")
	post(route.Handler, `{"body":"a note","author":"forged-clerk"}`, map[string]string{"finding_id": "f1"})
	if manager.commentedBy != "the-session" {
		t.Errorf("the comment names %q as its author; want the caller's own id the-session, never the body's forged-clerk", manager.commentedBy)
	}
}

type recordRowAuthors struct {
	mwanachamaagency.AgencyManager
	addedBy string
}

func (r *recordRowAuthors) CreateRecordRow(_ context.Context, row models.RecordRow) (models.RecordRow, error) {
	r.addedBy = row.AddedBy
	return row, nil
}

func TestARecordRowIsSignedByTheCallerNotTheBody(t *testing.T) {
	manager := &recordRowAuthors{}
	route := routeFor(t, manager, "agency.record_row.create")
	post(route.Handler, `{"record_id":"r1","role_id":"a1","added_by":"forged-clerk"}`, nil)
	if manager.addedBy != "the-session" {
		t.Errorf("the row names %q as its author; want the caller's own id the-session, never the body's forged-clerk", manager.addedBy)
	}
}
