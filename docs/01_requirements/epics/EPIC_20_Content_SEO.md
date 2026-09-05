# EPIC 20 - Content & SEO

## 1. User Stories
*(Copy các User Story của Epic này từ file Product Backlog sang đây để chi tiết hóa)*

## 2. Functional Requirements (FR)
*(Liệt kê các yêu cầu tính năng chi tiết, các API cần thiết)*

## 3. Acceptance Criteria (AC)
*(Viết các kịch bản Gherkin Given/When/Then, Validation Rules, Edge Cases)*


<!-- MIGRATED FROM backlog_software.md -->
### US-05: Tối Ưu SEO Kỹ Thuật (SSR Next.js 14, Schema.org JSON-LD & Dynamic Sitemap)
- **Mô tả**: *Là một Khách hàng tiềm năng (ACT-01) hoặc Bot tìm kiếm (EXT-04), Tôi muốn website có đường dẫn thân thiện, tải nhanh và có cấu trúc dữ liệu Schema chuẩn, Để tôi dễ dàng tìm thấy Mè Xửng O Mạ trên kết quả tìm kiếm Google.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Render trang chi tiết sản phẩm chuẩn SEO Server-Side Rendering (SSR)
    Given Googlebot truy cập URL "/san-pham/me-xung-deo-thuong-hang-500g"
    When Next.js Server Components tiếp nhận request
    Then Server render HTML hoàn chỉnh kèm đầy đủ thẻ "<title>", "<meta name='description'>"
    And Chèn cấu trúc Schema.org JSON-LD loại "Product", "BreadcrumbList", "Organization"
    And Hình ảnh có đầy đủ thuộc tính "alt" chứa từ khóa di sản Huế
    And Điểm kiểm tra Google Lighthouse SEO đạt tối thiểu 95/100, Performance đạt > 90/100

  Scenario: Tự động cập nhật sitemap.xml khi có sản phẩm mới
    Given Quản trị viên vừa xuất bản một sản phẩm OCOP mới
    When Người dùng hoặc Bot truy cập "/sitemap.xml"
    Then Hệ thống sinh danh sách URL động kèm ngày cập nhật "lastmod" mới nhất trong vòng < 50ms
  ```
- **Kỹ thuật**: Next.js 14 App Router SSR, Schema.org JSON-LD Generator, Dynamic `sitemap.ts`, `robots.ts`.

---


## 4. UI/UX Reference
*(Link Figma hoặc mockup cho tính năng này)*