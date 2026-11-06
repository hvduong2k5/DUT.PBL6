import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  testWidgets(
    'guest lookup verifies contact and opens only granted order route',
    (tester) async {
      await tester.pumpWidget(const OmaApp(initialLocation: '/track-order'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).at(0), 'OM-8921');
      await tester.enterText(find.byType(TextField).at(1), '0905123456');
      await tester.ensureVisible(find.text('Gửi mã xác minh'));
      await tester.tap(find.text('Gửi mã xác minh'));
      await tester.pumpAndSettle();
      expect(find.textContaining('Nếu thông tin phù hợp'), findsOneWidget);
      await tester.enterText(find.byType(TextField), '789214');
      await tester.ensureVisible(find.text('Xác minh & mở đơn'));
      await tester.tap(find.text('Xác minh & mở đơn'));
      await tester.pumpAndSettle();
      expect(find.text('Đơn hàng #OM-8921'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
