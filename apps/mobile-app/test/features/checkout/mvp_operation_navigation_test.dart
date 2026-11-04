import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  testWidgets(
    'full cart confirmation creates one operation; leaving and returning preserves it',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final d = RepositoryFactory.create(mode: DataSourceMode.mock);
      await tester.runAsync(() async {
        await d.authRepository.login(
          phoneNumber: '0905123456',
          password: 'local-fixture',
        );
        for (final id in ['PROD-MX-GION-500', 'PROD-MX-DEO-300']) {
          final product = await d.catalogRepository.product(id);
          await d.cartRepository.add(product, product.variants.first, 2);
        }
      });
      await tester.pumpWidget(
        OmaApp(initialLocation: '/cart', dependencies: d),
      );
      await tester.pumpAndSettle();
      expect(find.byType(Checkbox), findsNothing);
      expect(find.text('Thanh Toán'), findsNothing);
      await tester.tap(find.text('Xem báo giá'));
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(find.text('Lấy báo giá cả giỏ'), 250);
      await tester.tap(find.text('Lấy báo giá cả giỏ'));
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(find.text('Xác nhận đặt hàng'), 150);
      await tester.tap(find.text('Xác nhận đặt hàng'));
      await tester.pumpAndSettle();
      expect(find.text('Đang tạo đơn hàng'), findsOneWidget);
      final requestId = tester
          .widget<SelectableText>(find.byType(SelectableText))
          .data;
      await tester.tap(find.text('Kiểm tra lại yêu cầu'));
      await tester.pumpAndSettle();
      expect(find.text('Đang tạo đơn hàng'), findsOneWidget);
      await tester.tap(find.text('Về giỏ hàng'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Tiếp tục đơn'));
      await tester.pumpAndSettle();
      expect(find.text(requestId!), findsOneWidget);
      await tester.tap(find.text('Kiểm tra lại yêu cầu'));
      await tester.pumpAndSettle();
      expect(find.text('Đã tạo đơn hàng'), findsOneWidget);
      expect(find.byType(SelectableText), findsNWidgets(2));
      expect((await d.cartRepository.getCart()).totalItems, 4);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  testWidgets(
    'legacy checkout deep link cannot bypass full-cart review in new cohort',
    (tester) async {
      final d = RepositoryFactory.create(mode: DataSourceMode.mock);
      await tester.runAsync(() async {
        await d.authRepository.login(
          phoneNumber: '0905123456',
          password: 'local-fixture',
        );
        final product = await d.catalogRepository.product('PROD-MX-GION-500');
        await d.cartRepository.add(product, product.variants.first, 1);
      });
      await tester.pumpWidget(
        OmaApp(initialLocation: '/checkout?items=only-one', dependencies: d),
      );
      await tester.pumpAndSettle();
      expect(find.text('Báo giá cả giỏ'), findsOneWidget);
      expect(find.text('Đặt Hàng'), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
}
