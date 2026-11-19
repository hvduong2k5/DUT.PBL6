# MS-11 — Runbook vận hành

## Tín hiệu và xử lý

| Triệu chứng | Kiểm tra | Hành động |
| --- | --- | --- |
| Card stale, lag thấp | `sourceAsOf`, heartbeat/coverage, tuổi generation, trạng thái producer | Xác nhận kỳ/cohort còn dữ liệu. Giữ badge theo chất lượng và báo source owner nếu dữ liệu nguồn đứng. |
| Lag tăng | Lag theo partition, tuổi cursor, lease errors, Kafka/SQL latency | Dừng replay nền nếu tranh tài nguyên. Khôi phục DB/broker rồi resume từ SQL cursor. |
| Quarantine tăng hoặc offset gap | Reason, topic identity, epoch, log range, schema pin | Giữ tọa độ và digest. Xin fixture/correction từ owner, re-drive qua repair pipeline và đối chiếu coverage. |
| Count hoặc drill-down sai | Scope, snapshot ID, version, period, dedup, manifest, parity | Cô lập generation lỗi, chuyển về generation đã kiểm hoặc tắt card. Dựng lại từ source cutoffs đã xác nhận. |
| Query timeout hoặc lỗi ClickHouse | Query p95, memory/time limit, read replica và bảng theo generation | Giảm tải rebuild và ưu tiên query budget. Nếu không có generation hợp lệ, trả 503. |
| Finance/Customer/Inventory unavailable | Source gate, metric approval, scope và permissions | Xác minh với owner. Mở lại sau khi có evidence, đối soát và kiểm quyền. |
| Privacy hoặc quyền sai | Actor, scope, policy version, export/audit IDs | Thu hồi route/quyền/snapshot/export liên quan. Vô hiệu cache; xử lý theo quy trình incident với Identity và Data owner. |

## Dựng lại sau sự cố

1. Tạm dừng builder/promotion lỗi, ghi `snapshotId`, release, pipeline, SQL backup timestamp và cursor vector.
2. Đối chiếu topic identity/log range với cursor sau restore. Nếu dữ liệu nguồn không còn đủ, dùng export có revision hoặc ghi coverage gap cho phạm vi không thể chứng minh.
3. Dựng generation cách ly, so count/digest/card totals với owner và kiểm read visibility. Xác nhận các quarantine liên quan đã được giải quyết trước khi phục hồi trạng thái fresh.
4. Chuyển serving pointer có audit. Kiểm Dashboard và drill-down, rồi mở worker theo [rollout](../05_deployment/analytics_service_deployment.md).

Mục tiêu freshness, thời gian giữ dữ liệu và RTO/RPO cần Data Team và Platform xác nhận ở pilot. Dùng labels theo source/cohort/reason, tránh đưa order ID, customer ID và payload vào metric labels. Log vận hành dùng correlation ID và metadata đã lọc.
