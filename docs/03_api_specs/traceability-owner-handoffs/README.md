# Bản Markdown gửi riêng cho từng owner

Source commit `b5ab16e36a841ecb02204651db0d4e9d83d17476`, target0.4.0-draft. Bản này thay phiếu0.3.0/0.2.2, theo canonical Catalog/Media của Dương trong PR #175; giữ REST Web và BFF. Gửi đúng một file bên dưới cho từng owner; mỗi file có đề bài, link theo commit, contract nhúng và test. Không cần gửi toàn bộ lịch sử chat. Đây là snapshot phân phối, không phải spec độc lập để sửa tay.

- [Inventory — INV-TRC-01](01-inventory.md)
- [Catalog / Media — CAT-TRC-01](02-catalog.md)
- [Identity / Keycloak — IAM-TRC-01](03-identity.md)
- [Fulfillment — FUL-TRC-01](04-fulfillment.md)
- [Order — ORD-TRC-01](05-order.md)
- [Frontend / Gateway — WEB-TRC-01](06-web-gateway.md)

Sinh lại từ repo root (Python có PyYAML): `python scripts/export-traceability-owner-handoffs.py --source-commit b5ab16e36a841ecb02204651db0d4e9d83d17476`. Kiểm snapshot khớp source bằng thêm `--check`. Không tạo PDF/Word/ZIP.
