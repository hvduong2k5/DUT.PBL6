# Care Service observability assets

- `prometheus-rules.yaml` contains portable Prometheus rule groups.
- `grafana-dashboard.json` contains a provisionable Grafana dashboard.

Deployment owners must add environment-specific labels/selectors and notification
routing. Alert expressions intentionally use only bounded labels and link to the Care
runbook. Validate the rules with the same Prometheus version used by the target
environment before release.

The dashboard expects the metrics endpoint to be scraped with the original metric
names. If the platform adds a service or namespace label, select Care Service at the
datasource or provisioning layer rather than inserting customer/ticket identifiers as
metric labels.
