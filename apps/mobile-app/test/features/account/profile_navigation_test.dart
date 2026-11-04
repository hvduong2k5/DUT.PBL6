import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/features/auth/data/local_auth_repository.dart';
import 'package:mobile_app/features/auth/data/local_password_hasher.dart';

// Real seeded verifier/KDF is covered separately; this tests the profile form.
class _FixtureHasher implements LocalPasswordHasher {
  @override
  Future<LocalPasswordVerifier> hash(String password) =>
      throw UnimplementedError();
  @override
  Future<bool> verify(String password, LocalPasswordVerifier verifier) async =>
      password == LocalAuthRepository.fixturePassword;
}

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  testWidgets(
    'guest sign-in returns to profile and saves valid contact changes',
    (tester) async {
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/account/profile',
          dependencies: RepositoryFactory.create(
            mode: DataSourceMode.mock,
            localPasswordHasher: _FixtureHasher(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Đăng nhập để tiếp tục'), findsOneWidget);
      await tester.tap(find.text('Đăng nhập'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).at(0), '0905123456');
      await tester.enterText(find.byType(TextField).at(1), 'local-fixture');
      await tester.ensureVisible(find.text('Đăng nhập').last);
      await tester.tap(find.text('Đăng nhập').last);
      await tester.pumpAndSettle();
      expect(find.text('Hồ sơ cá nhân'), findsOneWidget);
      await tester.enterText(
        find.widgetWithText(TextField, 'Họ và tên'),
        'Nguyễn An Tri Kỷ',
      );
      await tester.enterText(
        find.widgetWithText(TextField, 'Email liên hệ'),
        'new@example.com',
      );
      await tester.ensureVisible(find.text('Lưu thay đổi'));
      await tester.tap(find.text('Lưu thay đổi'));
      await tester.pumpAndSettle();
      expect(find.text('Hồ sơ Tri Kỷ đã được cập nhật.'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
