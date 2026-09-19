package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerAgencyTools registers Agency/Draft/Publication tools — the
// Agency lifecycle: the working design itself, the draft an edit happens
// against, and the published, channel-served document production runs on.
func registerAgencyTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_agency", Title: "Create Agency", Description: "Create the Agency document (name) for a registered Instance."}, mcpCreateAgency(resolve))
	mcp.AddTool(server, &mcp.Tool{
		Name: "agency_get_agency", Title: "Get Agency", Description: "Get an Instance's Agency document.",
		Meta: dashboardUIMeta,
	}, mcpGetAgency(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_draft", Title: "Create Draft", Description: "Start a new Draft — the unit every Goal/Workflow/WorkItem/Role/RoleAssignment/EventFlowStep row belongs to. Start empty, fork another open Draft, or fork a published tag to revisit what an earlier release contained."}, mcpCreateDraft(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_draft", Title: "Get Draft", Description: "Get a Draft by id."}, mcpGetDraft(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_drafts", Title: "List Drafts", Description: "List every Draft for an Instance."}, mcpListDrafts(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_draft_description", Title: "Update Draft Description", Description: "Change a Draft's description."}, mcpUpdateDraftDescription(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_archive_draft", Title: "Archive Draft", Description: "Archive an open Draft — its scoped rows are left in place as harmless history."}, mcpArchiveDraft(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_publish_agency", Title: "Publish Agency", Description: "Publish a Draft: freeze its current rows into a new versioned AgencyPublication and point a live channel at it. The publication carries the entire Agency, and production operates from it — nothing running reads Draft rows directly. Pass the Draft to publish, a tag to name the release, and a channel to choose which environment it serves (default production). The Draft is validated first: if it has any error-severity issues the publish is refused and the errors are returned — call agency_list_issues with that draft_id to see them in full."}, mcpPublishAgency(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_publication", Title: "Get Publication", Description: "Get one AgencyPublication by its version number."}, mcpGetPublication(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_publications", Title: "List Publications", Description: "List every AgencyPublication for an Instance."}, mcpListPublications(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_set_publication_channel", Title: "Set Publication Channel", Description: "Point a live channel (e.g. \"production\") at an AgencyPublication, or archive one by passing an empty channel. Production reads whichever publication serves its channel, so this is both how a release goes live and how a rollback to an earlier publication is done."}, mcpSetPublicationChannel(resolve))
}

// mcpCreateAgencyParams mirrors models.Agency field-for-field, but with
// every field but Name marked omitempty: the go-sdk's schema inference
// (google/jsonschema-go) treats any field lacking omitempty/omitzero as
// required, and models.Agency's own json tags (decision #1: imported
// unmodified) carry none — embedding it directly would force a caller to
// invent values for id/created_at/etc. on every call.
type mcpCreateAgencyParams struct {
	InstanceID                   string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	Name                         string `json:"name" jsonschema:"the Agency's name"`
	Description                  string `json:"description,omitempty" jsonschema:"a short description of what this Agency is/does"`
	Enabled                      bool   `json:"enabled,omitempty" jsonschema:"whether the Agency is active"`
	DefaultFailurePipelineBudget int    `json:"default_failure_pipeline_budget,omitempty" jsonschema:"default retry budget for an on-failure pipeline"`
	InactivityTimeoutSeconds     int    `json:"inactivity_timeout_seconds,omitempty" jsonschema:"seconds of inactivity before a run is considered stalled"`
	EventFlows                   string `json:"event_flows,omitempty" jsonschema:"raw JSON event-flow blob; Workflow.EventFlows falls back to this when empty"`
}

func mcpCreateAgency(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpCreateAgencyParams) (*mcp.CallToolResult, models.Agency, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpCreateAgencyParams) (*mcp.CallToolResult, models.Agency, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Agency{}, err
		}
		out, err := m.CreateAgency(ctx, models.Agency{
			Name:                         p.Name,
			Description:                  p.Description,
			Enabled:                      p.Enabled,
			DefaultFailurePipelineBudget: p.DefaultFailurePipelineBudget,
			InactivityTimeoutSeconds:     p.InactivityTimeoutSeconds,
			EventFlows:                   p.EventFlows,
		})
		return summary("Created Agency %q (%s)", out.Name, out.ID), out, err
	}
}

func mcpGetAgency(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, instanceOnlyParams) (*mcp.CallToolResult, models.Agency, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p instanceOnlyParams) (*mcp.CallToolResult, models.Agency, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Agency{}, err
		}
		out, err := m.GetAgency(ctx)
		return summaryWithContext(fmt.Sprintf("Agency %q (%s)", out.Name, out.ID), agencyContextText(out)), out, err
	}
}

