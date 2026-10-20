import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';

Finder field(String label) => find.byWidgetPredicate(
  (w) => w is TextField && w.decoration?.labelText == label,
);
void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  testWidgets(
    'phone return form creates a real local case without implying refund',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final dependencies = RepositoryFactory.create(mode: DataSourceMode.mock);
      await tester.runAsync(() async {
        await dependencies.authRepository.login(
          phoneNumber: '0905123456',
          password: 'legacy',
        );
      });
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/orders/OM-8874/after-sales/request',
          dependencies: dependencies,
        ),
      );
      await tester.pumpAndSettle();
      Future<void> reveal(Finder target) async {
        await tester.scrollUntilVisible(
          target,
          250,
          scrollable: find.byType(Scrollable).first,
        );
        await Scrollable.ensureVisible(tester.element(target), alignment: 0.3);
        await tester.pumpAndSettle();
      }

      await tester.tap(find.byType(DropdownButtonFormField<String>).first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Kẹo Mè Xửng Dẻo · Hộp 250g').last);
      await tester.pumpAndSettle();
      await reveal(find.text('Lý do yêu cầu'));
      await tester.tap(find.text('Lý do yêu cầu'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Vấn đề chất lượng').last);
      await tester.pumpAndSettle();
      for (final entry in {
        'Mô tả tình trạng': 'Hộp bánh cần được kiểm tra chất lượng.',
        'URL bằng chứng': 'https://example.com/evidence.jpg',
        'Mã ngân hàng': 'LOCALBANK',
        'Số tài khoản': '12345678',
        'Tên chủ tài khoản': 'KHACH LOCAL',
      }.entries) {
        await reveal(field(entry.key));
        await tester.enterText(field(entry.key), entry.value);
      }
      await reveal(find.text('Gửi yêu cầu đổi trả'));
      await tester.tap(find.text('Gửi yêu cầu đổi trả'));
      await tester.pumpAndSettle();
      expect(find.text('Hồ sơ đổi trả'), findsOneWidget);
      expect(find.text('Đã tiếp nhận yêu cầu'), findsOneWidget);
      await reveal(
        find.textContaining('Chưa có xác nhận hoàn tiền thành công'),
      );
      expect(
        find.textContaining('Chưa có xác nhận hoàn tiền thành công'),
        findsOneWidget,
      );
      expect(tester.takeException(), isNull);
    },
  );
}
