package collector

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/pulumi-labs/pulumi-exporter/internal/client"
)

func (c *Collector) collectStack(ctx context.Context, stack client.StackSummary) {
	stackAttrs := metric.WithAttributes(
		attribute.String("org", stack.OrgName),
		attribute.String("project", stack.ProjectName),
		attribute.String("stack", stack.StackName),
	)

	// Resource count.
	rc, err := c.client.GetResourceCount(ctx, stack.OrgName, stack.ProjectName, stack.StackName)
	if err != nil {
		c.logger.Error("failed to get resource count",
			"org", stack.OrgName, "project", stack.ProjectName, "stack", stack.StackName, "error", err)
	} else {
		c.instruments.stackResourceCount.Record(ctx, int64(rc.Count), stackAttrs)
	}

	// Updates.
	updates, err := c.client.ListUpdates(ctx, stack.OrgName, stack.ProjectName, stack.StackName, 1, 100)
	if err != nil {
		c.logger.Error("failed to list updates",
			"org", stack.OrgName, "project", stack.ProjectName, "stack", stack.StackName, "error", err)
		return
	}

	stackKey := stack.OrgName + "/" + stack.ProjectName + "/" + stack.StackName

	c.mu.Lock()
	lastVersion := c.lastSeenVersion[stackKey]
	c.mu.Unlock()

	var latestEndTime int64
	var maxVersion int

	for _, update := range updates.Updates {
		// Only process updates newer than what we've seen.
		if update.Version <= lastVersion {
			continue
		}

		updateAttrs := metric.WithAttributes(
			attribute.String("org", stack.OrgName),
			attribute.String("project", stack.ProjectName),
			attribute.String("stack", stack.StackName),
			attribute.String("kind", update.Kind),
			attribute.String("result", update.Result),
		)

		// Duration.
		if update.EndTime > 0 && update.StartTime > 0 {
			duration := float64(update.EndTime - update.StartTime)
			c.instruments.updateDuration.Record(ctx, duration, updateAttrs)
		}

		// Update counter.
		c.instruments.updateTotal.Add(ctx, 1, updateAttrs)

		// Resource changes.
		for operation, count := range update.ResourceChanges {
			changeAttrs := metric.WithAttributes(
				attribute.String("org", stack.OrgName),
				attribute.String("project", stack.ProjectName),
				attribute.String("stack", stack.StackName),
				attribute.String("kind", update.Kind),
				attribute.String("operation", operation),
			)
			c.instruments.updateResourceChanges.Add(ctx, int64(count), changeAttrs)
		}

		// Track latest end time.
		if update.EndTime > latestEndTime {
			latestEndTime = update.EndTime
		}

		if update.Version > maxVersion {
			maxVersion = update.Version
		}
	}

	// Update last seen version under lock.
	if maxVersion > 0 {
		c.mu.Lock()
		if maxVersion > c.lastSeenVersion[stackKey] {
			c.lastSeenVersion[stackKey] = maxVersion
		}
		c.mu.Unlock()
	}

	// Record last update timestamp.
	if latestEndTime > 0 {
		c.instruments.stackLastUpdate.Record(ctx, float64(latestEndTime), stackAttrs)
	} else if stack.LastUpdate > 0 {
		c.instruments.stackLastUpdate.Record(ctx, float64(stack.LastUpdate), stackAttrs)
	}
}
