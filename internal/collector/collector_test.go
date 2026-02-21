package collector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/pulumi-labs/pulumi-exporter/internal/client"
	"github.com/pulumi-labs/pulumi-exporter/internal/config"
)

type mockAPI struct {
	stacks      *client.ListStacksResponse
	resources   map[string]*client.ResourceCountResponse
	updates     map[string]*client.ListUpdatesResponse
	deployments map[string]*client.ListDeploymentsResponse
}

func (m *mockAPI) ListStacks(_ context.Context) (*client.ListStacksResponse, error) {
	return m.stacks, nil
}

func (m *mockAPI) GetResourceCount(_ context.Context, org, project, stack string) (*client.ResourceCountResponse, error) {
	key := org + "/" + project + "/" + stack
	return m.resources[key], nil
}

func (m *mockAPI) ListUpdates(_ context.Context, org, project, stack string, _, _ int) (*client.ListUpdatesResponse, error) {
	key := org + "/" + project + "/" + stack
	return m.updates[key], nil
}

func (m *mockAPI) ListOrgDeployments(_ context.Context, org string) (*client.ListDeploymentsResponse, error) {
	return m.deployments[org], nil
}

func (m *mockAPI) ListMembers(_ context.Context, _ string) (*client.ListMembersResponse, error) {
	return &client.ListMembersResponse{}, nil
}

func (m *mockAPI) ListTeams(_ context.Context, _ string) (*client.ListTeamsResponse, error) {
	return &client.ListTeamsResponse{}, nil
}

func (m *mockAPI) ListEnvironments(_ context.Context, _ string) (*client.ListEnvironmentsResponse, error) {
	return &client.ListEnvironmentsResponse{}, nil
}

func (m *mockAPI) ListPolicyGroups(_ context.Context, _ string) (*client.ListPolicyGroupsResponse, error) {
	return &client.ListPolicyGroupsResponse{}, nil
}

func (m *mockAPI) ListPolicyPacks(_ context.Context, _ string) (*client.ListPolicyPacksResponse, error) {
	return &client.ListPolicyPacksResponse{}, nil
}

func (m *mockAPI) ListPolicyViolations(_ context.Context, _ string) (*client.ListPolicyViolationsResponse, error) {
	return &client.ListPolicyViolationsResponse{}, nil
}

func (m *mockAPI) ListNeoTasks(_ context.Context, _ string) (*client.ListNeoTasksResponse, error) {
	return &client.ListNeoTasksResponse{}, nil
}

func (m *mockAPI) GetPolicyResultsMetadata(_ context.Context, _ string) (*client.PolicyResultsMetadataResponse, error) {
	return &client.PolicyResultsMetadataResponse{}, nil
}

func newTestCollector(t *testing.T, api PulumiAPI) (*Collector, *sdkmetric.ManualReader) {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	cfg := &config.Config{
		Pulumi: config.PulumiConfig{
			Organizations:   []string{"test-org"},
			CollectInterval: 10 * time.Second,
			MaxConcurrency:  10,
		},
	}

	c, err := NewCollector(api, cfg, meter, slog.Default())
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	return c, reader
}

