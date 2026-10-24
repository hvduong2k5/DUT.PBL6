# 01 — Domain và bất biến, baseline 3.2

[Chỉ mục](README.md) · [HLD](00_order_service_hld.md) · [Tiếp](02_state_machine_and_lifecycle.md)

## 1. Aggregate và boundary

**Order Core addendum 17/10/2026:** [chương12](12_order_core_design.md) chốt root/control/workflow boundaries và OC01–OC15. [OrderSnapshot v1 schema](../../../03_api_specs/order-snapshot.v1.schema.json) định nghĩa snapshot chính thức, khác Quote DTO; shape validation và arithmetic/proof validation là hai lớp riêng.

| Đối tượng | Vai trò |
| --- | --- |
| CheckoutOperation | Acceptance, canonicalization/acquisition/compensation; chưa phải Order |
| Order + items + snapshot | Aggregate thương mại; tối đa một Order/operation |
| PaymentReceipt | Sự thật tiền verified, có thể chưa match Order |
| Allocation | Phân bổ receipt; MVP một receipt exact/một Order |
| Refund | Nghĩa vụ có approval/key/budget riêng |
| Job / saga step | Điều phối/recovery, độc lập Order status |
| ResourceOutcome | Projection Inventory/Promotion; không phải ledger do Order sở hữu |

Addendum [Cart/Checkout](11_cart_checkout_design.md) bổ sung cart incarnation, immutable selection, quote consumption và submission guard. Cohort VIETQR mục tiêu mở rộng invariant SUCCEEDED: Order/items/snapshot + PaymentIntent + history/outbox + cleanup intent cùng transaction PLACE; cleanup effect ở Redis diễn ra sau và không quyết định Order có hợp lệ hay không.

planned_order_id sinh trước side effect; public order_id chỉ có sau placement. order_code không phải ownership proof. customer_id là Profile ID, khác JWT sub; Guest customer_id null và scope session đã xác thực. Payment trong MS-04; Care quyết định refund theo quyền, Finance hạch toán; một refund chỉ một executor. FEFO thuộc Inventory.

## 2. Snapshot và money

Snapshot giữ SKU/name/quantity/unit price/line total/weight/packaging/version, địa chỉ đã xác minh, shipping quote và policy. Không sửa theo Catalog/Profile sau placement. Address-change command cần reprice/fulfillment guard, ngoài MVP.

Bất biến là nội dung nghiệp vụ: re-encryption có audit được phép khi logical_hash giữ nguyên; owner-at-placement bất biến, current owner chỉ thay qua verified claim. Amendment tương lai là record riêng, không ghi đè snapshot gốc. Xem chương12 mục3 để chốt version/nullability/source precedence và các field payment terms VIETQR/COD.

```text
subtotal = Σ(quantity × canonical_unit_price)
0 <= merchandise_discount <= subtotal
0 <= shipping_discount <= shipping_fee
final = subtotal - merchandise_discount + shipping_fee - shipping_discount
refundable(receipt) = received - succeeded_refunds - unresolved_refund_obligations
```

VND int64/BIGINT, nanos=0; kiểm overflow. MVP amount <=10^12 VND, tối đa 100 SKU, quantity/SKU 1..999, final>0. Zero-payment tạm reject ZERO_AMOUNT_ORDER_UNSUPPORTED. Free shipping giảm shipping component; bỏ luật final>=shipping_fee cũ.

Quote bind principal/cart revision/method/address; không reserve. Canonical price/metadata/weight phải coherent cùng snapshot/version hoặc provider revalidate atomic. Chỉ parallel independent reads; shipping phụ thuộc weight/address, voucher phụ thuộc totals.

## 3. Invariants

1. Giữ mọi verified receipt, kể cả late/unmatched/cancelled.
2. Durable intent + stable key trước side effect; timeout là caller UNKNOWN.
3. Operation SUCCEEDED iff Order/items/history/outbox/result đã commit.
4. PAID lần đầu cần allocated exact + verified receipt + stock COMMITTED + voucher finalized nếu bật.
5. Ready cần PAID hoặc COD eligible/CONFIRMED_COD, resources hợp lệ, không hold/cancel và barrier generation còn cho phép.
6. Release dành hold chưa commit; reverse là command riêng. Return đã dispatch cần inspection riêng.
7. Replay không tạo effect tài chính thứ hai; altered payload/key → conflict.
8. Cancel không tự hồi sinh bởi receipt/event trễ; refund không tự PROCESSING.
9. Owner chỉ đổi qua verified claim được phép.
10. Unknown/manual-review obligation không bị xóa khi hết retry/cache TTL.

## 4. Stock model

Adapter normalize Inventory PENDING → Order RESERVED. SUCCESS chỉ là result code.

```text
physical=100, reserved=0, available=100
reserve 2 → 100 / 2 / 98
commit 2  → 98 / 0 / 98
release trước commit → 100 / 0 / 100
```

Không thêm committed_quantity vào công thức nếu chưa có ledger đó. MVP một kho: unsupported warehouse phải reject. Reversal trace batch allocation gốc; không cộng khả dụng batch hết hạn/cách ly.

## 5. Privacy, ownership và mở rộng

Recipient/phone/street/tọa độ/free text mã hóa, phân quyền; không fan-out Kafka. shipping_address_id là Order snapshot reference, không phải địa chỉ sống Profile. Identifier cũng cần ACL/retention.

Customer/Guest chỉ đọc/command đúng scope. Service đọc snapshot có identity/purpose/task; staff có chức năng và warehouse/assignment scope. Không arbitrary PATCH status. Claim/OTP, deletion retention, B2B tax/credit, stacking và multi-address cần policy/contract riêng.
