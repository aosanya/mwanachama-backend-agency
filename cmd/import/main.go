package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func main() {
	file := flag.String("file", "", "the agency document to import")
	instance := flag.String("instance", "agency", "the instance whose tables it lands in")
	description := flag.String("description", "", "what this draft is for")
	apply := flag.Bool("apply", false, "write it; without this, only report what the document holds")
	flag.Parse()

	if *file == "" {
		fail(fmt.Errorf("-file is required"))
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		fail(err)
	}
	var doc models.AgencyDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		fail(err)
	}

	fmt.Printf("%s: %s\n", *file, doc.Agency.Name)
	for _, c := range counts(doc) {
		fmt.Printf("  %-18s %d\n", c.what, c.n)
	}
	if !*apply {
		fmt.Println("nothing written — pass -apply")
		return
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fail(fmt.Errorf("DATABASE_URL is not set"))
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fail(err)
	}
	s, err := mwanachamaagency.SpecFor(*instance)
	if err != nil {
		fail(err)
	}
	mgr, err := mwanachamaagency.NewAgencyManager(db, s)
	if err != nil {
		fail(err)
	}

	name := *description
	if name == "" {
		name = doc.Agency.Name
	}
	out, err := mgr.ImportDocument(context.Background(), name, doc)
	if err != nil {
		fail(err)
	}
	fmt.Printf("draft %s (%s), %d attachment(s)\n", out.Draft.ID, out.Draft.Description, out.Attachments)

	issues, err := mgr.ListIssues(context.Background(), out.Draft.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		fail(err)
	}
	fmt.Printf("scan: %d finding(s)\n", len(issues))
	for _, issue := range issues {
		fmt.Printf("  %s: %s\n", issue.Severity, issue.Message)
	}
}

type countOf struct {
	what string
	n    int
}

func counts(doc models.AgencyDocument) []countOf {
	return []countOf{
		{"roles", len(doc.Roles)},
		{"goals", len(doc.Goals)},
		{"objectives", len(doc.Objectives)},
		{"workflows", len(doc.Workflows)},
		{"work items", len(doc.WorkItems)},
		{"RACI links", len(doc.RoleAssignments)},
		{"values", len(doc.Values)},
		{"rules", len(doc.Rules)},
		{"attachments", len(doc.ValueAttachments) + len(doc.RuleAttachments)},
		{"modules", len(doc.Modules)},
		{"event flow steps", len(doc.EventFlowSteps)},
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "import: "+strings.TrimSpace(err.Error()))
	os.Exit(1)
}
