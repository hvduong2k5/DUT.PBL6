import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  testWidgets(
    'guest sign-in returns to profile and saves valid contact changes',
    (tester) async {
      await tester.pumpWidget(
        const OmaApp(initialLocation: '/account/profile'),
      );
      await tester.pumpAndSettle();
      expect(find.text('Đăng nhập để tiếp tục'), findsOneWidget);
      await tester.tap(find.text('Đăng nhập'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).at(0), '0905123456');
      await tester.enterText(find.byType(TextField).at(1), 'old');
      await tester.ensureVisible(find.text('Đăng nhập').last);
      await tester.tap(find.text('Đăng nhập').last);
      await tester.pumpAndSettle();
      expect(find.text('Hồ sơ cá nhân'), findsOneWidget);
      await tester.enterText(
        find.widgetWithText(TextField, 'Họ và tên'),
        'Nguyễn An Tri Kỷ',
      );
      await tester.enterText(
        find.widgetWithText(TextField, 'Email liên hệ'),
        'new@example.com',
      );
      await tester.ensureVisible(find.text('Lưu thay đổi'));
      await tester.tap(find.text('Lưu thay đổi'));
      await tester.pumpAndSettle();
      expect(find.text('Hồ sơ Tri Kỷ đã được cập nhật.'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
