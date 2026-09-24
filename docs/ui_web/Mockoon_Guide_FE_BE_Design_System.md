**Hướng dẫn sử dụng Mockoon**

Mock API cho Frontend, Backend và Design System

*Version 1.0 \| Tài liệu thực hành*

**Mục tiêu:** hướng dẫn team cài đặt, tạo, chạy, chia sẻ và sử dụng
Mockoon để mô phỏng REST API trong quá trình phát triển UI trước khi
Backend API hoàn thiện.

**Quick Start**

**1.** Cài Mockoon Desktop từ https://mockoon.com/.

**2.** Tạo Local Environment, ví dụ "Project API Local".

**3.** Tạo route, ví dụ GET /api/v1/users.

**4.** Chọn response 200 và nhập JSON response.

**5.** Start Environment và lấy base URL, ví dụ http://localhost:3000.

**6.** Cho Frontend gọi API qua API client như đang gọi API thật.

**7.** Thêm các scenario cần test: empty, validation error, 401, 403,
404, 409, 500, slow.

**8.** Export/commit Mockoon data file vào repository để cả team dùng
cùng cấu hình.

# 1. Mockoon được dùng để làm gì?

Mockoon tạo một HTTP API giả lập chạy local hoặc trong môi trường tự
động. Frontend có thể phát triển UI, validation, permission và error
handling mà không phải chờ API thật.

Frontend\
↓\
API Client\
↓\
Mockoon\
↓\
Mock Response

Khi Backend API sẵn sàng, Frontend chỉ cần chuyển nguồn API từ Mockoon
sang API Gateway; UI/business flow không nên phụ thuộc vào việc đang
dùng mock hay API thật.

# 2. Cài đặt

Tải Mockoon Desktop tại:

https://mockoon.com/

Mockoon cũng có CLI để chạy mock trong CI/CD hoặc Docker. Tài liệu CLI
chính thức: https://mockoon.com/cli/

# 3. Tạo Environment

**1.** Mở Mockoon và tạo New Environment.

**2.** Đặt tên theo project, ví dụ "Project API Local".

**3.** Chọn port local, ví dụ 3000.

**4.** Start Environment.

Base URL\
http://localhost:3000

# 4. Tạo GET API

Ví dụ cần API lấy danh sách user:

GET /api/v1/users

**1.** Chọn Environment.

**2.** Add Route.

**3.** Chọn method GET.

**4.** Nhập path /api/v1/users.

**5.** Chọn response status 200.

**6.** Nhập response body.

