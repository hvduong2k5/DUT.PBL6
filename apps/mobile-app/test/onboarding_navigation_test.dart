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
    expect(find.byKey(const Key('home-content-scroll')), findsOneWidget);
    expect(await store.isCompleted(), isTrue);
  });

  testWidgets('Android back from root Login returns to guest Home', (
    tester,
  ) async {
    await tester.pumpWidget(const OmaApp(initialLocation: AppRoutes.login));
    await tester.pumpAndSettle();
    expect(find.text('Đăng nhập tài khoản'), findsOneWidget);

    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('home-content-scroll')), findsOneWidget);
  });

  testWidgets('visible Login back button falls back to guest Home', (
    tester,
  ) async {
    await tester.pumpWidget(const OmaApp(initialLocation: AppRoutes.login));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Quay lại'));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('home-content-scroll')), findsOneWidget);
  });

  testWidgets('bottom navigation opens the empty guest cart', (tester) async {
    await tester.pumpWidget(const OmaApp());
    await tester.pumpAndSettle();

    await tester.tap(find.text('Giỏ hàng'));
    await tester.pumpAndSettle();

    expect(find.text('Giỏ hàng trống'), findsOneWidget);
  });

  testWidgets('Account tab replaces the side menu for guest navigation', (
    tester,
  ) async {
    await tester.pumpWidget(const OmaApp());
    await tester.pumpAndSettle();
    expect(find.byTooltip('Mở menu'), findsNothing);

    await tester.tap(find.text('Tài khoản'));
    await tester.pumpAndSettle();
    expect(find.text('Tra cứu đơn khách'), findsOneWidget);
    expect(find.text('Địa chỉ giao hàng'), findsNothing);

    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('home-content-scroll')), findsOneWidget);
  });

  testWidgets('Categories tab is a page with a category rail', (tester) async {
    await tester.pumpWidget(const OmaApp());
    await tester.pumpAndSettle();

    await tester.tap(find.text('Danh mục'));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('categories-rail')), findsOneWidget);
    expect(find.byType(BottomSheet), findsNothing);
  });

  testWidgets('Android back from root Cart returns Home', (tester) async {
    await tester.pumpWidget(const OmaApp(initialLocation: AppRoutes.cart));
    await tester.pumpAndSettle();
    expect(find.text('Giỏ hàng trống'), findsOneWidget);

    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('home-content-scroll')), findsOneWidget);
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

    expect(find.byKey(const Key('home-content-scroll')), findsOneWidget);
    expect(find.text('Trang chủ'), findsOneWidget);
    expect(find.text('Danh mục'), findsOneWidget);
    expect(find.text('Giỏ hàng'), findsOneWidget);
    expect(find.text('Thông báo'), findsOneWidget);
    expect(find.text('Tài khoản'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
