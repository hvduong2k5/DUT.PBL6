import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/theme/app_theme.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/checkout/presentation/payment_page.dart';

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));

  testWidgets(
    'selected cart checkout uses two-level address and preserves other lines',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final dependencies = RepositoryFactory.create(
        mode: DataSourceMode.mock,
        localMvpCheckout: false,
      );
      await tester.runAsync(() async {
        final first = await dependencies.catalogRepository.product(
          'PROD-MX-GION-500',
        );
        final second = await dependencies.catalogRepository.product(
          'PROD-MX-DEO-300',
        );
        await dependencies.cartRepository.add(first, first.variants.first, 2);
        await dependencies.cartRepository.add(second, second.variants.first, 3);
      });
      await tester.pumpWidget(
        OmaApp(dependencies: dependencies, initialLocation: '/cart'),
      );
      await tester.pumpAndSettle();
      final checks = find.byType(Checkbox);
      await tester.tap(checks.at(2)); // Leave only first product selected.
      await tester.pumpAndSettle();
      await tester.tap(find.text('Thanh Toán'));
      await tester.pumpAndSettle();
      expect(find.text('Quận/Huyện *'), findsNothing);
      for (final input in {
        'Họ và tên *': 'Nguyễn Văn An',
        'Số điện thoại *': '0905123456',
        'Số nhà, tên đường *': '123 Lê Lợi',
      }.entries) {
        final field = find.widgetWithText(TextField, input.key);
        await tester.ensureVisible(field);
        await tester.enterText(field, input.value);
      }
      await tester.scrollUntilVisible(
        find.text('Tỉnh/Thành phố *'),
        100,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.tap(find.text('Tỉnh/Thành phố *'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Thành phố Huế').last);
      await tester.pumpAndSettle();
      final addressInput = find.descendant(
        of: find.widgetWithText(TextField, 'Số nhà, tên đường *'),
        matching: find.byType(EditableText),
      );
      expect(
        tester.widget<EditableText>(addressInput).focusNode.hasFocus,
        isFalse,
      );
      await tester.scrollUntilVisible(
        find.text('Phường/Xã *'),
        80,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.tap(find.text('Phường/Xã *'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Phường Thuận Hòa').last);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Đặt Hàng'));
      await tester.pumpAndSettle();
      expect(find.text('Đã tiếp nhận đơn COD'), findsOneWidget);
      final cart = await dependencies.cartRepository.getCart();
      expect(cart.items.single.product.id, 'PROD-MX-DEO-300');
      expect(cart.items.single.quantity, 3);
      final createdId = tester
          .widgetList<Text>(find.byType(Text))
          .map((text) => text.data)
          .whereType<String>()
          .firstWhere((text) => RegExp(r'^OM-\d+$').hasMatch(text));
      final orderFuture = dependencies.ordersRepository.order(createdId);
      await tester.pump(const Duration(milliseconds: 150));
      final order = await orderFuture;
      expect(order.total, Money.vnd(170000));
      expect(order.items.single.quantity, 2);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('invalid confirmation never fabricates successful order', (
    tester,
  ) async {
    await tester.pumpWidget(
      const OmaApp(initialLocation: '/order-confirmation/UNKNOWN'),
    );
    await tester.pumpAndSettle();
    expect(find.text('Đặt hàng thành công!'), findsNothing);
    expect(
      find.textContaining('Vui lòng đăng nhập hoặc xác minh'),
      findsOneWidget,
    );
  });

  testWidgets('expired payment hides QR and cannot mark itself paid', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.light,
        home: PaymentPage(
          orderId: 'ORD-1',
          confirmation: OrderConfirmation(
            orderId: 'ORD-1',
            status: 'PENDING_PAYMENT',
            total: Money.vnd(170000),
            paymentMethod: PaymentMethod.vietqr,
            vietQrUrl: 'https://example.test/qr.png',
            paymentExpiresAt: DateTime.now().subtract(
              const Duration(minutes: 1),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Thời hạn thanh toán đã kết thúc'), findsOneWidget);
    expect(find.byType(Image), findsNothing);
    expect(find.text('Thanh toán đã được ghi nhận'), findsNothing);
    await tester.pumpWidget(const SizedBox.shrink());
  });
}
