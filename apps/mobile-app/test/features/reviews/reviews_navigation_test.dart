import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  testWidgets(
    'phone product review selects owned order and exact SKU then refreshes verified public list',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final deps = RepositoryFactory.create(mode: DataSourceMode.mock);
      await tester.runAsync(() async {
        await deps.authRepository.login(
          phoneNumber: '0905123456',
          password: 'legacy',
        );
      });
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/product/PROD-MX-GION-500',
          dependencies: deps,
        ),
      );
      await tester.pumpAndSettle();
      Future<void> reveal(Finder target, Finder scroll) async {
        await tester.scrollUntilVisible(target, 250, scrollable: scroll);
        await Scrollable.ensureVisible(tester.element(target), alignment: .3);
        await tester.pumpAndSettle();
      }

      await reveal(find.text('Viết đánh giá'), find.byType(Scrollable).first);
      await tester.tap(find.text('Viết đánh giá'));
      await tester.pumpAndSettle();
      await tester.tap(find.byType(DropdownButtonFormField<String>).first);
      await tester.pumpAndSettle();
      await tester.tap(find.textContaining('#OM-8860').last);
      await tester.pumpAndSettle();
      expect(find.text('Hộp 500g'), findsWidgets);
      final comment = find.byWidgetPredicate(
        (w) => w is TextField && w.decoration?.labelText == 'Cảm nhận của bạn',
      );
      final modalScroll = find.byType(SingleChildScrollView).last;
      final scroll = find
          .descendant(of: modalScroll, matching: find.byType(Scrollable))
          .first;
      await reveal(comment, scroll);
      await tester.showKeyboard(comment);
      tester.view.viewInsets = const FakeViewPadding(bottom: 400);
      addTearDown(tester.view.resetViewInsets);
      await tester.pumpAndSettle();
      final editable = tester.state<EditableTextState>(
        find.descendant(of: comment, matching: find.byType(EditableText)),
      );
      final caret = editable.renderEditable.getLocalRectForCaret(
        const TextPosition(offset: 0),
      );
      final caretBottom = editable.renderEditable
          .localToGlobal(caret.bottomLeft)
          .dy;
      expect(
        caretBottom,
        lessThan(444),
        reason:
            'Focused review caret must remain above the 400px keyboard without a manual scroll',
      );
      await tester.enterText(comment, 'Mè giòn thơm, đóng gói rất cẩn thận.');
      await reveal(find.text('Gửi đánh giá'), scroll);
      await tester.tap(find.text('Gửi đánh giá'));
      await tester.pumpAndSettle();
      tester.view.resetViewInsets();
      await tester.pumpAndSettle();
      await reveal(find.text('Đã mua sản phẩm'), find.byType(Scrollable).first);
      expect(find.text('Mè giòn thơm, đóng gói rất cẩn thận.'), findsOneWidget);
      expect(find.text('5 trên 5 · 1 đánh giá'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