func TestCollectStack(t *testing.T) {
	t.Parallel()

	api := &mockAPI{
		resources: map[string]*client.ResourceCountResponse{
			"test-org/my-project/dev": {Count: 42, Version: 1},
		},
		updates: map[string]*client.ListUpdatesResponse{
			"test-org/my-project/dev": {
				Updates: []client.UpdateInfo{
					{
						Kind:      "update",
						Result:    "succeeded",
						StartTime: 1000,
						EndTime:   1060,
						Version:   1,
						ResourceChanges: map[string]int{
							"create": 5,
							"update": 2,
						},
					},
				},
			},
		},
	}

	c, reader := newTestCollector(t, api)
	ctx := context.Background()

	stack := client.StackSummary{
		OrgName:     "test-org",
		ProjectName: "my-project",
		StackName:   "dev",
	}

	c.collectStack(ctx, stack)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	// Verify we got metrics.
	if len(rm.ScopeMetrics) == 0 {
		t.Fatal("expected scope metrics")
	}

	metrics := rm.ScopeMetrics[0].Metrics
	if len(metrics) == 0 {
		t.Fatal("expected metrics")
	}

	// Check that expected metric names are present.
	found := make(map[string]bool)
	for _, m := range metrics {
		found[m.Name] = true
	}

	expectedMetrics := []string{
		"pulumi_stack_resource_count",
		"pulumi_update_duration_seconds",
		"pulumi_update_total",
		"pulumi_update_resource_changes",
		"pulumi_stack_last_update_timestamp",
	}
	for _, name := range expectedMetrics {
		if !found[name] {
			t.Errorf("expected metric %q not found", name)
		}
	}
}

func TestCollectOrgDeployments(t *testing.T) {
	t.Parallel()

	api := &mockAPI{
		deployments: map[string]*client.ListDeploymentsResponse{
			"test-org": {
				Deployments: []client.DeploymentInfo{
					{ID: "1", Status: "running", Created: "2024-01-01T00:00:00Z"},
					{ID: "2", Status: "running", Created: "2024-01-01T00:00:00Z"},
					{ID: "3", Status: "succeeded", Created: "2024-01-01T00:00:00Z"},
				},
			},
		},
	}

	c, reader := newTestCollector(t, api)
	ctx := context.Background()

	c.collectOrgDeployments(ctx, "test-org")

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	if len(rm.ScopeMetrics) == 0 {
		t.Fatal("expected scope metrics")
	}

	found := false
	for _, m := range rm.ScopeMetrics[0].Metrics {
		if m.Name == "pulumi_deployment_status" {
			found = true
		}
	}

	if !found {
		t.Error("expected pulumi_deployment_status metric")
	}
}

func TestLastSeenVersionTracking(t *testing.T) {
	t.Parallel()

	api := &mockAPI{
		resources: map[string]*client.ResourceCountResponse{
			"test-org/my-project/dev": {Count: 10, Version: 1},
		},
		updates: map[string]*client.ListUpdatesResponse{
			"test-org/my-project/dev": {
				Updates: []client.UpdateInfo{
					{Kind: "update", Result: "succeeded", StartTime: 1000, EndTime: 1060, Version: 1},
					{Kind: "update", Result: "succeeded", StartTime: 2000, EndTime: 2120, Version: 2},
				},
			},
		},
	}

	c, reader := newTestCollector(t, api)
	ctx := context.Background()

	stack := client.StackSummary{
		OrgName:     "test-org",
		ProjectName: "my-project",
		StackName:   "dev",
	}

	// First collection should process both updates.
	c.collectStack(ctx, stack)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect: %v", err)
	}

	// Verify lastSeenVersion was updated.
	key := "test-org/my-project/dev"
	if c.lastSeenVersion[key] != 2 {
		t.Errorf("expected lastSeenVersion=2, got %d", c.lastSeenVersion[key])
	}

	// Second collection with same updates should not increment counters.
	c.collectStack(ctx, stack)

	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect: %v", err)
	}

	// lastSeenVersion should still be 2.
	if c.lastSeenVersion[key] != 2 {
		t.Errorf("expected lastSeenVersion=2 after second collect, got %d", c.lastSeenVersion[key])
	}
}

