import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));

  testWidgets(
    'recovery is reachable from login and returns with checkout intent',
    (tester) async {
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/login?returnUrl=%2Fcheckout',
          dependencies: RepositoryFactory.create(
            mode: DataSourceMode.mock,
            localMvpCheckout: false,
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('Quên mật khẩu?'));
      await tester.pumpAndSettle();
      expect(find.text('Email đã đăng ký'), findsOneWidget);
      await tester.enterText(
        find.byType(TextField).first,
        'unknown@example.com',
      );
      await tester.ensureVisible(find.text('Gửi liên kết khôi phục'));
      await tester.tap(find.text('Gửi liên kết khôi phục'));
      await tester.pumpAndSettle();
      expect(find.text('Hãy kiểm tra email'), findsOneWidget);
      expect(find.textContaining('không xác nhận tài khoản'), findsOneWidget);
      await tester.ensureVisible(find.text('Quay lại đăng nhập Tri Kỷ'));
      await tester.tap(find.text('Quay lại đăng nhập Tri Kỷ'));
      await tester.pumpAndSettle();
      expect(
        find.text('Tiếp tục thanh toán với tư cách khách'),
        findsOneWidget,
      );
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'recovery link resets and returns to normal login without authentication',
    (tester) async {
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/account-recovery?recoveryToken=mock-token',
          dependencies: RepositoryFactory.create(
            mode: DataSourceMode.mock,
            localMvpCheckout: false,
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Tạo mật khẩu mới'), findsOneWidget);
      await tester.enterText(find.byType(TextField).at(0), 'TriKyMatKhau!2026');
      await tester.enterText(find.byType(TextField).at(1), 'TriKyMatKhau!2026');
      await tester.ensureVisible(find.text('Đặt lại mật khẩu'));
      await tester.tap(find.text('Đặt lại mật khẩu'));
      await tester.pumpAndSettle();
      expect(find.text('Mật khẩu đã được cập nhật'), findsOneWidget);
      await tester.tap(find.text('Quay lại đăng nhập'));
      await tester.pumpAndSettle();
      expect(find.text('Quên mật khẩu?'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'recovery reset scrolls with keyboard on narrow large-text viewport',
    (tester) async {
      tester.view.physicalSize = const Size(320, 640);
      tester.view.devicePixelRatio = 1;
      tester.view.viewInsets = const FakeViewPadding(bottom: 280);
      tester.platformDispatcher.textScaleFactorTestValue = 1.4;
      addTearDown(tester.view.reset);
      addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/account-recovery?recoveryToken=mock-token',
          dependencies: RepositoryFactory.create(
            mode: DataSourceMode.mock,
            localMvpCheckout: false,
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.text('Đặt lại mật khẩu'),
        120,
        scrollable: find.byType(Scrollable).first,
      );
      expect(tester.takeException(), isNull);
    },
  );
}
