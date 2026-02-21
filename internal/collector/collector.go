// Package collector implements periodic metrics collection from Pulumi Cloud.
package collector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/errgroup"

	"github.com/pulumi-labs/pulumi-exporter/internal/client"
	"github.com/pulumi-labs/pulumi-exporter/internal/config"
)

// PulumiAPI defines the interface for interacting with the Pulumi Cloud API.
type PulumiAPI interface {
	ListStacks(ctx context.Context) (*client.ListStacksResponse, error)
	GetResourceCount(ctx context.Context, org, project, stack string) (*client.ResourceCountResponse, error)
	ListUpdates(ctx context.Context, org, project, stack string, page, pageSize int) (*client.ListUpdatesResponse, error)
	ListOrgDeployments(ctx context.Context, org string) (*client.ListDeploymentsResponse, error)
	ListMembers(ctx context.Context, org string) (*client.ListMembersResponse, error)
	ListTeams(ctx context.Context, org string) (*client.ListTeamsResponse, error)
	ListEnvironments(ctx context.Context, org string) (*client.ListEnvironmentsResponse, error)
	ListPolicyGroups(ctx context.Context, org string) (*client.ListPolicyGroupsResponse, error)
	ListPolicyPacks(ctx context.Context, org string) (*client.ListPolicyPacksResponse, error)
	ListPolicyViolations(ctx context.Context, org string) (*client.ListPolicyViolationsResponse, error)
	ListNeoTasks(ctx context.Context, org string) (*client.ListNeoTasksResponse, error)
	GetPolicyResultsMetadata(ctx context.Context, org string) (*client.PolicyResultsMetadataResponse, error)
}

// Collector periodically collects metrics from the Pulumi Cloud API.
type Collector struct {
	client          PulumiAPI
	cfg             *config.Config
	logger          *slog.Logger
	mu              sync.Mutex
	lastSeenVersion map[string]int
	instruments     *Instruments
}

// NewCollector creates a new Collector.
func NewCollector(apiClient PulumiAPI, cfg *config.Config, meter metric.Meter, logger *slog.Logger) (*Collector, error) {
	instruments, err := NewInstruments(meter)
	if err != nil {
		return nil, err
	}

	return &Collector{
		client:          apiClient,
		cfg:             cfg,
		logger:          logger,
		lastSeenVersion: make(map[string]int),
		instruments:     instruments,
	}, nil
}

// Run starts the collection loop, polling at the configured interval.
func (c *Collector) Run(ctx context.Context) error {
	c.logger.Info("starting collector", "interval", c.cfg.Pulumi.CollectInterval)

	// Collect immediately on start.
	c.collect(ctx)

	ticker := time.NewTicker(c.cfg.Pulumi.CollectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("collector stopped")
			return ctx.Err()
		case <-ticker.C:
			c.collect(ctx)
		}
	}
}

func (c *Collector) collect(ctx context.Context) {
	c.logger.Info("collecting metrics")

	// Apply a collection timeout: 90% of the collect interval, clamped to a 10s minimum.
	timeout := c.cfg.Pulumi.CollectInterval * 9 / 10
	if timeout < 10*time.Second {
		timeout = 10 * time.Second
	}
	collectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stacks, err := c.client.ListStacks(collectCtx)
	if err != nil {
		c.logger.Error("failed to list stacks", "error", err)
		return
	}

	// Build a set of configured organizations for filtering.
	orgSet := make(map[string]struct{}, len(c.cfg.Pulumi.Organizations))
	for _, org := range c.cfg.Pulumi.Organizations {
		orgSet[org] = struct{}{}
	}

	// Fan out stack collection with a semaphore.
	sem := make(chan struct{}, c.cfg.Pulumi.MaxConcurrency)
	var wg sync.WaitGroup

	for _, stack := range stacks.Stacks {
		if _, ok := orgSet[stack.OrgName]; !ok {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(s client.StackSummary) {
			defer wg.Done()
			defer func() { <-sem }()

			c.collectStack(collectCtx, s)
		}(stack)
	}

	wg.Wait()

	// Collect org-level metrics with bounded parallelism.
	g, gCtx := errgroup.WithContext(collectCtx)
	g.SetLimit(3)
	for _, org := range c.cfg.Pulumi.Organizations {
		g.Go(func() error {
			c.collectOrgDeployments(gCtx, org)
			c.collectOrgMetrics(gCtx, org)
			return nil
		})
	}
	_ = g.Wait()

	c.logger.Info("collection complete")
}
