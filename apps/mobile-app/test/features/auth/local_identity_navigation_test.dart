import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/features/auth/presentation/auth_page.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';
import 'package:mobile_app/features/auth/data/local_password_hasher.dart';

// Form-state tests use a deterministic verifier; real PBKDF2 has separate coverage.
class _FormHasher implements LocalPasswordHasher {
  String _digest(String password) {
    final sum = password.codeUnits.fold<int>(0, (a, b) => a + b);
    return base64Encode(List<int>.generate(32, (i) => (sum + i) % 256));
  }

  @override
  Future<LocalPasswordVerifier> hash(String password) async =>
      LocalPasswordVerifier(
        salt: base64Encode(List<int>.filled(16, 0)),
        digest: _digest(password),
      );
  @override
  Future<bool> verify(String password, LocalPasswordVerifier verifier) async =>
      verifier.digest == _digest(password);
}

Finder _field(String label) => find.byWidgetPredicate(
  (w) => w is TextField && w.decoration?.labelText == label,
);
Future<void> _enter(WidgetTester tester, String label, String value) async {
  await tester.ensureVisible(_field(label));
  await tester.enterText(_field(label), value);
}

Future<void> _submit(
  WidgetTester tester,
  String label,
  AuthStatus expected,
) async {
  final context = tester.element(find.byType(AuthPage));
  final auth = context.read<AuthCubit>();
  await tester.ensureVisible(find.text(label).last);
  await tester.tap(find.text(label).last);
  await tester.pumpAndSettle();
  expect(auth.state.status, expected);
}

void main() {
  testWidgets(
    'register form persists identity; restart restores it; wrong login fails and exact password succeeds',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/register?returnUrl=%2Faccount',
          dependencies: RepositoryFactory.create(
            mode: DataSourceMode.mock,
            localPasswordHasher: _FormHasher(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await _enter(tester, 'Họ và tên', 'UI Local Account');
      await _enter(tester, 'Email khôi phục', 'ui.fixture@example.invalid');
      await _enter(tester, 'Số điện thoại', '0901110099');
      await _enter(tester, 'Mật khẩu', 'UiFixturePass!2026');
      await _enter(tester, 'Xác nhận mật khẩu', 'UiFixturePass!2026');
      await tester.ensureVisible(find.byType(Checkbox));
      await tester.tap(find.byType(Checkbox));
      await tester.pump();
      await _submit(tester, 'Đăng ký', AuthStatus.authenticated);
      expect(find.text('UI Local Account'), findsWidgets);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/account',
          dependencies: RepositoryFactory.create(
            mode: DataSourceMode.mock,
            localPasswordHasher: _FormHasher(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('UI Local Account'), findsWidgets);
      await tester.scrollUntilVisible(
        find.text('Đăng xuất'),
        250,
        scrollable: find.byType(Scrollable).first,
      );
      final logoutButton = find.widgetWithText(OutlinedButton, 'Đăng xuất');
      await tester.ensureVisible(logoutButton);
      await tester.pumpAndSettle();
      expect(logoutButton.hitTestable(), findsOneWidget);
      await tester.tap(logoutButton);
      await tester.pumpAndSettle();
      expect(find.text('UI Local Account'), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/login?returnUrl=%2Faccount',
          dependencies: RepositoryFactory.create(
            mode: DataSourceMode.mock,
            localPasswordHasher: _FormHasher(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await _enter(tester, 'Số điện thoại', '0901110099');
      await _enter(tester, 'Mật khẩu', 'WrongFixture!2026');
      await _submit(tester, 'Đăng nhập', AuthStatus.failure);
      expect(
        find.text('Số điện thoại hoặc mật khẩu không đúng.'),
        findsOneWidget,
      );
      await _enter(tester, 'Mật khẩu', 'UiFixturePass!2026');
      await _submit(tester, 'Đăng nhập', AuthStatus.authenticated);
      expect(find.text('UI Local Account'), findsWidgets);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
}
