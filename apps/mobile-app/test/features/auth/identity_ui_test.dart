import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/data/identity_purchase_guards.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/auth/domain/identity_persona.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_repository.dart';
import 'identity_auth_repository_test.dart' show Fixture, session;
import 'identity_foundations_test.dart' show profile;

AppDependencies identityDependencies(AuthRepository auth) {
  final d = RepositoryFactory.create(mode: DataSourceMode.mock);
  return AppDependencies(
    catalogRepository: d.catalogRepository,
    cartRepository: d.cartRepository,
    checkoutRepository: d.checkoutRepository,
    ordersRepository: d.ordersRepository,
    addressRepository: d.addressRepository,
    authRepository: auth,
    notificationsRepository: d.notificationsRepository,
    promotionsRepository: d.promotionsRepository,
    customerAccountRepository: d.customerAccountRepository,
    guestOrdersRepository: d.guestOrdersRepository,
    supportRepository: d.supportRepository,
    capabilitiesRepository: d.capabilitiesRepository,
    traceRepository: d.traceRepository,
    returnsRepository: d.returnsRepository,
    b2bRepository: d.b2bRepository,
    reviewsRepository: d.reviewsRepository,
  );
}

class _Cart extends Mock implements CartRepository {}

class _Checkout extends Mock implements CheckoutRepository {}

Finder field(String label) => find.byWidgetPredicate(
  (w) => w is TextField && w.decoration?.labelText == label,
);

void phoneViewport(WidgetTester tester) {
  tester.view.physicalSize = const Size(393, 852);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
}

Future<void> fillRegistration(WidgetTester tester) async {
  for (final entry in {
    'Họ': 'Customer',
    'Tên': 'Synthetic',
    'Email': 'ui@example.invalid',
    'Mật khẩu': 'SyntheticTest!2026',
    'Xác nhận mật khẩu': 'SyntheticTest!2026',
  }.entries) {
    await tester.ensureVisible(field(entry.key));
    await tester.enterText(field(entry.key), entry.value);
  }
  tester.testTextInput.hide();
  await tester.ensureVisible(find.byType(CheckboxListTile));
  await tester.pumpAndSettle();
  await tester.tap(find.byType(CheckboxListTile));
}

Future<void> submitRegistration(WidgetTester tester) async {
  await tester.scrollUntilVisible(
    find.text('Đăng ký'),
    250,
    scrollable: find.byType(Scrollable).first,
  );
  await tester.tap(find.text('Đăng ký').last);
  await tester.pumpAndSettle();
}

Future<void> registrationTop(WidgetTester tester) async {
  tester
      .state<ScrollableState>(find.byType(Scrollable).first)
      .position
      .jumpTo(0);
  await tester.pumpAndSettle();
}