// agencyContextText is what an LLM sees for General Information opened
// outside its own tool loop (mwanachama-wakala-studio's sidebar shortcut) —
// field-by-field rather than a struct dump, so dropping a field from an
// LLM's context later is just deleting it from this Sprintf.
func agencyContextText(a models.Agency) string {
	description := a.Description
	if description == "" {
		description = "—"
	}
	return fmt.Sprintf(
		"Name: %s; Description: %s; Enabled: %t; Created: %s; Last updated: %s.",
		a.Name, description, a.Enabled, a.CreatedAt, a.LastUpdated,
	)
}

type mcpCreateDraftParams struct {
	InstanceID     string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	Description    string `json:"description" jsonschema:"what this Draft is for"`
	ForkedFromID   string `json:"forked_from_id,omitempty" jsonschema:"another Draft's id, or a publication's tag — matching forked_from_type; omit along with forked_from_type to start empty"`
	ForkedFromType string `json:"forked_from_type,omitempty" jsonschema:"omit or \"\" to start an empty Draft, \"draft\" to fork another open Draft, or \"publication\" to start from what a published tag actually contains (the way to revisit an earlier release)"`
}

func mcpCreateDraft(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpCreateDraftParams) (*mcp.CallToolResult, models.Draft, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpCreateDraftParams) (*mcp.CallToolResult, models.Draft, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Draft{}, err
		}
		out, err := m.CreateDraft(ctx, p.Description, p.ForkedFromID, p.ForkedFromType)
		return summary("Forked Draft %s: %q", out.ID, out.Description), out, err
	}
}

type mcpDraftIDParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	DraftID    string `json:"draft_id" jsonschema:"the Draft's id"`
}

func mcpGetDraft(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpDraftIDParams) (*mcp.CallToolResult, models.Draft, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpDraftIDParams) (*mcp.CallToolResult, models.Draft, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Draft{}, err
		}
		out, err := m.GetDraft(ctx, p.DraftID)
		return summary("Draft %s: %q (%s)", out.ID, out.Description, out.Status), out, err
	}
}

// mcpListDraftsParams extends instanceOnlyParams with DraftFilter's fields.
type mcpListDraftsParams struct {
	instanceOnlyParams
	Description string `json:"description,omitempty" jsonschema:"filter: case-insensitive substring match on Description"`
	Status      string `json:"status,omitempty" jsonschema:"filter: exact match on Status (\"open\", \"promoted\", or \"archived\")"`
}

func mcpListDrafts(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListDraftsParams) (*mcp.CallToolResult, ListResult[models.Draft], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListDraftsParams) (*mcp.CallToolResult, ListResult[models.Draft], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Draft]{}, err
		}
		out, err := m.ListDrafts(ctx, mwanachamaagency.DraftFilter{Description: p.Description, Status: p.Status})
		return summaryWithContext(fmt.Sprintf("Found %d draft(s)", len(out)), draftsContextText(out)), ListResult[models.Draft]{Items: out}, err
	}
}

// draftsContextText is what an LLM sees for a Drafts list — one line per
// Draft with its id and status. Unlike rolesContextText/objectivesContextText,
// this isn't just a convenience: a Draft's id is exactly what a caller
// needs next to create or edit a Draft-scoped row (RoleAssignment, Role,
// Objective, ...), and it has no other reliable way to discover it — a
// bare "Found N draft(s)" summary line left the id invisible to the model
// entirely.
func draftsContextText(drafts []models.Draft) string {
	if len(drafts) == 0 {
		return "There are no drafts yet."
	}
	lines := make([]string, 0, len(drafts))
	for _, d := range drafts {
		lines = append(lines, fmt.Sprintf("- %q (%s): draft id: %s", d.Description, d.Status, d.ID))
	}
	return "Current drafts:\n" + strings.Join(lines, "\n")
}

type mcpUpdateDraftDescriptionParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	DraftID     string `json:"draft_id" jsonschema:"the Draft's id"`
	Description string `json:"description" jsonschema:"the Draft's new description"`
}

func mcpUpdateDraftDescription(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpUpdateDraftDescriptionParams) (*mcp.CallToolResult, models.Draft, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpUpdateDraftDescriptionParams) (*mcp.CallToolResult, models.Draft, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Draft{}, err
		}
		out, err := m.UpdateDraftDescription(ctx, p.DraftID, p.Description)
		return summary("Updated Draft %s description to %q", out.ID, out.Description), out, err
	}
}

