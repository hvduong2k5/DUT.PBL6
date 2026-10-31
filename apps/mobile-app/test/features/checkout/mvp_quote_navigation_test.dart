import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/theme/app_theme.dart';
import 'package:mobile_app/features/checkout/data/local_mvp_purchase_repository.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_repository.dart';
import 'package:mobile_app/features/checkout/presentation/mvp_quote_cubit.dart';
import 'package:mobile_app/features/checkout/presentation/mvp_quote_page.dart';

Future<void> seedCart(AppDependencies d) async {
  final a = await d.catalogRepository.product('PROD-MX-GION-500');
  final b = await d.catalogRepository.product('PROD-MX-DEO-300');
  await d.cartRepository.add(a, a.variants.first, 1);
  await d.cartRepository.add(b, b.variants.first, 2);
}

Future<void> login(AppDependencies d) => d.authRepository
    .login(phoneNumber: '0905123456', password: 'local-fixture')
    .then((_) {});

class _Scenario implements MvpPurchaseRepository {
  _Scenario(this.delegate);
  final MvpPurchaseRepository delegate;
  bool offline = true, noAddresses = false;
  Completer<MvpQuote>? pending;
  int quoteCalls = 0;
  @override
  Future<MvpCart> cart() {
    if (offline) throw StateError('Local transport failure');
    return delegate.cart();
  }

  @override
  Future<List<MvpSavedAddress>> savedAddresses() async =>
      noAddresses ? [] : delegate.savedAddresses();

