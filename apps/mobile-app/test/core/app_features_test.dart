import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/core/config/app_features.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  tearDown(() => AppFeatures.current = const AppFeatures.all());

  test('mock keeps every feature on; real data only what is defined', () {
    final mock = AppFeatures.resolve(DataSourceMode.mock);
    expect(AppFeature.values.every(mock.enabled), isTrue);

    final real = AppFeatures.resolve(
      DataSourceMode.backend,
      defines: {AppFeature.support: true, AppFeature.trace: false},
    );
    expect(real.enabled(AppFeature.support), isTrue);
    expect(real.enabled(AppFeature.trace), isFalse);
    expect(real.enabled(AppFeature.promotions), isFalse);
  });

  test('gated routes map to their feature', () {
    expect(AppFeatures.forPath('/promotions'), AppFeature.promotions);
    expect(AppFeatures.forPath('/account/loyalty'), AppFeature.loyalty);
    expect(AppFeatures.forPath('/after-sales/R1'), AppFeature.returns);
    expect(
      AppFeatures.forPath('/orders/O1/after-sales/request'),
      AppFeature.returns,
    );
    expect(AppFeatures.forPath('/support/tickets/T1'), AppFeature.support);
    expect(AppFeatures.forPath('/trace/QR'), AppFeature.trace);
    expect(AppFeatures.forPath('/b2b/quotes/Q'), AppFeature.b2b);
    expect(AppFeatures.forPath('/track-order'), AppFeature.guestOrderLookup);
    expect(AppFeatures.forPath('/orders/O1'), isNull);
    expect(AppFeatures.forPath('/catalog'), isNull);
    expect(AppFeatures.forPath('/tracery'), isNull);
  });

  testWidgets(
    'a disabled feature shows the unavailable page and hides entries',
    (tester) async {
      AppFeatures.current = const AppFeatures.only({});
      await tester.pumpWidget(const OmaApp(initialLocation: '/promotions'));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('feature-unavailable')), findsOneWidget);
      expect(find.text('Tính năng chưa khả dụng'), findsOneWidget);

      await tester.tap(find.text('Về trang chủ'));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('home-promotions')), findsNothing);
      await tester.tap(find.text('Tài khoản'));
      await tester.pumpAndSettle();
      expect(find.text('Ưu đãi Tri Kỷ'), findsNothing);
      expect(find.text('Truy xuất nguồn gốc'), findsNothing);
      expect(find.text('MUA SẮM & HỖ TRỢ'), findsNothing);
    },
  );
}
