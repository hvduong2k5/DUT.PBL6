import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';

Finder field(String label) => find.byWidgetPredicate(
  (widget) => widget is TextField && widget.decoration?.labelText == label,
);
void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  testWidgets(
    'guest support request opens actual ticket and appends public reply',
    (tester) async {
      Future<void> reveal(Finder target, double delta) => tester.scrollUntilVisible(target, delta, scrollable: find.byType(Scrollable).first);
      await tester.pumpWidget(
        const OmaApp(initialLocation: '/support/request'),
      );
      await tester.pumpAndSettle();
      await reveal(field('Tiêu đề yêu cầu'), 350);
      await tester.enterText(
        field('Tiêu đề yêu cầu'),
        'Hướng dẫn bảo quản bánh',
      );
      await reveal(field('Nội dung cần hỗ trợ'), 250);
      await tester.enterText(
        field('Nội dung cần hỗ trợ'),
        'Xin hướng dẫn cách bảo quản bánh sau khi mở hộp.',
      );
      await reveal(
        find.text('Gửi yêu cầu đến Ban Tri Kỷ'),
        300,
      );
      await tester.tap(find.text('Gửi yêu cầu đến Ban Tri Kỷ'));
      await tester.pumpAndSettle();
      expect(find.text('Hướng dẫn bảo quản bánh'), findsOneWidget);
      await reveal(field('Lời nhắn gửi Ban Tri Kỷ'), 250);
      await tester.enterText(
        field('Lời nhắn gửi Ban Tri Kỷ'),
        'Tôi dự định dùng bánh trong hai ngày.',
      );
      await reveal(find.text('Gửi phản hồi'), 200);
      await tester.tap(find.text('Gửi phản hồi'));
      await tester.pumpAndSettle();
      await tester.drag(find.byType(Scrollable).first, const Offset(0, 200));
      await tester.pumpAndSettle();
      expect(
        find.text('Tôi dự định dùng bánh trong hai ngày.'),
        findsOneWidget,
      );
      expect(tester.takeException(), isNull);
    },
  );
}
