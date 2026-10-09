# MS-04 — acceptance plan 3.1

[Implementation](../../02_architecture/services/order-service/08_implementation_and_acceptance.md) · [Evidence](03_implementation_verification.md).

## Priority và release

P0 chặn tính năng nếu sai ownership, money/stock, durable intent, duplicate effects hoặc recovery. P1 là API/contract/release behavior và bằng chứng. P2 là dataset lớn, scale/HA/PITR/tracing exporter và tối ưu. Một test refund không chặn bắt đầu v1.0 khi refund chưa bật; phải pass trước release tương ứng.

| Release | P0 | P1 | P2 |
| --- | --- | --- | --- |
| v1.0 | operation/idempotency, lost ACK, Redis loss/restart, canonical price, receipt integrity, expiry/late/unmatched statement, COD/cancel | status/Location/replay/retention, immutable snapshot, Inventory RPC, real Kafka outbox | dataset lớn/scale |
| v1.1 | ready/cancel barrier, held Order cannot start task, signed source/association, deferred/dedup | scoped queue, hold/unhold, registered Order RPC, event lifecycle | delivery workload scale |
| v1.2 | approved refund budget/UNKNOWN, paid cancel independent refund, Care completion barrier | local audit append-only/HMAC, trusted recovery | external audit anchor/PITR/HA |

## Evidence

Suite Go chạy trong real PostgreSQL/Redis/Kafka; dependencies có ledger độc lập. Mỗi harness có PostgreSQL schemas riêng; Redis DB1 không đụng Cart sandbox DB0. Postman chạy qua HTTP của binary/container thật. Report ghi command/exit code, case names, state assertions, environment và giới hạn.

ORD-T01..26 được tách thành cases v1.0/v1.1/v1.2 trong code; tests liên quan voucher/claim/POS/marketplace còn thuộc feature mở rộng. Không chuyển các phần đó thành PASS từ một mock không có implementation.

```powershell
./services/order-service/scripts/test.ps1 -StartStack
```

Baseline acceptance/completion:

```powershell
python services/order-service/scripts/measure-baseline.py
```

Script baseline cần PyYAML/jsonschema trong Python; workload 20 fresh keys, concurrency 5, một SKU. Completion client-observed có polling delay; không phải server histogram chính xác hay proof SLO production. Synthetic unpaid fixture orders được hủy qua public guarded API sau đo.
