# Trạng thái Gói Huế

## Công việc hiện tại

UI khách hàng đã được người dùng duyệt tại checkpoint 4e3d170. Đợt hiện tại bổ sung dữ liệu mẫu SQL và chuyển nguồn API để duyệt lần nữa. Nhánh codex/storefront-home-navigation, clone E:/.vscode/PBL6/DUT.PBL6-dev-ui-review. Commit chứa file này là checkpoint dữ liệu mẫu. Chi tiết ở docs/plans/active/STOREFRONT_SAMPLE_DATA.md và roadmap ở docs/plans/MASTER_PLAN.md.

## Đã hoàn tất

- src/data/goihue-sample.json chứa 5 sản phẩm / 4 danh mục từ goihue_backup.sql. Chỉ trích trường công khai, SQL nguồn không đổi, có hash kiểm chứng.
- API catalogue, chi tiết và recommendation đọc cùng nguồn qua STOREFRONT_DATA_MODE. real là mặc định; mock luôn dùng mẫu; auto ưu tiên API thật, fallback khi lỗi/mất endpoint hoặc toàn kho rỗng. Giữ kết quả lọc rỗng, 400/401/403 và slug không tồn tại trong kho thật.
- SKU mẫu dùng DEMO-. Ảnh minh họa lấy trong thư viện Gói Huế, bánh in giữ placeholder. Không gán chứng nhận hoặc khối lượng chưa có. Chặn SKU DEMO gửi vào giỏ thật và dữ kiện cá nhân hóa. Consent và giao dịch tiếp tục dùng API thật.
- Có script Python xuất lại fixture, lệnh preview:data chuyển nguồn và nhãn mẫu trên UI. File .env cá nhân không đổi.

## Validation và trạng thái chạy

Production build/TypeScript, ESLint và 235 test đạt. Edge đã kiểm tra trực tiếp API local chế độ auto, không intercept: danh sách, tìm kiếm, lọc, chi tiết cùng slug, gợi ý ở home/product_detail/cart và loại sản phẩm hiện tại, desktop/mobile. Ảnh ở E:/Goi-Hue-Brand-Report/mon-02/DUT-PBL6-home-review/sample-data.

Cổng 4321 chạy npm run preview:data -- auto. Trong apps/web-user, build với NEXT_PUBLIC_RECOMMENDATIONS_ENABLED=true. Dừng tiến trình để đổi sang npm run preview:data -- mock hoặc -- real, tải lại trình duyệt. Chỉ đổi nguồn không cần rebuild hoặc sửa FE. Môi trường thật dùng gateway hiện hữu và ANALYTICS_RECOMMENDATION_UPSTREAM_URL.

## Bước tiếp theo

Người dùng duyệt dữ liệu mẫu trước khi tiếp tục UI khác. Chưa xác nhận backend thật hoặc giao dịch end-to-end. Không PR/merge.
