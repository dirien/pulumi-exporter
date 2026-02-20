# Grafana dashboard

The repo ships with a ready-to-use Grafana dashboard at [`dashboards/pulumi-exporter.json`](../dashboards/pulumi-exporter.json). It covers all 17 metrics across 26 panels.

## What's on it

The dashboard is organized into five rows:

**Organization overview** — stat panels for members, teams, ESC environments, and policy packs across your orgs.

**Stack resources** — a bar gauge of the top 20 stacks by resource count and a table showing how long ago each stack was last updated.

**Updates** — pie charts for updates by result (succeeded/failed) and by kind (update/preview/destroy), average duration, total update count, and a resource changes breakdown.

**Policy and compliance** — policy compliance score (percentage), resource compliance score, policies with issues count, resources with issues count, total violations, policy groups, policy packs, and a violations-by-level pie chart.

**Pulumi Neo AI tasks** — tasks by status and total task count.

The dashboard includes a multi-select Organization dropdown at the top. Pick one org, several, or all.

## Using it with Docker Compose

The Docker Compose stack auto-provisions this dashboard. No manual import needed:

```bash
make compose-up
# Grafana at http://localhost:3000 (admin / admin)
```

The dashboard loads as the Grafana home page.

## Importing into an existing Grafana

If you already have Grafana running:

1. Open Grafana and go to Dashboards > Import
2. Upload `dashboards/pulumi-exporter.json` or paste its contents
3. Select your Prometheus data source
4. Click Import

The dashboard expects a Prometheus data source named `prometheus`. If yours is named differently, update the `uid` in the JSON or select the right one during import.

## Customizing

The JSON file is a standard Grafana dashboard export. Edit it directly or modify it in the Grafana UI and re-export. If you change it locally, run `make compose-restart` to pick up the changes (Grafana polls the file every 10 seconds, but a restart is more reliable).

The dashboard version field in the JSON controls Grafana's reload behavior. Bump it if Grafana isn't picking up your changes.

## Panels reference

| Row | Panels | Metrics used |
|-----|--------|-------------|
| Organization overview | Members, Teams, ESC Environments, Policy Packs | `pulumi_org_member_count`, `team_count`, `environment_count`, `policy_pack_count` |
| Stack resources | Resources by Stack, Last Update Age | `pulumi_stack_resource_count`, `pulumi_stack_last_update_timestamp` |
| Updates | By Result, By Kind, Avg Duration, Total Updates, Resource Changes | `pulumi_update_total`, `pulumi_update_duration_seconds`, `pulumi_update_resource_changes` |
| Policy and compliance | Compliance Score, Resource Compliance, Policies with Issues, Resources with Issues, Violations, Policy Groups, Policy Packs, Violations by Level | `pulumi_org_policy_total`, `policy_with_issues`, `governed_resources_total`, `governed_resources_with_issues`, `policy_violations`, `policy_group_count`, `policy_pack_count` |
| Neo AI tasks | Tasks by Status, Total Tasks | `pulumi_org_neo_task_count` |
| Deployments | Deployment Status, Deployments Over Time | `pulumi_deployment_status` |
