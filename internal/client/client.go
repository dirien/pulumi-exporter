// Package client provides a typed wrapper around the generated Pulumi Cloud OpenAPI client.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dirien/pulumi-exporter/internal/pulumiapi"
)

// Client wraps the generated OpenAPI client with typed convenience methods.
type Client struct {
	gen *pulumiapi.ClientWithResponses
}

// NewClient creates a new Pulumi Cloud API client.
func NewClient(baseURL, token string) (*Client, error) {
	authProvider := func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "token "+token)
		return nil
	}

	// Ensure trailing slash for correct relative URL resolution in the generated client.
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	gen, err := pulumiapi.NewClientWithResponses(baseURL,
		pulumiapi.WithHTTPClient(httpClient),
		pulumiapi.WithRequestEditorFn(authProvider),
	)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}

	return &Client{gen: gen}, nil
}

// ListStacks returns all stacks accessible to the authenticated user, handling pagination.
func (c *Client) ListStacks(ctx context.Context) (*ListStacksResponse, error) {
	var allStacks []StackSummary
	var contToken *string

	for {
		resp, err := c.gen.ListUserStacksWithResponse(ctx, &pulumiapi.ListUserStacksParams{
			ContinuationToken: contToken,
		})
		if err != nil {
			return nil, fmt.Errorf("listing stacks: %w", err)
		}
		if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return nil, fmt.Errorf("listing stacks: unexpected status %d", resp.StatusCode())
		}

		for _, s := range resp.JSON200.Stacks {
			allStacks = append(allStacks, StackSummary{
				OrgName:       s.OrgName,
				ProjectName:   s.ProjectName,
				StackName:     s.StackName,
				LastUpdate:    derefInt64(s.LastUpdate),
				ResourceCount: int(derefInt64(s.ResourceCount)),
			})
		}

		if resp.JSON200.ContinuationToken == nil || *resp.JSON200.ContinuationToken == "" {
			break
		}
		contToken = resp.JSON200.ContinuationToken
	}

	return &ListStacksResponse{Stacks: allStacks}, nil
}

