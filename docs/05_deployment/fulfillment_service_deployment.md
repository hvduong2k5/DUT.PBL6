# Fulfillment deployment plan — candidate

> **Triển khai MVP hiện tại:** [Order–Inventory–Fulfillment–Shipping](../../packages/contracts/mvp/fulfillment-integration.md) mô tả biến môi trường thực, token-file, TLS, migrations và lệnh E2E. Các bảng bên dưới là thiết kế ngày 30/10, không thay thế cấu hình runtime này.

Ngày30/10/2026. [Design](../02_architecture/services/fulfillment-service/README.md) · [Runbook](../06_operations/fulfillment_service_runbook.md). Chưa có service image/Compose/K8s được triển khai; paths/variables dưới đây là target để người implement áp dụng, không phải lệnh chạy service hiện có.

## 1. Topology

Stateless API deployment cho REST8002/gRPC9002 (đề xuất), worker deployment dùng cùng code/config và role phù hợp; PostgreSQL private fulfillment_db; Kafka topics owner-created; private object store with immutable final prefix; scanner isolated worker. Prometheus/OTLP qua network nội bộ. Gateway route staff API, không route DB/gRPC/metrics công khai.

DB migrations one-shot service-owned job trước new application revision; app schema compatibility check, không mỗi pod tự destructive migrate. DB roles riêng runtime/migrator/readonly/recovery, least privilege. UUID app-generated, DDL candidate không cần extension. Readiness phải báo schema/config prerequisite, không liveness-fail vì Kafka/S3 outage gây restart storm.

## 2. Config inventory đề xuất

| Group / variable | Nội dung / default | Bảo quản |
| --- | --- | --- |
| FULFILLMENT_HTTP_ADDR / GRPC_ADDR | :8002 / :9002 | ConfigMap/env |
| FULFILLMENT_DATABASE_URL | Private service DB, schema fulfillment | Secret, never committed |
| FULFILLMENT_KAFKA_BROKERS | Broker list; TLS/SASL profile | Config + credentials Secret |
| FULFILLMENT_ORDER_GRPC_ENDPOINT | Order owning provider, no simulator fallback | Config with identity/audience |
| FULFILLMENT_INVENTORY_GRPC_ENDPOINT | Inventory provider | Config with method ACL |
| FULFILLMENT_ORDER_SOURCE_ENDPOINT | Agreed source adapter ingress/query | Config; exact codec/capability |
| FULFILLMENT_STORAGE_ENDPOINT / BUCKET | Private S3-compatible store and own bucket | Config; credentials secret |
| FULFILLMENT_STORAGE_FINAL_PREFIX | Verified immutable final object namespace | Config with deny-overwrite policy |
| FULFILLMENT_SCANNER_PROFILE | Tested actual scanner/policy version | Config; no permissive production stub |
| FULFILLMENT_JWT_ISSUER / AUDIENCE | Approved issuer; fulfillment-api proposal | Config; JWKS source allowlisted |
| FULFILLMENT_SERVICE_AUTH_PROFILE | TLS/service audience/method ACL policy | Secret/config, local isolated profile distinct |
| FULFILLMENT_*_ENABLED | Flags in roadmap, default false | Versioned deployment change |
| OTEL_EXPORTER_OTLP_ENDPOINT | Internal collector | Config; optional telemetry availability |

Config loader validates unsupported combinations at startup: integrated-v2 + media optional; mutation enabled without peer authority; final immutable store missing; reopen enabled without capability. Secrets from deployment mechanism, not `docs` examples. No real user/bank/carrier credentials in fixture data.

## 3. Infra provisioning and topics

Create Kafka topics before producer/consumer enable: order.events.v3 read ACL; fulfillment.events.v2 produce ACL; candidate fulfillment.task.events.v1 chỉ cho Fulfillment producer/Order source adapter sau sign-off G04; audit.events.v1 scoped produce ACL; quarantine topic proposed with restricted readers. Partition key Order ID; initial partition count capacity-plan, don't assert arbitrary value as requirement. Replication/min-in-sync policy per actual cluster; local single broker is TEST only.

Object store: staging prefix upload-only scoped sessions, verified prefix write only by trusted promotion worker, no anonymous/list/download access. Versioning/immutability and exact checksum capabilities acceptance required. Lifecycle rules must not delete attached/held evidence independently of DB retention decisions. Backup with object version manifest.

## 4. Deployment sequence

1. Fulfillment S0 peer contract review + gate owners; migrations tested isolated; backup/restore plan validated.
2. Deploy schema and read/query/recovery workers; no admission/seal yet. Verify connectivity/auth/health/config.
3. Deploy actual Order permit/packing/source/shipping snapshot and Inventory allocation/terminal profile; verify compatibility with pinned SHAs.
4. Deploy Shipping v2 consumer and Order adapter; no producer new cohort until negative contract tests pass.
5. Enable media test bucket/scanner, scoped staff auth; run D2/D3 acceptance test scope.
6. Enable limited integrated-v2 TEST admissions with exact policy snapshot; observe lag/errors/duplicate metrics.
7. Broaden only after named sign-off; production SePay/carrier acceptance belongs owning teams, not inferred from TEST demo.

## 5. Resources, shutdown và scaling

API bounded request bodies/deadlines/DB pool, scanner separate memory/CPU/file-size limits, workers max inflight/peer bulkhead. Start small and benchmark actual dataset; CPU/memory numbers set after measurement. HPA uses supported signals, no autoscale admission worker beyond DB/peer throughput blindly.

Shutdown: stop readiness/admission, stop fetching new Kafka messages/jobs, finish bounded local Tx, drain in-flight RPC within deadline, release/expire job leases safely, close producers/DB. Outbox remains durable; process exit does not mark resource failed. Rebalance uses contiguous durable inbox offsets; shutdown must not commit unpersisted message.

## 6. Release rollback

Disable new admission/cohort first. Keep original query/registry/revisions, outbox and reconciliation running for accepted tasks. Keep v2 consumers while v2 packages exist. Roll back image only if schema forward-compatible; expand migrations not undone by DROP. Restore is separate incident procedure, not normal rollback. Evidence records gate/config/image digest/schema/peer SHAs and observed milestone, never “all services ready” from health checks alone.