{\
\"items\": \[\
{\
\"id\": \"USR-001\",\
\"name\": \"John Doe\",\
\"email\": \"john@example.com\",\
\"status\": \"ACTIVE\"\
}\
\],\
\"page\": 1,\
\"pageSize\": 20,\
\"total\": 1\
}

# 5. Test API

Sau khi Start Environment, có thể test bằng browser, Postman, curl hoặc
Frontend.

curl http://localhost:3000/api/v1/users

# 6. Kết nối Frontend

Frontend nên gọi Mockoon thông qua API client/HTTP layer, không gọi
Mockoon trực tiếp từ từng component.

Component\
↓\
Feature / Hook\
↓\
API Client\
↓\
Mockoon

Ví dụ Axios:

const api = axios.create({\
baseURL: \"http://localhost:3000\"\
});

Khi chuyển sang API Gateway, thay baseURL thay vì sửa toàn bộ component.

# 7. Path Parameter

GET /api/v1/users/:id

Có thể test:

GET /api/v1/users/USR-001\
GET /api/v1/users/USR-999

# 8. POST / PUT / DELETE

Ví dụ tạo user:

POST /api/v1/users\
\
{\
\"name\": \"John Doe\",\
\"email\": \"john@example.com\"\
}

Với CRUD, Mockoon hỗ trợ CRUD routes gắn với Data Bucket; Data Bucket
giữ trạng thái giữa các request và phù hợp để mô phỏng một database nhỏ.
Xem: https://mockoon.com/docs/latest/api-endpoints/crud-routes/ và
https://mockoon.com/docs/latest/data-buckets/using-data-buckets/

# 9. Data Bucket

Dùng Data Bucket khi nhiều route cần dùng chung dữ liệu hoặc cần state
thay đổi sau POST/PUT/DELETE.

Data Bucket: users\
\
\[\
{\
\"id\": \"USR-001\",\
\"name\": \"John Doe\",\
\"status\": \"ACTIVE\"\
},\
{\
\"id\": \"USR-002\",\
\"name\": \"Jane Doe\",\
\"status\": \"INACTIVE\"\
}\
\]

Data Bucket có thể được tham chiếu trong response bằng các data helpers
và được dùng với CRUD routes. Xem:
https://mockoon.com/docs/latest/data-buckets/using-data-buckets/

# 10. Query Parameters

GET /api/v1/users?page=1&pageSize=20&keyword=john&status=ACTIVE

Dùng để phát triển và kiểm tra search, filter, sorting và pagination.

# 11. Mock Pagination và Empty State

{\
\"items\": \[\],\
\"page\": 1,\
\"pageSize\": 20,\
\"total\": 0\
}

Hãy test ít nhất các trường hợp total = 0, 1, đúng bằng pageSize và lớn
hơn pageSize.

# 12. Mock Validation Error

HTTP 400\
\
{\
\"code\": \"VALIDATION_ERROR\",\
\"message\": \"Request is invalid\",\
\"errors\": \[\
{\
\"field\": \"email\",\
\"code\": \"INVALID_EMAIL\",\
\"message\": \"Email không hợp lệ\"\
}\
\]\
}

Dùng scenario này để kiểm tra form mapping lỗi từ API xuống đúng field.

# 13. Mock HTTP Error

  --------------------------------------------------------------------------------
  Status                  Ý nghĩa                 Mục đích test
  ----------------------- ----------------------- --------------------------------
  400                     Validation / bad        Kiểm tra field/form error
                          request                 

  401                     Unauthorized            Kiểm tra session/authentication

  403                     Forbidden               Kiểm tra permission denied

  404                     Not Found               Kiểm tra detail/not-found state

  409                     Conflict                Kiểm tra
                                                  duplicate/concurrency/conflict

  500                     Server Error            Kiểm tra generic error state
  --------------------------------------------------------------------------------

# 14. Mock Permission

Với ứng dụng dùng chung cho User/Staff/Admin, nên có các scenario
permission cố định.

User:\
\[\"PROFILE_READ\", \"ORDER_READ\", \"ORDER_CREATE\"\]\
\
Staff:\
\[\"PROFILE_READ\", \"ORDER_READ\", \"ORDER_CREATE\",
\"ORDER_APPROVE\"\]\
\
Admin:\
\[\"PROFILE_READ\", \"ORDER_READ\", \"ORDER_CREATE\", \"ORDER_APPROVE\",
\"STAFF_MANAGE\", \"SYSTEM_CONFIG\"\]

Dùng các scenario này để kiểm tra menu, route, page và action/button.

# 15. Mock Authentication

POST /api/v1/auth/login\
GET /api/v1/me

Ví dụ /me trả về user và permissions. Đây chỉ là dữ liệu mô phỏng;
authentication/authorization thật phải theo API contract của hệ thống.

# 16. Dynamic Mock Data

Mockoon hỗ trợ Handlebars, Faker.js và custom helpers cho dynamic
response data. Xem: https://mockoon.com/docs/latest/templating/overview/

{\
\"id\": \"{{faker \'string.uuid\'}}\",\
\"name\": \"{{faker \'person.fullName\'}}\",\
\"email\": \"{{faker \'internet.email\'}}\"\
}

Nên dùng dynamic data cho volume/layout/pagination. Với business
scenario cần tái hiện ổn định, ưu tiên fixed data.

# 17. Slow Response

Thêm response delay, ví dụ 3000 ms, để kiểm tra loading state, skeleton,
button loading và chống duplicate submit.

# 18. Nhiều Scenario cho một API

Có thể tổ chức các scenario theo tên dễ tìm:

success\
empty\
validation-error\
unauthorized\
forbidden\
not-found\
conflict\
server-error\
slow

# 19. Import OpenAPI

Nếu đã có OpenAPI, có thể import OpenAPI v2 hoặc v3 vào Mockoon để tạo
environment ban đầu. Mockoon cũng hỗ trợ re-import để bổ sung
route/response mới. Xem:
https://mockoon.com/docs/latest/openapi/import-export-openapi-format/

OpenAPI\
↓\
Import vào Mockoon\
↓\
Mock Environment\
↓\
Frontend development

Lưu ý: compatibility giữa OpenAPI và mọi tính năng Mockoon là không hoàn
toàn 1:1; đặc biệt một số rule/body behavior của Mockoon không biểu diễn
đầy đủ trong OpenAPI. Vì vậy khi chia sẻ chính xác mock configuration,
nên dùng Mockoon data file. Xem:
https://mockoon.com/docs/latest/openapi/openapi-specification-compatibility/

# 20. Lưu và chia sẻ Mockoon bằng Git

Khuyến nghị lưu Mockoon data file trong repository, ví dụ:

mocks/\
└── mockoon/\
└── project-api.json

Không commit password, token thật, credential hoặc dữ liệu production.
Khi cần giữ nguyên đầy đủ behavior của mock, nên chia sẻ Mockoon data
file. Xem:
https://mockoon.com/docs/latest/openapi/openapi-specification-compatibility/

# 21. Chạy Mockoon bằng CLI

Khi cần chạy mock tự động hoặc trong CI/CD:

npm install -g \@mockoon/cli\
mockoon-cli start \--data ./mocks/mockoon/project-api.json \--port 3000

Mockoon CLI có thể chạy từ data file, Docker hoặc GitHub Actions. Xem:
https://mockoon.com/cli/

# 22. Quy trình tạo một Mock API mới

**1.** Xác định endpoint và method.

**2.** Tạo route trong Mockoon.

**3.** Tạo success response.

**4.** Thêm mock data.

**5.** Thêm error scenario cần thiết.

**6.** Test trực tiếp bằng curl/Postman/Frontend.

**7.** Kiểm tra loading, empty, validation và permission state.

**8.** Export/commit Mockoon data file.

**9.** Cập nhật API Contract/OpenAPI nếu project đã áp dụng
contract-first.

# 23. Cấu trúc thư mục khuyến nghị

project/\
├── docs/\
│ └── api/\
│ └── openapi.yaml\
├── mocks/\
│ └── mockoon/\
│ └── project-api.json\
└── src/\
├── api/\
├── auth/\
├── permissions/\
└── features/

# 24. Checklist trước khi hoàn thành Mock

-   Route và HTTP method đúng.

-   Request/response đã kiểm tra.

-   Success và Empty scenario đã có khi cần.

-   Validation error đã có khi cần.

-   401/403/404/409/500 đã được mô phỏng ở nơi phù hợp.

-   Permission scenario đã test.

-   Loading/slow scenario đã test nếu UI cần.

-   Mock data không chứa dữ liệu nhạy cảm.

-   Mockoon data file đã được lưu và chia sẻ qua Git.

# 25. Tài liệu tham khảo

-   Mockoon: https://mockoon.com/

-   OpenAPI import/export:
    https://mockoon.com/docs/latest/openapi/import-export-openapi-format/

-   Data Buckets:
    https://mockoon.com/docs/latest/data-buckets/using-data-buckets/

-   CRUD Routes:
    https://mockoon.com/docs/latest/api-endpoints/crud-routes/

-   Templating: https://mockoon.com/docs/latest/templating/overview/

-   CLI: https://mockoon.com/cli/
