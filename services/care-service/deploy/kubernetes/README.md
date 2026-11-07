# Care Service Kubernetes baseline

This directory is a staging baseline, not a production-ready environment overlay.
Before applying it, the deployment owner must provide the namespace, immutable image
reference, real dependency endpoints, secret management, network policy, and ingress
or Gateway routing.

The committed ConfigMap intentionally uses `ENVIRONMENT=staging` and
`SCANNER_MODE=noop`. Care Service rejects that scanner in production. Do not change
`ENVIRONMENT` to `production` until a production malware-scanner adapter and its
quarantine/runbook are implemented.

Required secret keys:

- `CARE_INTERNAL_TOKEN`
- `DATABASE_URL`
- `S3_ACCESS_KEY`
- `S3_SECRET_KEY`
- optional `OTEL_EXPORTER_OTLP_ENDPOINT`

Example creation from an authorized deployment shell (values must come from the
environment's secret manager, not source control):

```powershell
kubectl create secret generic care-service-secrets `
  --from-literal=CARE_INTERNAL_TOKEN="$env:CARE_INTERNAL_TOKEN" `
  --from-literal=DATABASE_URL="$env:CARE_DATABASE_URL" `
  --from-literal=S3_ACCESS_KEY="$env:CARE_S3_ACCESS_KEY" `
  --from-literal=S3_SECRET_KEY="$env:CARE_S3_SECRET_KEY"
```

Render without changing cluster state:

```powershell
kubectl kustomize services/care-service/deploy/kubernetes
```

Then create an environment overlay that replaces at least:

- `care-service:local` with a digest-pinned registry image;
- `KAFKA_BROKERS`, `S3_ENDPOINT`, and `S3_PUBLIC_ENDPOINT`;
- any environment-specific resource and autoscaling values.

Database migration is a separate deployment gate. The migration binary embeds the
versioned `up` files, records version/checksum in `care_schema_migrations`, and uses a
PostgreSQL advisory transaction lock so concurrent jobs cannot apply the schema twice.
Run the Job before the application rollout (the Job is intentionally not part of the
base Kustomization, preventing an apply-time race with application pods):

```powershell
kubectl delete job care-service-migrate --ignore-not-found
kubectl apply -f services/care-service/deploy/kubernetes/migration-job.yaml
kubectl wait --for=condition=complete job/care-service-migrate --timeout=180s
kubectl logs job/care-service-migrate
```

Use the same digest-pinned image replacement for both Deployment and Job. A checksum
mismatch for an already-applied migration fails closed; never edit an applied SQL
file—add a new version instead. The runner never executes down migrations.

An older database that already contains Care tables but has no
`care_schema_migrations` ledger also fails closed. It must be inspected and baselined
through an explicitly reviewed operational procedure; the runner never guesses that
an existing schema matches its embedded migrations. Fresh Kubernetes environments do
not require this compatibility step.
