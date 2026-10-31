import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  for (final scale in [1.0, 1.6]) {
    testWidgets(
      'placed order history/payment/cancellation stays authoritative at text scale $scale',
      (tester) async {
        tester.view.physicalSize = const Size(360, 800);
        tester.view.devicePixelRatio = 1;
        tester.platformDispatcher.textScaleFactorTestValue = scale;
        addTearDown(tester.view.reset);
        addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
        final d = RepositoryFactory.create(mode: DataSourceMode.mock);
        await tester.runAsync(() async {
          await d.authRepository.login(
            phoneNumber: '0905123456',
            password: 'local-fixture',
          );
          final product = await d.catalogRepository.product('PROD-MX-GION-500');
          await d.cartRepository.add(product, product.variants.first, 2);
          final purchase = d.mvpPurchaseRepository!;
          final cart = await purchase.cart(),
              addresses = await purchase.savedAddresses();
          final quote = await purchase.quote(
            cartRevision: cart.revision,
            addressId: addresses.first.id,
          );
          final checkout = d.mvpCheckoutRuntime!.repository;
          final accepted = await checkout.accept(
            MvpCheckoutInput.fromQuote(quote),
            key: 'fixture-order-123456',
          );
          await checkout.operation(accepted.id);
          await checkout.operation(accepted.id);
        });
        await tester.pumpWidget(
          OmaApp(initialLocation: '/orders', dependencies: d),
        );
        await tester.pumpAndSettle();
        expect(find.text('LOCAL-1'), findsOneWidget);
        expect(find.byType(TextField), findsNothing);
        await tester.tap(find.text('LOCAL-1'));
        await tester.pumpAndSettle();
        await tester.scrollUntilVisible(find.text('Xem thanh toán'), 200);
        await tester.tap(find.text('Xem thanh toán'));
        await tester.pumpAndSettle();
        expect(find.text('Chờ thanh toán'), findsOneWidget);
        expect(find.byType(Image), findsNothing);
        expect(
          find.textContaining('chưa có tài khoản nhận tiền'),
          findsOneWidget,
        );
        await tester.scrollUntilVisible(
          find.text('Kiểm tra trạng thái thanh toán'),
          180,
        );
        await tester.tap(find.text('Kiểm tra trạng thái thanh toán'));
        await tester.pumpAndSettle();
        await tester.scrollUntilVisible(find.text('Chờ thanh toán'), -180);
        expect(find.text('Chờ thanh toán'), findsOneWidget);
        await tester.scrollUntilVisible(find.text('Xem đơn hàng'), 150);
        await tester.tap(find.text('Xem đơn hàng'));
        await tester.pumpAndSettle();
        await tester.scrollUntilVisible(find.text('Yêu cầu hủy đơn'), 200);
        await tester.tap(find.text('Yêu cầu hủy đơn'));
        await tester.pumpAndSettle();
        final send = find.widgetWithText(FilledButton, 'Gửi yêu cầu hủy');
        expect(tester.widget<FilledButton>(send).onPressed, isNull);
        await tester.enterText(
          find.byType(TextField),
          'Thử nghiệm hủy đơn local',
        );
        await tester.pump();
        expect(tester.widget<FilledButton>(send).onPressed, isNotNull);
        await tester.ensureVisible(send);
        await tester.tap(send);
        await tester.pumpAndSettle();
        expect(find.byType(AlertDialog), findsNothing);
        await tester.scrollUntilVisible(find.text('Đã hủy theo yêu cầu'), -200);
        expect(find.text('Đã hủy theo yêu cầu'), findsOneWidget);
        expect(find.text('Đã hoàn tiền'), findsNothing);
        await tester.scrollUntilVisible(find.text('Danh sách đơn hàng'), 200);
        final historyButton = find.widgetWithText(
          OutlinedButton,
          'Danh sách đơn hàng',
        );
        await tester.ensureVisible(historyButton);
        await tester.pumpAndSettle();
        await tester.tap(historyButton);
        await tester.pumpAndSettle();
        expect(find.textContaining('Đã hủy theo yêu cầu'), findsOneWidget);
        expect((await d.cartRepository.getCart()).totalItems, 2);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
      },
    );
  }
}
