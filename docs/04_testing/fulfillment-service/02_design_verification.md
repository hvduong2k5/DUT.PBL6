# Fulfillment design verification — 30/10/2026

[Design index](../../02_architecture/services/fulfillment-service/README.md) · [Acceptance plan](01_acceptance_and_verification_plan.md).

**Kết quả: PASS trong phạm vi artifact thiết kế và DDL smoke bên dưới.** Kết quả máy đọc tại [design-verification.json](design-verification.json) và [ddl-verification.json](ddl-verification.json). Runtime Fulfillment và provider acceptance vẫn **PENDING**.

## Phạm vi

Markdown links/fences; pinned source manifest; OpenAPI/examples/negative guards; packing event schema + Shipping shape constraints; cross-artifact IDs/digest/version/parcel; candidate proto; SQL candidate; Draw.io XML/edges. Không suy ra E2E, media scanner safety, concurrent DB correctness hoặc production readiness.

## Lệnh

```powershell
python docs/04_testing/fulfillment-service/verify_design.py --report docs/04_testing/fulfillment-service/design-verification.json
```

## Kết quả thực chạy

| Kiểm tra | Kết quả |
| --- | --- |
| Markdown |23 file; liên kết local và code fences hợp lệ; số link chính xác trong JSON report |
| Pinned sources |34 Git blobs khớp SHA256, không thiếu source commit;1 schema snapshot nguyên byte |
| OpenAPI |17 operations,28 schemas; schema/examples hợp lệ |
| Negative API/event fixtures |9 API payload sai và8 event sai bị từ chối đúng; shape checks bổ sung theo Shipping consumer |
| Linked examples |Order/task/package/generation/digest/allocation/version/parcel/video checksum nhất quán |
| Canonical hash |4 vectors có canonical UTF-8/SHA256 đúng; interop provider Go/Java chưa nghiệm thu |
| Accepted discovery candidate |Schema task.accepted.v1 và linked example hợp lệ; bindings khớp admission checkpoint version1; chưa là shared runtime contract |
| Candidate proto |Compile3 services,14 messages;4 dependency/proof examples parse đúng message |
| SQL parse |36 SQL statements parse thành công bằng pglast |
| Draw.io |4 pages XML hợp lệ, IDs/edge references hợp lệ; chưa visual-render QA |
| PostgreSQL disposable |PostgreSQL16.15; DDL tạo16 bảng thành công;8 constraint smoke checks PASS |
| Cleanup |Container riêng đã dừng, không mount host volume/publish port; image digest trong JSON |

DDL checks thực chạy: terminal admission rewrite denied; generation regression denied; accepted identity removal denied; duplicate task/order denied; zero required quantity denied; seal thiếu task version denied; immutable revision UPDATE denied; immutable revision DELETE denied. Fixture transaction rollback, container bị hủy sau kiểm tra. Đây không phải kiểm tranh chấp qua application/runtime chưa tồn tại.

Môi trường static: Python3.14.3, các dependency versions trong JSON report và [requirements-design.txt](requirements-design.txt). Lượt chạy dùng compiler/parser libraries trong thư mục tooling tạm riêng qua PYTHONPATH; không sửa dependency/runtime service. Có thể tái lập trong virtualenv với file requirements, rồi dùng lệnh ở trên.

## Giới hạn

Chưa có implementation Fulfillment, không chạy full matrix FUL-T01–42; chưa gọi Order/Inventory/Shipping/Keycloak/S3/scanner/carrier thật, chưa kiểm concurrency accept-cancel/service commit UNKNOWN, chưa triển khai migration vào DB dự án. Không đổi acceptance của shared MVP registry hoặc bật capability. Mermaids được rà cấu trúc trong Markdown; Draw.io mới kiểm XML/edges, không claim render hình ảnh đã hoàn hảo.
