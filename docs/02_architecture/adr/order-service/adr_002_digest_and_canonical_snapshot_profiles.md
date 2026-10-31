# ADR-002 — Digest và canonical snapshot profiles

- Ngày: 17/10/2026
- Trạng thái: Accepted — local Core và fixtures đã cập nhật; real Inventory gate còn mở.

## Bối cảnh và gap

[Core12](../../services/order-service/12_order_core_design.md) trước coding mô tả items_digest bằng hash `{items,warehouse_id}`, còn [Inventory profile order-3.2](../../../../packages/contracts/order-3.2.md) dùng sorted array SKU/quantity, không gồm warehouse. Shared fixture có thể pass snapshot schema nhưng không khớp resource proof. Sandbox Hash(json.Marshal) cũng không là canonical profile có kiểm duplicate/Unicode/integer precision.

## Quyết định

1. Inventory wire profile là authority cho `items_digest`: UTF-8 JSON array sort SKU, keys quantity/sku_code, positive integer quantities. Warehouse luôn kiểm field riêng. Core nhận consolidated unique checkout lines với quantity1..999; adapter cho raw Reserve RPC vẫn phải consolidate/validate theo Inventory contract.
2. SelectionHash và ManifestHash khác semantic type; cùng normalized SKU/quantity có thể cho cùng hash. PayloadHash/snapshot logical hash và command request_hash không dùng thay nhau. EffectIdentity giữ key/hash do adapter pin; không sinh key từ epoch/attempt. Local CommandIdentity dùng explicit business JSON, không giả tính Inventory protobuf hash/default omission.
3. `order-json-v1`: sorted object keys, items sort SKU, exact int64 decimal, explicit null, UTF-8, minimal JSON escaping; không HTML-escape hoặc escape Unicode hợp lệ để đổi bytes. Canonical stored timestamps dùng RFC3339Nano từ typed time giữ offset; instant comparison dùng time.Equal, không so pointer/location của time.Time.
4. Unicode NFC phải được chuẩn hóa trước accepted input. Decoder/constructor reject non-NFC/invalid UTF-8 hoặc lone UTF-16 surrogate, không silently sửa commercial input đã chấp nhận. Normalize silently trong decoder không được chọn vì có thể làm lệch accepted hash.
5. Reject duplicate keys/unknown/missing fields, trailing JSON, float/exponent/-0, int64 overflow, unknown schema version. Mọi field snapshot required, kể cả field có giá0 hoặc null; không dùng Go zero values để đoán field bị thiếu.

## Thay đổi và hệ quả

Cập nhật Core12, [snapshot fixtures](../../../03_api_specs/order-snapshot.v1.examples.json) và expected logical hashes. Snapshot schema shape không đổi; chỉ sửa mô tả tham chiếu chương12. Đây là fixture synthetic, không rewrite snapshot/effect keys đang tồn tại. Integration phải pin/golden-test profile với Inventory thật; không fallback khi proof sai digest/warehouse.

Semantic validator kiểm money arithmetic, coherent Catalog version, chronology/validity, explicit owner/address/payment variants, discounts/promotion disabled. Parcel weight không nhỏ hơn tổng unit weight×quantity; cho phép packaging overhead như fixture1250g so với1200g items. Đây là consistency guard, không tự tính phí hoặc thay Shipping quote authority; provider contract fixtures phải xác nhận weight semantics trước rollout.

## Kiểm chứng

Golden hashes dùng chung docs/Go, typed decode roundtrip, item reordering, defensive copies, exact integers trên2^53, raw Unicode/HTML-sensitive/U+2028, surrogate/duplicate/missing/null negative cases. Xem [verification](../../../04_testing/order-service/04_order_core_foundations.md).
