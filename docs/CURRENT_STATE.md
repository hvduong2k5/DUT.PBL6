# Trạng thái UI khách hàng Gói Huế

Nhánh làm việc `codex/storefront-home-navigation`, clone riêng `E:/.vscode/PBL6/DUT.PBL6-dev-ui-review`. Bắt đầu trùng tu từ `e5b8612`; commit chứa tài liệu này là checkpoint hiện tại. Đọc `docs/plans/MASTER_PLAN.md` và `docs/plans/active/STOREFRONT_RENOVATION.md`.

Home ưu tiên sản phẩm nổi bật, điều hướng tới sản phẩm thật. Công cụ đổi hero và giả lập mobile đã gỡ. Các trang khách hàng cùng font Playfair Display / Be Vietnam Pro, form, nút và bề mặt thống nhất. Catalogue mobile có bộ lọc thu gọn; thanh bước checkout được sửa tràn ngang. Rail gợi ý ở Home, chi tiết và giỏ dùng API adapter hiện hữu; consent được đặt cạnh rail trong mục mở rộng, giữ opt-in.

Đã chạy typecheck, ESLint, production build và 230 test. Browser Edge kiểm tra với fixture gợi ý và consent, khách vãng lai, đổi category, mở menu mobile, bật/tắt consent và phục hồi sau lỗi 503. Khảo sát giao diện 1440/390/320. Dịch vụ thật recommendations hiện trả 503 nên chưa xác nhận nội dung từ backend hoặc giao dịch end-to-end.

Bản local chạy cổng 4321 với `NEXT_PUBLIC_RECOMMENDATIONS_ENABLED=true` khi build và start; flag hiện hữu giữ nguyên giá trị mặc định trong mã. Không sửa env của người dùng. Để bật ở môi trường khác, người vận hành cần đặt flag này ở cả build và runtime, cùng `ANALYTICS_RECOMMENDATION_UPSTREAM_URL` đúng dịch vụ thực tế. Không nhúng fixture vào sản phẩm.

Bước tiếp theo duyệt Home, catalogue, chi tiết, giỏ và checkout trên trình duyệt. Sau đó kiểm thử cùng dịch vụ và tài khoản thật. Không PR/merge.
