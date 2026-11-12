# MS-11 — Kế hoạch triển khai

**Trạng thái đề xuất.** Port HTTP tham chiếu `8011`, endpoint admin qua Gateway. Mỗi role API, collector và builder có cấu hình/credential riêng. DSS worker tách quota khi mở Phase 3.

| Thành phần | Chuẩn bị |
| --- | --- |
| Kafka | Broker ACL read cho topic đã duyệt, consumer group riêng theo môi trường, retention đủ bootstrap và repair. Cấu hình topic identity và chỉ định initial offset theo nguồn. |
| PostgreSQL | Database riêng do Analytics quản lý, migration owner, quyền tối thiểu và backup/restore. `om_analytics_db` có tên trong bootstrap repo; schema control dùng `analytics_config` theo DDL đề xuất. |
| ClickHouse | Database `analytics_db`, read role cho API, write role cho builder, query limit, retention và đủ dung lượng cho ít nhất hai generations cùng staging. |
| Gateway / Identity | Route opt-in, staff audience, permission mapping và rate limit. Chặn truy cập trực tiếp từ public network. |
| Observability | Collector lag/cursor, coverage, quarantine, generation build/publish, query latency, quyền bị từ chối và dung lượng hai store. |

## Rollout

1. Chốt source/metric version, coverage, bootstrap cutoffs và ngân sách tài nguyên. Ghi version của service, schema và cấu hình khi bắt đầu.
2. Chạy migrations ở môi trường test. Kiểm query và phục hồi. Bật collector shadow với pipeline riêng và đối chiếu số mốc theo cohort/kỳ với Order.
3. Dựng ClickHouse generation và kiểm manifest theo đường API. Chạy fault tests, kill/retry và permission matrix. Mở route giới hạn cho staff đã nghiệm thu.
4. Quan sát freshness/coverage trong kỳ pilot. Từng card Finance/Inventory/Customer chỉ bật sau khi có nguồn và đối soát theo [integration gates](../02_architecture/services/analytics-service/07_integrations_and_contract_gaps.md).
5. Chỉ mở Phase 3 sau khi duyệt model, retention/consent, quyền và contract bàn giao.

## Rollback

Tắt route hoặc capability riêng. Dừng publisher con trỏ của phiên bản lỗi và ghim generation an toàn trong cùng scope nếu quyền/policy còn hiệu lực. Gắn trạng thái chất lượng theo tuổi/coverage. Giữ cursor và journal để tiếp tục ingest hoặc dựng lại, tùy phạm vi sự cố. Rollback code dùng schema tương thích hoặc kế hoạch expand/contract đã nghiệm thu. [Runbook](../06_operations/analytics_service_runbook.md)