void main() {
  testWidgets(
    'pending registration retains form and retry key; changed input gets a new key',
    (tester) async {
      phoneViewport(tester);
      final f = Fixture();
      f.overrides['/api/v1/public/customer-registrations'] = {
        'status': 'PENDING',
      };
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/register',
          dependencies: identityDependencies(f.repo),
        ),
      );
      await tester.pumpAndSettle();
      await fillRegistration(tester);
      await submitRegistration(tester);
      final first = f.requests.last;
      expect(f.browserCalls, 0);
      await registrationTop(tester);
      expect(find.textContaining('Đăng ký đang được xử lý'), findsOneWidget);
      expect(find.textContaining('Đã đăng ký thành công'), findsNothing);
      expect(
        tester.widget<TextField>(field('Email')).controller!.text,
        'ui@example.invalid',
      );
      await submitRegistration(tester);
      expect(
        f.requests.where((r) => r.uri.path.endsWith('/customer-registrations')),
        hasLength(2),
      );
      expect(
        f.requests.last.headers['Idempotency-Key'],
        first.headers['Idempotency-Key'],
      );
      expect(f.requests.last.data, first.data);
      expect(f.browserCalls, 0);
      await registrationTop(tester);
      await tester.ensureVisible(field('Email'));
      await tester.enterText(field('Email'), 'changed@example.invalid');
      tester.testTextInput.hide();
      await submitRegistration(tester);
      expect(
        f.requests.where((r) => r.uri.path.endsWith('/customer-registrations')),
        hasLength(3),
      );
      expect(
        f.requests.last.headers['Idempotency-Key'],
        isNot(first.headers['Idempotency-Key']),
      );
      expect((f.requests.last.data as Map)['email'], 'changed@example.invalid');
      expect(f.browserCalls, 0);
      expect(await f.store.read(), isNull);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets(
    'completed retry opens login; cancellation offers login without registering twice',
    (tester) async {
      phoneViewport(tester);
      final f = Fixture();
      f.overrides['/api/v1/public/customer-registrations'] = {
        'status': 'PENDING',
      };
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/register',
          dependencies: identityDependencies(f.repo),
        ),
      );
      await tester.pumpAndSettle();
      await fillRegistration(tester);
      await submitRegistration(tester);
      final key = f.requests.last.headers['Idempotency-Key'];
      f.overrides['/api/v1/public/customer-registrations'] = {
        'status': 'COMPLETED',
      };
      f.browser = (_) async =>
          throw StateError('Synthetic browser cancellation');
      await submitRegistration(tester);
      final registrations = f.requests
          .where((r) => r.uri.path.endsWith('/customer-registrations'))
          .toList();
      expect(registrations, hasLength(2));
      expect(registrations.last.headers['Idempotency-Key'], key);
      expect(f.browserCalls, 1);
      await registrationTop(tester);
      expect(find.textContaining('Đã đăng ký thành công'), findsOneWidget);
      expect(
        find.text('Chưa thể đăng nhập. Vui lòng thử lại.'),
        findsOneWidget,
      );
      expect(field('Mật khẩu'), findsNothing);
      expect(await f.store.read(), isNull);
      f.browser = null;
      await tester.ensureVisible(find.text('Tiếp tục đăng nhập'));
      await tester.tap(find.text('Tiếp tục đăng nhập'));
      await tester.pumpAndSettle();
      expect(f.browserCalls, 2);
      expect(
        f.requests.where((r) => r.uri.path.endsWith('/customer-registrations')),
        hasLength(2),
      );
      expect(find.text('Synthetic Customer'), findsWidgets);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  test(
    'Employee/suspended cannot call cart or checkout writes even without the UI',
    () async {
      final d = RepositoryFactory.create(mode: DataSourceMode.mock);
      final product = await d.catalogRepository.product('PROD-MX-GION-500');
      for (final p in [
        profile(type: 'EMPLOYEE'),
        profile(status: 'SUSPENDED'),
      ]) {
        final user = AuthUser(
          id: 'p',
          phoneNumber: '',
          fullName: 'Synthetic',
          persona: IdentityPersona.fromJson(p),
        );
        final cart = _Cart(), checkout = _Checkout();
        await expectLater(
          IdentityCartGuard(
            cart,
            () => user,
          ).add(product, product.variants.first, 1),
          throwsA(isA<ApiException>()),
        );
        await expectLater(
          IdentityCartGuard(cart, () => user).update('item', 2),
          throwsA(isA<ApiException>()),
        );
        await expectLater(
          IdentityCheckoutGuard(checkout, () => user).submit(
            const CheckoutDraft(
              fullName: 'Synthetic',
              phone: '',
              streetAddress: '',
              ward: '',
              province: '',
              cart: Cart(),
            ),
            idempotencyKey: 'key',
          ),
          throwsA(isA<ApiException>()),
        );
        verifyZeroInteractions(cart);
        verifyZeroInteractions(checkout);
      }
    },
  );
  testWidgets(
    'Identity login has browser handoff, prevents duplicate launch and cancellation can retry',
    (tester) async {
      phoneViewport(tester);
      final f = Fixture(), gate = Completer<Uri>();
      f.browser = (_) => gate.future;
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/login',
          dependencies: identityDependencies(f.repo),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(TextField), findsNothing);
      expect(find.textContaining('trình duyệt bảo mật'), findsWidgets);
      await tester.tap(find.text('Tiếp tục đăng nhập'));
      await tester.pump(const Duration(milliseconds: 50));
      expect(find.text('Đang xử lý...'), findsOneWidget);
      expect(f.browserCalls, 1);
      gate.completeError(StateError('Synthetic cancellation'));
      await tester.pumpAndSettle();
      expect(
        find.text('Chưa thể đăng nhập. Vui lòng thử lại.'),
        findsOneWidget,
      );
      f.browser = null;
      await tester.tap(find.text('Tiếp tục đăng nhập'));
      await tester.pumpAndSettle();
      expect(find.text('Synthetic Customer'), findsWidgets);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  testWidgets(
    'Identity registration has separate name/email inputs and immediately starts browser login',
    (tester) async {
      phoneViewport(tester);
      final f = Fixture();
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/register',
          dependencies: identityDependencies(f.repo),
        ),
      );
      await tester.pumpAndSettle();
      for (final entry in {
        'Họ': 'Customer',
        'Tên': 'Synthetic',
        'Email': 'ui@example.invalid',
        'Mật khẩu': 'SyntheticTest!2026',
        'Xác nhận mật khẩu': 'SyntheticTest!2026',
      }.entries) {
        await tester.ensureVisible(field(entry.key));
        await tester.enterText(field(entry.key), entry.value);
      }
      expect(field('Số điện thoại'), findsNothing);
      tester.testTextInput.hide();
      await tester.ensureVisible(find.byType(CheckboxListTile));
      await tester.pumpAndSettle();
      expect(find.byType(Checkbox).hitTestable(), findsOneWidget);
      await tester.tap(find.byType(CheckboxListTile));
      await tester.ensureVisible(find.text('Đăng ký').last);
      await tester.tap(find.text('Đăng ký').last);
      await tester.pumpAndSettle();
      expect(f.browserCalls, 1);
      expect(find.text('Synthetic Customer'), findsWidgets);
      expect(
        f.requests
            .where((r) => r.uri.path.endsWith('/customer-registrations'))
            .length,
        1,
      );
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  testWidgets(
    'Employee sees purchase restriction and product add button is disabled',
    (tester) async {
      phoneViewport(tester);
      final f = Fixture()..principalType = 'EMPLOYEE';
      await f.store.write(session);
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/product/PROD-MX-GION-500',
          dependencies: identityDependencies(f.repo),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text(employeePurchaseMessage), findsOneWidget);
      final button = tester.widget<FilledButton>(
        find.ancestor(
          of: find.text('Thêm vào giỏ'),
          matching: find.byType(FilledButton),
        ),
      );
      expect(button.onPressed, isNull);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
}