func mcpArchiveDraft(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpDraftIDParams) (*mcp.CallToolResult, models.Draft, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpDraftIDParams) (*mcp.CallToolResult, models.Draft, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Draft{}, err
		}
		out, err := m.ArchiveDraft(ctx, p.DraftID)
		return summary("Archived Draft %s: %q", out.ID, out.Description), out, err
	}
}

type mcpPublishAgencyParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	DraftID    string `json:"draft_id" jsonschema:"the Draft to publish — its current rows are frozen into the new publication"`
	Tag        string `json:"tag,omitempty" jsonschema:"a name for this publication, unique across publications; empty falls back to v<version>"`
	Channel    string `json:"channel,omitempty" jsonschema:"the live channel this publication serves (e.g. \"production\", \"staging\"); empty means production. Whichever publication held that channel is archived"`
}

func mcpPublishAgency(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpPublishAgencyParams) (*mcp.CallToolResult, models.AgencyPublication, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpPublishAgencyParams) (*mcp.CallToolResult, models.AgencyPublication, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.AgencyPublication{}, err
		}
		out, err := m.PublishAgency(ctx, p.DraftID, p.Tag, p.Channel)
		return summary("Published %s (version %d) to %s", out.Tag, out.Version, channelLabel(out.Channel)), out, err
	}
}

type mcpPublicationVersionParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	Version    int    `json:"version" jsonschema:"the AgencyPublication's version number"`
}

func mcpGetPublication(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpPublicationVersionParams) (*mcp.CallToolResult, models.AgencyPublication, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpPublicationVersionParams) (*mcp.CallToolResult, models.AgencyPublication, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.AgencyPublication{}, err
		}
		out, err := m.GetPublication(ctx, p.Version)
		return summary("%s (version %d, %s)", out.Tag, out.Version, channelLabel(out.Channel)), out, err
	}
}

// mcpListPublicationsParams extends instanceOnlyParams with
// PublicationFilter's fields.
type mcpListPublicationsParams struct {
	instanceOnlyParams
	Tag     string `json:"tag,omitempty" jsonschema:"filter: case-insensitive substring match on Tag"`
	Channel string `json:"channel,omitempty" jsonschema:"filter: exact match on the live channel name (e.g. \"production\"); omit to list every publication including archived ones"`
}

func mcpListPublications(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListPublicationsParams) (*mcp.CallToolResult, ListResult[models.AgencyPublication], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListPublicationsParams) (*mcp.CallToolResult, ListResult[models.AgencyPublication], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.AgencyPublication]{}, err
		}
		out, err := m.ListPublications(ctx, mwanachamaagency.PublicationFilter{Tag: p.Tag, Channel: p.Channel})
		return summaryWithContext(fmt.Sprintf("Found %d publication(s)", len(out)), publicationsContextText(out)), ListResult[models.AgencyPublication]{Items: out}, err
	}
}

// publicationsContextText is what an LLM sees for a Publications list —
// one line per publication. Version, not ID, is what agency_get_publication/
// agency_set_publication_channel actually take, so it leads each line.
func publicationsContextText(publications []models.AgencyPublication) string {
	if len(publications) == 0 {
		return "There are no publications yet."
	}
	lines := make([]string, 0, len(publications))
	for _, pub := range publications {
		lines = append(lines, fmt.Sprintf("- %s: version %d, %s", pub.Tag, pub.Version, channelLabel(pub.Channel)))
	}
	return "Current publications:\n" + strings.Join(lines, "\n")
}

// channelLabel renders a publication's channel for a human reading a tool
// log, where "" means archived rather than an empty name.
func channelLabel(channel string) string {
	if channel == models.ChannelArchived {
		return "archived"
	}
	return channel
}

type mcpSetPublicationChannelParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	Version    int    `json:"version" jsonschema:"the AgencyPublication's version number"`
	Channel    string `json:"channel" jsonschema:"the live channel this publication should serve (e.g. \"production\"), or empty to archive it. Setting a channel takes it off whichever publication held it — that is how a rollback to an older publication works"`
}

func mcpSetPublicationChannel(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpSetPublicationChannelParams) (*mcp.CallToolResult, models.AgencyPublication, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpSetPublicationChannelParams) (*mcp.CallToolResult, models.AgencyPublication, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.AgencyPublication{}, err
		}
		out, err := m.SetPublicationChannel(ctx, p.Version, p.Channel)
		return summary("%s is now %s", out.Tag, channelLabel(out.Channel)), out, err
	}
}
