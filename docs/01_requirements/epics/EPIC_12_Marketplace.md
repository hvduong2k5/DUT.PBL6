# EPIC 12 — MARKETPLACE

## 1. Acceptance Criteria

### AC-05 — Marketplace Order
```gherkin
Given có đơn hàng mới từ Marketplace
When hệ thống nhận thông tin đơn
Then đơn được ghi nhận với mã đơn Marketplace
And không tạo đơn nội bộ trùng lặp
And đơn được đưa vào quy trình fulfillment
```

### AC-08 — Marketplace & Tồn kho chung
```gherkin
Given Sản phẩm X chỉ còn 1 cái trong kho thực
When Khách hàng A chốt đơn trên Website
Then Số lượng tồn kho hệ thống về 0
And Hệ thống lập tức gọi API sang Shopee/TikTok để set tồn kho X = 0
```
