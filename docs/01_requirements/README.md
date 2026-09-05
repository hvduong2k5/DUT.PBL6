# Hướng Dẫn Cấu Trúc Tài Liệu Requirements

Chào mừng bạn đến với thư mục tài liệu Requirements của dự án Hệ sinh thái Mè Xửng O Mạ. Để tránh tình trạng một file tài liệu bị phình to gây khó khăn cho việc tra cứu và bảo trì, hệ thống tài liệu đã được chuẩn hóa và chia tách theo chuẩn Agile chuyên nghiệp.

Dưới đây là sơ đồ và logic tổ chức của thư mục này để bạn dễ dàng nắm bắt:

## Cấu trúc thư mục

```text
docs/01_requirements/
│
├── README.md                          <-- (Bạn đang xem file này) Hướng dẫn đọc tài liệu.
│
├── 01_Product_Backlog.md              <-- XƯƠNG SỐNG: Định nghĩa Actor, Business Rules, Epic Map và tóm tắt toàn bộ User Stories kèm Mức độ ưu tiên (Priority).
├── 02_Sprint_Planning.md              <-- KẾ HOẠCH: Phân kỳ phát triển (MVP), Bản đồ User Story, Lộ trình Sprint 12 tuần, Định nghĩa hoàn thành (DoD) và Quản trị rủi ro.
├── 03_Non_Functional_Requirements.md  <-- PHI CHỨC NĂNG: Yêu cầu về Security, Performance, Audit cho team Architecture/DevOps.
├── 04_Functional_Requirements.md      <-- CHỨC NĂNG: Bảng ánh xạ tổng hợp các Yêu cầu Chức năng (FR) tới các Epics.
├── use_case_specification.md          <-- USE CASES: Đặc tả chi tiết các Use Case, Actor và ánh xạ tới User Story.
│
└── epics/                             <-- CHI TIẾT TÍNH NĂNG: Chứa file riêng cho từng Epic.
    ├── EPIC_01_Authentication.md      <-- Chứa Acceptance Criteria (Gherkin), Functional Reqs chi tiết.
    ├── EPIC_02_Customer_Profile.md
    └── ...
```

## Logic tổ chức (Cách đọc tài liệu)

1. **Người mới (Onboarding) / Quản lý / BA:**
   - Đọc `01_Product_Backlog.md` đầu tiên để nắm các Nhóm người dùng (Actor), các Quy tắc lõi (Business Rules) và Bản đồ tính năng (Epic Map). Đây là tài liệu cốt lõi định hình scope dự án.
   - Đọc tiếp `02_Sprint_Planning.md` để biết tính năng nào được ưu tiên làm trước (MVP v1.x) và phân bổ vào Sprint nào.
   - Để hiểu sâu hơn về luồng tương tác của người dùng với hệ thống, có thể tra cứu `use_case_specification.md`.

2. **Team Developer & QA:**
   - Khi được giao làm tính năng nào (Ví dụ: Giỏ hàng), hãy vào thư mục `epics/` và mở file tương ứng (VD: `EPIC_05_Shopping_Cart.md`).
   - Trong file đó sẽ chứa chi tiết các kịch bản nghiệm thu (Acceptance Criteria), Validation Rules để lập trình và test.
   - Hãy luôn bám sát mục Định nghĩa hoàn thành (DoD) trong `02_Sprint_Planning.md` để đảm bảo không bỏ sót Unit Test, Audit Log, Phân quyền.

3. **Team System Architecture & DevOps:**
   - Tập trung vào `03_Non_Functional_Requirements.md` để chuẩn bị hạ tầng, thiết lập RBAC, Pipeline CI/CD, Caching, Database Audit Log và Monitoring.

## Nguyên tắc Traceability (Truy xuất nguồn gốc)

Khi viết thêm requirement hoặc kiểm tra tính đầy đủ, hệ thống tài liệu phải đảm bảo tính liên kết chặt chẽ:

`Business Goal` -> `Actor` -> `User Story` (Nằm ở file 01) -> `Epic` -> `Acceptance Criteria` (Nằm ở thư mục epics/) -> `Use Case` -> `Functional Requirement` -> `Technical Design` / `API`.

Mọi thay đổi lớn về luồng nghiệp vụ cốt lõi phải được cập nhật đồng bộ vào `01_Product_Backlog.md` và phản ánh lên kế hoạch trong `02_Sprint_Planning.md`.
