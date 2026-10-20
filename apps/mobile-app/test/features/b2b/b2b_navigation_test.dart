import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/features/auth/data/mock_auth_repository.dart';

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  testWidgets('registered actor cannot open verified B2B workspace', (
    tester,
  ) async {
    final dependencies = RepositoryFactory.create(mode: DataSourceMode.mock);
    await tester.runAsync(() async {
      await dependencies.authRepository.login(
        phoneNumber: '0905123456',
        password: 'legacy',
      );
    });
    await tester.pumpWidget(
      OmaApp(initialLocation: '/b2b', dependencies: dependencies),
    );
    await tester.pumpAndSettle();
    expect(find.text('Chưa có quyền truy cập'), findsOneWidget);
    expect(find.text('Doanh nghiệp thử nghiệm local'), findsNothing);
  });
  testWidgets(
    'verified phone UI requests quote and accepts released quote without paid success',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final dependencies = RepositoryFactory.create(mode: DataSourceMode.mock);
      await tester.runAsync(() async {
        await dependencies.authRepository.login(
          phoneNumber: MockAuthRepository.b2bFixturePhone,
          password: 'legacy',
        );
      });
      await tester.pumpWidget(
        OmaApp(initialLocation: '/b2b', dependencies: dependencies),
      );
      await tester.pumpAndSettle();
      Future<void> reveal(Finder target) async {
        await tester.scrollUntilVisible(
          target,
          300,
          scrollable: find.byType(Scrollable).first,
        );
        await Scrollable.ensureVisible(tester.element(target), alignment: 0.3);
        await tester.pumpAndSettle();
      }

      final dateField = find.byWidgetPredicate(
        (w) => w is TextField && w.decoration?.labelText == 'Ngày nhận dự kiến',
      );
      await reveal(dateField);
      await tester.enterText(
        dateField,
        DateTime.now()
            .add(const Duration(days: 14))
            .toIso8601String()
            .substring(0, 10),
      );
      await reveal(find.text('Gửi yêu cầu báo giá B2B'));
      await tester.tap(find.text('Gửi yêu cầu báo giá B2B'));
      await tester.pumpAndSettle();
      await tester.drag(find.byType(Scrollable).first, const Offset(0, 1600));
      await tester.pumpAndSettle();
      expect(find.textContaining('Đã tiếp nhận YC-LOCAL'), findsOneWidget);
      await reveal(find.text('Xem báo giá'));
      await tester.tap(find.text('Xem báo giá'));
      await tester.pumpAndSettle();
      expect(find.text('Bảng báo giá quà biếu'), findsOneWidget);
      await reveal(find.byType(Checkbox));
      await tester.tap(find.byType(Checkbox));
      await tester.pumpAndSettle();
      final accept = find.widgetWithText(
        AppPrimaryButton,
        'Chấp thuận báo giá',
      );
      await reveal(accept);
      await tester.tap(accept);
      await tester.pumpAndSettle();
      await tester.drag(find.byType(Scrollable).first, const Offset(0, 4000));
      await tester.pumpAndSettle();
      expect(find.text('Đã ghi nhận chấp thuận'), findsOneWidget);
      expect(
        find.text('Chấp thuận không đồng nghĩa đã đặt cọc hoặc thanh toán.'),
        findsOneWidget,
      );
      expect(tester.takeException(), isNull);
    },
  );
}
