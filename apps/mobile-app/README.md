# DUT.PBL6 Mobile App

Ứng dụng Flutter dành cho khách hàng của DUT.PBL6 / Ô Mạ, bao gồm duyệt sản phẩm, giỏ hàng, thanh toán, đơn hàng, tài khoản, địa chỉ và thông báo.

## Yêu cầu

- Flutter `3.44.9` (stable) và Dart `3.12.2` — phiên bản đã dùng để kiểm tra dự án.
- Android Studio và Android SDK để chạy/build Android.
- Android Emulator hoặc thiết bị Android thật đã bật chế độ gỡ lỗi USB.
- Kiểm tra môi trường bằng:

```bash
flutter doctor
flutter devices
```

Windows không thể build ứng dụng iOS. Muốn build/test iOS cần macOS, Xcode và cấu hình signing phù hợp.

## Vị trí dự án

Từ thư mục gốc repository:

```text
apps/mobile-app
```

- Tên thư mục trên filesystem: `mobile-app`
- Tên package Dart trong `pubspec.yaml`: `mobile_app`

Không đổi import Dart từ `package:mobile_app/...` thành `mobile-app`.

## Cài đặt dependencies

```bash
cd apps/mobile-app
flutter pub get
```

## Chạy ứng dụng

Cách đơn giản nhất (mặc định an toàn về local mock):

```bash
flutter run
```

Nếu không truyền `DATA_SOURCE`, hoặc truyền giá trị không hợp lệ, app tự dùng `mock`. App **không cần backend hoặc Mockoon** để mở và kiểm thử các luồng chính.

Chọn thiết bị cụ thể:

```bash
flutter devices
flutter run -d <device-id>
```

Không hard-code ID thiết bị cá nhân vào source hoặc tài liệu.

## Android Emulator và localhost

Trong Android Emulator, `localhost` trỏ tới chính máy ảo Android, không phải máy Windows đang chạy server. Để truy cập một server trên máy host từ Android Emulator, dùng `10.0.2.2` thay cho `localhost`.

Ví dụ, Prism chạy trên cổng `4010` của máy host sẽ được emulator nhìn thấy tại `http://10.0.2.2:4010`.

Với thiết bị Android thật, `10.0.2.2` không áp dụng; dùng địa chỉ IP LAN của máy chạy server và bảo đảm firewall/network cho phép kết nối.

## Chế độ dữ liệu

Runtime được chọn tập trung bằng ba compile-time define:

- `DATA_SOURCE`: `mock`, `mockoon`, hoặc `backend`. Thiếu/rỗng/không hợp lệ đều fallback về `mock`.
- `APP_ENV`: `mock` (mặc định), `development`, hoặc `production`.
- `API_BASE_URL`: ghi đè base URL nếu giá trị không rỗng.

Các profile không chứa secret nằm trong thư mục `config/`. Chạy local mock:

```bash
flutter run --dart-define-from-file=config/mock.json
```

Chạy Mockoon từ Android Emulator (Mockoon trên Windows tại cổng `4010`):

```bash
flutter run --dart-define-from-file=config/mockoon.json
```

Chạy Mockoon trên web:

```bash
flutter run -d chrome \
  --dart-define-from-file=config/mockoon.json \
  --dart-define=API_BASE_URL=http://localhost:4010
```

Chạy Mockoon từ thiết bị Android thật:

```bash
flutter run \
  --dart-define-from-file=config/mockoon.json \
  --dart-define=API_BASE_URL=http://<WINDOWS_LAN_IP>:4010
```

Chạy với backend thật:

```bash
flutter run \
  --dart-define-from-file=config/backend.json \
  --dart-define=API_BASE_URL=<REAL_API_GATEWAY>
```

Có thể dùng `--dart-define` trực tiếp thay cho file. `API_BASE_URL` luôn có ưu tiên cao nhất. Nếu không ghi đè, cấu hình mặc định nhận biết platform mà không dùng `dart:io`: Android Emulator dùng `10.0.2.2`; web/desktop dùng `localhost`. Mockoon dùng cổng `4010`, backend development dùng cổng `8000`, và backend production mặc định dùng `https://api.omama.vn`.

`RepositoryFactory` trong `lib/app/app_dependencies.dart` là điểm duy nhất lựa chọn repository. `mockoon` và `backend` dùng cùng các lớp REST; chỉ URL/môi trường khác nhau.

