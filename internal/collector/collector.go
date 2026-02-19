package collector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/dirien/pulumi-exporter/internal/client"
	"github.com/dirien/pulumi-exporter/internal/config"
)

// PulumiAPI defines the interface for interacting with the Pulumi Cloud API.
type PulumiAPI interface {
	ListStacks(ctx context.Context) (*client.ListStacksResponse, error)
	GetResourceCount(ctx context.Context, org, project, stack string) (*client.ResourceCountResponse, error)
	ListUpdates(ctx context.Context, org, project, stack string, page, pageSize int) (*client.ListUpdatesResponse, error)
	ListOrgDeployments(ctx context.Context, org string) (*client.ListDeploymentsResponse, error)
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

	stacks, err := c.client.ListStacks(ctx)
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
	const maxConcurrency = 5
	sem := make(chan struct{}, maxConcurrency)
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

			c.collectStack(ctx, s)
		}(stack)
	}

	wg.Wait()

	// Collect org-level deployments.
	for _, org := range c.cfg.Pulumi.Organizations {
		c.collectOrgDeployments(ctx, org)
	}

	c.logger.Info("collection complete")
}
