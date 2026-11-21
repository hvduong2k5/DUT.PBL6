# Prompt bàn giao cho model triển khai

Copy đoạn dưới vào chat mới **khi bạn muốn giao triển khai**, không dùng sự tồn tại của file này làm authorization tự động.

---

Bạn triển khai Notification Service bằng Go trong repo `E:\.vscode\PBL6\DUT.PBL6` theo bộ plan đã chuẩn bị. Bắt đầu bằng đọc instructions, source audit và gate review; không tái lập kế hoạch từ toàn repository.

**Git và phạm vi bắt buộc:**

- Làm trên `codex/notification-service`, base đã pin là `a24492b63c364f93ad19d880798137e554898c3f`. Kiểm tra branch, HEAD, status và local checkpoint trước edit. Nếu branch khác, bảo toàn worktree rồi chỉ switch đúng nhánh khi an toàn; không merge branch nào.
- **KHÔNG PUSH, MERGE, PR, REBASE, RESET hoặc AMEND.** Local coherent commits được phép chỉ cho task-owned files. Không có ngoại lệ push khi checkpoint/low context.
- Chỉ sửa `services/notification-service/**`, gồm code, docs, candidate contracts, tests, Compose riêng. Các nhánh khác read-only qua `git show <pinned SHA>:<path>`.
- Không sửa root `docs`, `packages`, `infra`, `.github`, Mobile/Web hoặc service khác. Nếu một integration cần peer API/schema/gateway route mới: ghi proposal và owner gate trong service docs, hỏi scope trước khi thay đổi ngoài boundary.
- Giữ nguyên untracked `docs/MOBILE_MVP_DISCUSSION_REFERENCE.md`; không stage/copy vào commit Notification.

**Đọc theo thứ tự:**

1. `services/notification-service/AGENTS.md` và ancestor rules.
2. `services/notification-service/docs/CURRENT_STATE.md`.
3. `docs/plans/MASTER_PLAN.md`, `docs/plans/active/N0_N1_CONTRACT_FOUNDATION.md` bên trong service.
4. `docs/DECISIONS.md`, `docs/SOURCE_AUDIT.md`, `docs/PROJECT_SPEC.md`.
5. `docs/ARCHITECTURE.md`, `docs/CONTRACTS.md`, `docs/DATA_AND_DELIVERY.md`, `docs/VALIDATION.md` khi làm phần tương ứng.

**Thực hiện:**

- Classify LARGE/ARCHITECTURAL. N0 audit đã có; xác nhận source pins và giải quyết gate cần cho current slice, không đọc lại toàn repo/service docs khác.
- Trình bày ngắn các lựa chọn cần chốt: local core trước hay real provider; recipient owner contract; phạm vi nghiệm thu Email. Không tự chọn nhà cung cấp trả phí/cài toolchain nặng/đụng credentials. Nếu chưa có các owner contract, đề xuất local synthetic core và xin chốt baseline cần thiết; vẫn tiến hành phần phân tích/fixtures độc lập.
- Khi baseline current slice được chấp thuận, triển khai N0.1–N0.4 rồi N1 theo active plan; cập nhật kết quả thực. Chỉ mở N2 sau N1 DoD, với detailed phase plan mới. Các phase sau không tự mở nếu user chưa giao phạm vi đó.
- Giữ domain/usecase/ports/adapters tách biệt. Kafka/HTTP intake tạo durable request/job trước ACK. Không I/O ngoài trong SQL transaction. Dedup 3 mức, lease epoch, idempotency conflict, safe callback, explicit UNKNOWN.
- Order v1/v2/v3 là cohort khác nhau; Shipping producer CloudEvent khác normalized Order ingress. Không tạo email khi chưa biết authorized recipient/audience. Không dùng Profile email optional như consent hoặc quyền gửi cho Guest.
- Fake provider/capture, synthetic recipients, real DB/Kafka isolated cho reliability tests. Default real providers/routes disabled. Không OTP generation, invoice issuing, production send, deployments hoặc cloud changes.
- Mọi trạng thái “accepted/delivered/read” phải đúng evidence. Timeout không kết luận failed/retry-safe. Không tuyên bố full US-NOTI-01 nếu OTP/Invoice/gift scenarios còn pending; chỉ báo local/partial acceptance.
- Validate proportionally theo VALIDATION, ghi explicit NOT RUN/SKIPPED/BLOCKED và lý do. Không bỏ tests để PASS, không chạy unrelated Flutter/Web suite.
- Milestone: git status/diff, cập nhật local CURRENT_STATE + active plan (branch, HEAD, paths, checks, gates, next action), local commit đúng scope nếu có thay đổi đáng kể. Không push.

**Đầu ra mong đợi của mỗi slice:** code/contract reviewable, tests phù hợp, docs cùng thư mục service, báo cáo ngắn what changed/verified/open/next. Nếu blocker ngoài scope, dừng ở gate đó và ghi proposal chính xác, không bịa endpoint hoặc sửa service khác để tự vượt blocker.

---

Lưu ý: bản plan hiện tại không có Go implementation. Prompt này không bao gồm credentials hay production environment. Người giao có thể giới hạn rõ “chỉ N0/N1” hoặc chỉ định phase tiếp theo; model phải giữ giới hạn đó.