| Feature | `mock` | `mockoon` / `backend` |
| --- | --- | --- |
| Catalog, Home, Search, Product Detail | `MockCatalogRepository` | `RestCatalogRepository` |
| Cart | `InMemoryCartRepository` | `RestCartRepository` |
| Checkout, Order Confirmation | `MockCheckoutRepository` | `MockCheckoutRepository` |
| Order History, Order Detail/Tracking | `MockOrdersRepository` | `RestOrdersRepository` |
| Addresses | `InMemoryAddressRepository` | `RestAddressRepository` |
| Login/Register/Session | `MockAuthRepository` | `RestAuthRepository` |
| Notifications | `LocalNotificationsRepository` | `LocalNotificationsRepository` |

Checkout vẫn local có chủ ý: model UI hiện chỉ có một chuỗi địa chỉ, chưa đủ các trường địa chỉ có cấu trúc mà OpenAPI bắt buộc. Notifications cũng giữ local vì hợp đồng không có endpoint tương ứng. Không có endpoint nào được tự suy đoán.

## Mockoon / OpenAPI mock

Mockoon không cần thiết cho runtime `mock`. Import hợp đồng OpenAPI vào Mockoon, chạy server host tại `http://localhost:4010`, rồi dùng các lệnh `mockoon` ở trên.

Nếu muốn dùng Prism thay thế, có thể khởi động thủ công:

```bash
npx --yes @stoplight/prism-cli mock <path-to-openapi_d2c.yaml> -p 4010
```

Sau đó dùng profile `mockoon` với URL thích hợp cho platform. Không đặt đường dẫn tuyệt đối riêng của máy cá nhân vào cấu hình dự án.

Dữ liệu local/in-memory sẽ trở về trạng thái ban đầu khi app/process khởi động lại. Authentication trong mode `mock` không gọi API và không khôi phục session từ secure storage.

## Kiểm thử và chất lượng

Chạy test:

```bash
flutter test
```

Baseline gần nhất: **32/32 test pass**.

Phân tích tĩnh:

```bash
flutter analyze
```

Định dạng:

```bash
dart format .
```

Build APK debug:

```bash
flutter build apk --debug
```

APK được tạo tại:

```text
build/app/outputs/flutter-apk/app-debug.apk
```

## Checklist kiểm thử thủ công

- [ ] Home tải đúng và không overflow.
- [ ] Catalog, lọc danh mục và tìm kiếm.
- [ ] Product Detail, tăng/giảm số lượng và thêm vào giỏ.
- [ ] Thêm sản phẩm, đổi số lượng và xóa sản phẩm trong Cart.
- [ ] Empty Cart không hiển thị badge/tổng tiền/CTA thanh toán.
- [ ] Checkout validation, lựa chọn thanh toán và chống submit lặp.
- [ ] Đặt hàng và Order Confirmation.
- [ ] Order History, lọc trạng thái và empty state.
- [ ] Order Detail/Tracking timeline.
- [ ] Guest Account và điều hướng tới Login/Register.
- [ ] Login/Register: validation, loading, lỗi và đăng xuất.
- [ ] Address: thêm, sửa, xóa và đặt mặc định.
- [ ] Notifications và empty state nếu dùng repository rỗng.
- [ ] Loading, empty và error states khi có thể tái hiện.
- [ ] Offline/network behavior khi REST được nối vào runtime.
- [ ] Back navigation và Bottom Navigation.
- [ ] UI không overflow trên viewport rộng khoảng 390 px.

## Giới hạn hiện tại

- Runtime mặc định là `mock`; chọn `mockoon` hoặc `backend` sẽ chuyển các feature có hợp đồng sang REST qua `RepositoryFactory`.
- Checkout vẫn dùng local mock ở mọi mode vì model UI hiện chưa biểu diễn đầy đủ địa chỉ có cấu trúc bắt buộc bởi OpenAPI. Profile/account chưa có repository riêng để nối REST.
- Hợp đồng backend hiện không định nghĩa notifications endpoint, nên notifications cố ý dùng local repository; không tự bịa API.
- `ConnectivityCubit` và `AppOfflineBanner` đã tồn tại nhưng chưa được cung cấp/nối vào app shell; offline UX end-to-end chưa hoàn chỉnh.
- `TokenStore` dùng secure storage và REST Auth có lưu/xóa token, nhưng mock auth mặc định không persist/restore session. Automatic refresh-token retry khi gặp `401` chưa được triển khai.
- Ảnh sản phẩm hiện dùng icon placeholder; logo, ảnh sản phẩm và font Be Vietnam Pro/Noto Serif được duyệt chưa được bundle.
- Độ khớp pixel tuyệt đối với Figma vẫn cần kiểm tra trên thiết bị thật/emulator.

## Kiến trúc

Luồng chính:

```text
Widget
  → Cubit
  → Repository
  → RemoteDataSource (khi có)
  → Dio / ApiClient
```

`go_router` quản lý điều hướng; `flutter_bloc` quản lý state; Dio, error mapping, token storage, money units/nanos và theme/shared widgets nằm trong `lib/core`.