func TestCollectConcurrency(t *testing.T) {
	t.Parallel()

	stacks := make([]client.StackSummary, 20)
	resources := make(map[string]*client.ResourceCountResponse)
	updates := make(map[string]*client.ListUpdatesResponse)

	for i := range 20 {
		name := "stack-" + string(rune('a'+i))
		stacks[i] = client.StackSummary{
			OrgName:     "test-org",
			ProjectName: "project",
			StackName:   name,
		}
		key := "test-org/project/" + name
		resources[key] = &client.ResourceCountResponse{Count: i}
		updates[key] = &client.ListUpdatesResponse{}
	}

	api := &mockAPI{
		stacks:      &client.ListStacksResponse{Stacks: stacks},
		resources:   resources,
		updates:     updates,
		deployments: map[string]*client.ListDeploymentsResponse{"test-org": {Deployments: nil}},
	}

	c, _ := newTestCollector(t, api)
	ctx := context.Background()

	// This should not deadlock with the semaphore.
	c.collect(ctx)
}

func TestCollectTimeout(t *testing.T) {
	t.Parallel()

	// Use a slow mock that blocks until context is cancelled.
	api := &slowMockAPI{delay: 5 * time.Second}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	cfg := &config.Config{
		Pulumi: config.PulumiConfig{
			Organizations:   []string{"test-org"},
			CollectInterval: 1 * time.Second, // Short interval -> timeout clamps to 10s minimum
			MaxConcurrency:  5,
		},
	}

	c, err := NewCollector(api, cfg, meter, slog.Default())
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	// Cancel the parent context quickly; collect should respect it.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// collect should return without hanging thanks to context cancellation.
	done := make(chan struct{})
	go func() {
		c.collect(ctx)
		close(done)
	}()

	select {
	case <-done:
		// OK - collect returned
	case <-time.After(5 * time.Second):
		t.Fatal("collect did not respect context cancellation")
	}
}

// slowMockAPI is a mock that blocks on ListStacks until the context is cancelled.
type slowMockAPI struct {
	delay time.Duration
}

func (m *slowMockAPI) ListStacks(ctx context.Context) (*client.ListStacksResponse, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(m.delay):
		return &client.ListStacksResponse{}, nil
	}
}

func (m *slowMockAPI) GetResourceCount(_ context.Context, _, _, _ string) (*client.ResourceCountResponse, error) {
	return &client.ResourceCountResponse{}, nil
}

func (m *slowMockAPI) ListUpdates(_ context.Context, _, _, _ string, _, _ int) (*client.ListUpdatesResponse, error) {
	return &client.ListUpdatesResponse{}, nil
}

func (m *slowMockAPI) ListOrgDeployments(_ context.Context, _ string) (*client.ListDeploymentsResponse, error) {
	return &client.ListDeploymentsResponse{}, nil
}

func (m *slowMockAPI) ListMembers(_ context.Context, _ string) (*client.ListMembersResponse, error) {
	return &client.ListMembersResponse{}, nil
}

func (m *slowMockAPI) ListTeams(_ context.Context, _ string) (*client.ListTeamsResponse, error) {
	return &client.ListTeamsResponse{}, nil
}

func (m *slowMockAPI) ListEnvironments(_ context.Context, _ string) (*client.ListEnvironmentsResponse, error) {
	return &client.ListEnvironmentsResponse{}, nil
}

func (m *slowMockAPI) ListPolicyGroups(_ context.Context, _ string) (*client.ListPolicyGroupsResponse, error) {
	return &client.ListPolicyGroupsResponse{}, nil
}

func (m *slowMockAPI) ListPolicyPacks(_ context.Context, _ string) (*client.ListPolicyPacksResponse, error) {
	return &client.ListPolicyPacksResponse{}, nil
}

func (m *slowMockAPI) ListPolicyViolations(_ context.Context, _ string) (*client.ListPolicyViolationsResponse, error) {
	return &client.ListPolicyViolationsResponse{}, nil
}

func (m *slowMockAPI) ListNeoTasks(_ context.Context, _ string) (*client.ListNeoTasksResponse, error) {
	return &client.ListNeoTasksResponse{}, nil
}

func (m *slowMockAPI) GetPolicyResultsMetadata(_ context.Context, _ string) (*client.PolicyResultsMetadataResponse, error) {
	return &client.PolicyResultsMetadataResponse{}, nil
}