  @override
  Future<MvpQuote> quote({
    required int cartRevision,
    required String addressId,
  }) {
    quoteCalls++;
    return pending?.future ??
        delegate.quote(cartRevision: cartRevision, addressId: addressId);
  }
}

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test(
    'local quote shares complete cart, preserves cart and rejects changed revision/identity',
    () async {
      final d = RepositoryFactory.create(mode: DataSourceMode.mock);
      await login(d);
      await seedCart(d);
      final r = d.mvpPurchaseRepository!;
      final cart = await r.cart();
      final quote = await r.quote(
        cartRevision: cart.revision,
        addressId: LocalMvpPurchaseRepository.addressId,
      );
      expect(quote.items.length, 2);
      expect(quote.items.fold<int>(0, (n, l) => n + l.quantity), 3);
      expect((await d.cartRepository.getCart()).items.length, 2);
      final old = (await d.cartRepository.getCart()).items.first;
      await d.cartRepository.update(old.itemId, 3);
      await expectLater(
        r.quote(
          cartRevision: cart.revision,
          addressId: LocalMvpPurchaseRepository.addressId,
        ),
        throwsA(isA<ApiException>()),
      );
      await d.authRepository.logout();
      await expectLater(r.cart(), throwsA(isA<ApiException>()));
    },
  );
  testWidgets(
    'cart quote entry returns through login and reviews all lines without placing order',
    (tester) async {
      final d = RepositoryFactory.create(mode: DataSourceMode.mock);
      await tester.runAsync(() => seedCart(d));
      await tester.pumpWidget(
        OmaApp(initialLocation: '/cart', dependencies: d),
      );
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Xem báo giá cả giỏ'));
      await tester.tap(find.text('Xem báo giá cả giỏ'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Đăng nhập'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).first, '0905123456');
      await tester.enterText(find.byType(TextField).at(1), 'local-fixture');
      await tester.ensureVisible(find.text('Đăng nhập'));
      await tester.tap(find.text('Đăng nhập'));
      await tester.pumpAndSettle();
      expect(find.text('Báo giá cả giỏ'), findsOneWidget);
      expect(find.byType(Checkbox), findsNothing);
      await tester.scrollUntilVisible(find.text('Lấy báo giá cả giỏ'), 250);
      await tester.tap(find.text('Lấy báo giá cả giỏ'));
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(find.text('Tổng báo giá'), -250);
      expect(find.text('Tổng báo giá'), findsOneWidget);
      await tester.scrollUntilVisible(
        find.textContaining('Chưa thể đặt hàng'),
        100,
      );
      expect(find.textContaining('Chưa thể đặt hàng'), findsOneWidget);
      expect((await d.cartRepository.getCart()).items.length, 2);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  testWidgets(
    'narrow large-text quote expires and one-tap requote replaces old authority',
    (tester) async {
      tester.view.physicalSize = const Size(360, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      var now = DateTime.utc(2026, 10, 31);
      final d = RepositoryFactory.create(mode: DataSourceMode.mock);
      await tester.runAsync(() => login(d));
      await tester.runAsync(() => seedCart(d));
      final r = LocalMvpPurchaseRepository(
        d.cartRepository,
        () => true,
        now: () => now,
      );
      final c = MvpQuoteCubit(r, now: () => now);
      await c.load();
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.light,
          home: MediaQuery(
            data: const MediaQueryData(
              size: Size(360, 800),
              textScaler: TextScaler.linear(1.6),
            ),
            child: BlocProvider.value(
              value: c,
              child: MvpQuotePage(localPreview: true, now: () => now),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(find.text('Lấy báo giá cả giỏ'), 250);
      await tester.tap(find.text('Lấy báo giá cả giỏ'));
      await tester.pumpAndSettle();
      final expiredId = c.state.quote!.id;
      now = now.add(const Duration(minutes: 3));
      await tester.pump(const Duration(seconds: 1));
      await tester.scrollUntilVisible(
        find.textContaining('Báo giá đã hết hạn'),
        150,
      );
      expect(find.textContaining('Báo giá đã hết hạn'), findsOneWidget);
      await tester.scrollUntilVisible(find.text('Lấy báo giá mới'), 150);
      await tester.tap(find.text('Lấy báo giá mới'));
      await tester.pumpAndSettle();
      expect(c.state.quote!.id, isNot(expiredId));
      expect(c.state.usableAt(now), isTrue);
      await tester.scrollUntilVisible(find.text('Tổng báo giá'), -150);
      expect(find.text('Tổng báo giá'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await c.close();
    },
  );
  testWidgets(
    'quote error retries through missing address; pending disables duplicate taps',
    (tester) async {
      final d = RepositoryFactory.create(mode: DataSourceMode.mock);
      await tester.runAsync(() => login(d));
      await tester.runAsync(() => seedCart(d));
      final r = _Scenario(d.mvpPurchaseRepository!);
      final c = MvpQuoteCubit(r);
      await c.load();
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.light,
          home: BlocProvider.value(value: c, child: const MvpQuotePage()),
        ),
      );
      expect(find.text('Thử lại'), findsOneWidget);
      r.offline = false;
      r.noAddresses = true;
      await tester.tap(find.text('Thử lại'));
      await tester.pumpAndSettle();
      expect(find.text('Chưa có địa chỉ phù hợp'), findsOneWidget);
      expect(find.text('Lấy báo giá cả giỏ'), findsNothing);
      expect((await d.cartRepository.getCart()).items.length, 2);
      r.noAddresses = false;
      await tester.tap(find.text('Tải lại địa chỉ'));
      await tester.pumpAndSettle();
      r.pending = Completer<MvpQuote>();
      await tester.scrollUntilVisible(find.text('Lấy báo giá cả giỏ'), 250);
      await tester.tap(find.text('Lấy báo giá cả giỏ'));
      await tester.pump();
      expect(find.text('Đang xử lý...'), findsOneWidget);
      final button = tester.widget<FilledButton>(find.byType(FilledButton));
      expect(button.onPressed, isNull);
      expect(
        tester
            .widget<DropdownButtonFormField<String>>(
              find.byType(DropdownButtonFormField<String>),
            )
            .onChanged,
        isNull,
      );
      expect(r.quoteCalls, 1);
      final cart = await d.mvpPurchaseRepository!.cart();
      r.pending!.complete(
        await d.mvpPurchaseRepository!.quote(
          cartRevision: cart.revision,
          addressId: LocalMvpPurchaseRepository.addressId,
        ),
      );
      await tester.pumpAndSettle();
      expect(c.state.status, MvpQuoteStatus.ready);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await c.close();
    },
  );
  testWidgets('empty cart has no quote action and remains empty', (
    tester,
  ) async {
    final d = RepositoryFactory.create(mode: DataSourceMode.mock);
    await tester.runAsync(() => login(d));
    final c = MvpQuoteCubit(d.mvpPurchaseRepository!);
    await c.load();
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.light,
        home: BlocProvider.value(value: c, child: const MvpQuotePage()),
      ),
    );
    expect(find.text('Giỏ hàng trống'), findsOneWidget);
    expect(find.text('Lấy báo giá cả giỏ'), findsNothing);
    expect((await d.cartRepository.getCart()).items, isEmpty);
    await tester.pumpWidget(const SizedBox.shrink());
    await c.close();
  });
}