// GetResourceCount returns the resource count for a specific stack.
func (c *Client) GetResourceCount(ctx context.Context, org, project, stack string) (*ResourceCountResponse, error) {
	resp, err := c.gen.GetStackResourceCountWithResponse(ctx, org, project, stack)
	if err != nil {
		return nil, fmt.Errorf("getting resource count: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("getting resource count: unexpected status %d", resp.StatusCode())
	}

	return &ResourceCountResponse{
		Count:   int(resp.JSON200.ResourceCount),
		Version: int(resp.JSON200.Version),
	}, nil
}

// ListUpdates returns the updates for a specific stack.
// The Pulumi OpenAPI spec returns an untyped response for this endpoint,
// so we parse the raw JSON body from the generated client's response.
func (c *Client) ListUpdates(ctx context.Context, org, project, stack string, page, pageSize int) (*ListUpdatesResponse, error) {
	p := int64(page)
	ps := int64(pageSize)
	resp, err := c.gen.GetStackUpdatesWithResponse(ctx, org, project, stack, &pulumiapi.GetStackUpdatesParams{
		Page:     &p,
		PageSize: &ps,
	})
	if err != nil {
		return nil, fmt.Errorf("listing updates: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("listing updates: unexpected status %d", resp.StatusCode())
	}

	var result ListUpdatesResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("listing updates: decoding response: %w", err)
	}

	return &result, nil
}

// ListOrgDeployments returns the deployments for an organization.
func (c *Client) ListOrgDeployments(ctx context.Context, org string) (*ListDeploymentsResponse, error) {
	resp, err := c.gen.ListOrgDeploymentsWithResponse(ctx, org, nil)
	if err != nil {
		return nil, fmt.Errorf("listing org deployments: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("listing org deployments: unexpected status %d", resp.StatusCode())
	}

	deployments := make([]DeploymentInfo, 0, len(resp.JSON200.Deployments))
	for _, d := range resp.JSON200.Deployments {
		deployments = append(deployments, DeploymentInfo{
			ID:      d.Id,
			Status:  string(d.Status),
			Created: d.Created,
		})
	}

	return &ListDeploymentsResponse{Deployments: deployments}, nil
}

// ListMembers returns the members of an organization.
func (c *Client) ListMembers(ctx context.Context, org string) (*ListMembersResponse, error) {
	resp, err := c.gen.ListOrganizationMembersWithResponse(ctx, org, nil)
	if err != nil {
		return nil, fmt.Errorf("listing members: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("listing members: unexpected status %d", resp.StatusCode())
	}

	members := make([]MemberInfo, 0, len(resp.JSON200.Members))
	for _, m := range resp.JSON200.Members {
		members = append(members, MemberInfo{
			Role: string(m.Role),
			User: UserInfo{Name: m.User.Name, GitHubLogin: m.User.GithubLogin},
		})
	}

	return &ListMembersResponse{Members: members}, nil
}

// ListTeams returns the teams of an organization.
func (c *Client) ListTeams(ctx context.Context, org string) (*ListTeamsResponse, error) {
	resp, err := c.gen.ListTeamsWithResponse(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("listing teams: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("listing teams: unexpected status %d", resp.StatusCode())
	}

	teams := make([]TeamInfo, 0, len(resp.JSON200.Teams))
	for _, t := range resp.JSON200.Teams {
		teams = append(teams, TeamInfo{
			Name:        t.Name,
			DisplayName: t.DisplayName,
			Kind:        string(t.Kind),
		})
	}

	return &ListTeamsResponse{Teams: teams}, nil
}

// ListEnvironments returns the ESC environments of an organization.
func (c *Client) ListEnvironments(ctx context.Context, org string) (*ListEnvironmentsResponse, error) {
	resp, err := c.gen.ListOrgEnvironmentsEscWithResponse(ctx, org, nil)
	if err != nil {
		return nil, fmt.Errorf("listing environments: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("listing environments: unexpected status %d", resp.StatusCode())
	}

	envs := make([]EnvironmentInfo, 0, len(resp.JSON200.Environments))
	for _, e := range resp.JSON200.Environments {
		envs = append(envs, EnvironmentInfo{
			Name:         derefStr(e.Name),
			Organization: derefStr(e.Organization),
			Project:      derefStr(e.Project),
		})
	}

	return &ListEnvironmentsResponse{Environments: envs}, nil
}

// ListPolicyGroups returns the policy groups of an organization.
func (c *Client) ListPolicyGroups(ctx context.Context, org string) (*ListPolicyGroupsResponse, error) {
	resp, err := c.gen.ListPolicyGroupsWithResponse(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("listing policy groups: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("listing policy groups: unexpected status %d", resp.StatusCode())
	}

	groups := make([]PolicyGroupInfo, 0, len(resp.JSON200.PolicyGroups))
	for _, g := range resp.JSON200.PolicyGroups {
		groups = append(groups, PolicyGroupInfo{
			Name:                  g.Name,
			NumStacks:             int(g.NumStacks),
			NumEnabledPolicyPacks: int(g.NumEnabledPolicyPacks),
			IsOrgDefault:          g.IsOrgDefault,
		})
	}

	return &ListPolicyGroupsResponse{PolicyGroups: groups}, nil
}

// ListPolicyPacks returns the policy packs of an organization.
func (c *Client) ListPolicyPacks(ctx context.Context, org string) (*ListPolicyPacksResponse, error) {
	resp, err := c.gen.ListPolicyPacksOrgsWithResponse(ctx, org, nil)
	if err != nil {
		return nil, fmt.Errorf("listing policy packs: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("listing policy packs: unexpected status %d", resp.StatusCode())
	}

	packs := make([]PolicyPackInfo, 0, len(resp.JSON200.PolicyPacks))
	for _, p := range resp.JSON200.PolicyPacks {
		packs = append(packs, PolicyPackInfo{
			Name:        p.Name,
			DisplayName: p.DisplayName,
		})
	}

	return &ListPolicyPacksResponse{PolicyPacks: packs}, nil
}

// ListPolicyViolations returns the policy violations for an organization.
func (c *Client) ListPolicyViolations(ctx context.Context, org string) (*ListPolicyViolationsResponse, error) {
	resp, err := c.gen.ListPolicyViolationsV2WithResponse(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("listing policy violations: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("listing policy violations: unexpected status %d", resp.StatusCode())
	}

	violations := make([]PolicyViolation, 0, len(resp.JSON200.PolicyViolations))
	for _, v := range resp.JSON200.PolicyViolations {
		violations = append(violations, PolicyViolation{
			ID:          v.Id,
			ProjectName: v.ProjectName,
			StackName:   derefStr(v.StackName),
			PolicyPack:  v.PolicyPack,
			PolicyName:  v.PolicyName,
			Level:       v.Level,
			Kind:        string(v.Kind),
		})
	}

	return &ListPolicyViolationsResponse{PolicyViolations: violations}, nil
}

// ListNeoTasks returns all Neo AI tasks for an organization, handling pagination.
func (c *Client) ListNeoTasks(ctx context.Context, org string) (*ListNeoTasksResponse, error) {
	var allTasks []NeoTask
	var contToken *string
	pageSize := int64(100)

	for {
		resp, err := c.gen.ListTasksWithResponse(ctx, org, &pulumiapi.ListTasksParams{
			PageSize:          &pageSize,
			ContinuationToken: contToken,
		})
		if err != nil {
			return nil, fmt.Errorf("listing neo tasks: %w", err)
		}
		if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return nil, fmt.Errorf("listing neo tasks: unexpected status %d", resp.StatusCode())
		}

		for _, t := range resp.JSON200.Tasks {
			allTasks = append(allTasks, NeoTask{
				ID:     t.Id,
				Name:   t.Name,
				Status: string(t.Status),
			})
		}

		if resp.JSON200.ContinuationToken == nil || *resp.JSON200.ContinuationToken == "" {
			break
		}
		contToken = resp.JSON200.ContinuationToken
	}

	return &ListNeoTasksResponse{Tasks: allTasks}, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}
