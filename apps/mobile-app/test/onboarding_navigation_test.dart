import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/storage/onboarding_store.dart';

void main() {
  setUp(() {
    FlutterSecureStorage.setMockInitialValues({});
  });

  test('onboarding completion is persisted locally', () async {
    final store = OnboardingStore();
    expect(await store.isCompleted(), isFalse);
    await store.complete();
    expect(await store.isCompleted(), isTrue);
  });

  testWidgets('first launch offers guest entry and opens Home', (tester) async {
    final store = OnboardingStore();
    await tester.pumpWidget(
      OmaApp(onboardingStore: store, onboardingCompleted: false),
    );
    await tester.pumpAndSettle();

    expect(find.text('Tinh hoa quà Huế'), findsOneWidget);
    await tester.tap(find.text('Tiếp tục'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Tiếp tục'));
    await tester.pumpAndSettle();

    expect(find.text('Đăng nhập'), findsOneWidget);
    expect(find.text('Đăng ký'), findsOneWidget);
    expect(find.text('Tiếp tục với tư cách khách'), findsOneWidget);

    await tester.tap(find.text('Tiếp tục với tư cách khách'));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('home-search-field')), findsOneWidget);
    expect(await store.isCompleted(), isTrue);
  });

  testWidgets('Android back from root Login returns to guest Home', (
    tester,
  ) async {
    await tester.pumpWidget(const OmaApp(initialLocation: AppRoutes.login));
    await tester.pumpAndSettle();
    expect(find.text('Đăng Nhập'), findsOneWidget);

    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('home-search-field')), findsOneWidget);
  });

  testWidgets('visible Login back button falls back to guest Home', (
    tester,
  ) async {
    await tester.pumpWidget(const OmaApp(initialLocation: AppRoutes.login));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Quay lại'));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('home-search-field')), findsOneWidget);
  });

  testWidgets('bottom navigation opens the empty guest cart', (tester) async {
    await tester.pumpWidget(const OmaApp());
    await tester.pumpAndSettle();

    await tester.tap(find.text('Giỏ Hàng'));
    await tester.pumpAndSettle();

    expect(find.text('Giỏ hàng trống'), findsOneWidget);
  });

  testWidgets('Home menu opens and offers functional guest navigation', (
    tester,
  ) async {
    await tester.pumpWidget(const OmaApp());
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Mở menu'));
    await tester.pumpAndSettle();
    expect(find.text('Mua sắm với tư cách khách'), findsOneWidget);
    expect(find.text('Đăng nhập'), findsOneWidget);
    expect(find.text('Địa chỉ giao hàng'), findsNothing);

    await tester.tap(find.text('Đăng nhập'));
    await tester.pumpAndSettle();
    expect(find.text('Đăng Nhập'), findsOneWidget);
  });

  testWidgets('Android back closes the menu before leaving Home', (
    tester,
  ) async {
    await tester.pumpWidget(const OmaApp());
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Mở menu'));
    await tester.pumpAndSettle();

    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();

    expect(find.text('Mua sắm với tư cách khách'), findsNothing);
    expect(find.byKey(const Key('home-search-field')), findsOneWidget);
  });

  testWidgets('Android back from root Cart returns Home', (tester) async {
    await tester.pumpWidget(const OmaApp(initialLocation: AppRoutes.cart));
    await tester.pumpAndSettle();
    expect(find.text('Giỏ hàng trống'), findsOneWidget);

    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('home-search-field')), findsOneWidget);
  });

  testWidgets('Home fits a 390px viewport and keeps bottom navigation', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(const OmaApp());
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('home-search-field')), findsOneWidget);
    expect(find.text('Trang Chủ'), findsOneWidget);
    expect(find.text('Thực Đơn'), findsOneWidget);
    expect(find.text('Giỏ Hàng'), findsOneWidget);
    expect(find.text('Tài Khoản'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
