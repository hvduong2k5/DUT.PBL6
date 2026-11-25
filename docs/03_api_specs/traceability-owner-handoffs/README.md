# Bản Markdown gửi riêng cho từng owner

Source commit `89e15ca19432749b606c6d0a61700e98fdb7fd67`, target0.2.2-draft. Gửi đúng một file bên dưới cho từng owner; mỗi file có đề bài, link theo commit, contract nhúng và test. Không cần gửi toàn bộ lịch sử chat. Đây là snapshot phân phối, không phải spec độc lập để sửa tay.

- [Inventory — INV-TRC-01](01-inventory.md)
- [Catalog — CAT-TRC-01](02-catalog.md)
- [Identity / Keycloak — IAM-TRC-01](03-identity.md)
- [Fulfillment — FUL-TRC-01](04-fulfillment.md)
- [Order — ORD-TRC-01](05-order.md)
- [Frontend / Gateway — WEB-TRC-01](06-web-gateway.md)

Sinh lại từ repo root (Python có PyYAML): `python scripts/export-traceability-owner-handoffs.py --source-commit 89e15ca19432749b606c6d0a61700e98fdb7fd67`. Kiểm snapshot khớp source bằng thêm `--check`. Không tạo PDF/Word/ZIP.
