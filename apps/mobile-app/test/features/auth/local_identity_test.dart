import 'dart:async';
import 'dart:convert';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/data/local_account_store.dart';
import 'package:mobile_app/features/auth/data/local_auth_repository.dart';
import 'package:mobile_app/features/auth/data/local_password_hasher.dart';
import 'package:mobile_app/features/auth/data/mock_auth_repository.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';

const _passwordA = 'AlphaFixture!2026', _passwordB = 'BetaFixture!2026';
Future<AuthSession> _register(LocalAuthRepository r, {bool second = false}) =>
    r.register(
      phoneNumber: second ? '0901110002' : '0901110001',
      password: second ? _passwordB : _passwordA,
      fullName: second ? 'Local B' : 'Local A',
      email: second ? 'b.fixture@example.invalid' : 'a.fixture@example.invalid',
    );

class _FailStore extends MemoryLocalAccountStore {
  bool fail = false;
  @override
  Future<void> write(String value) async {
    if (fail) throw StateError('Storage unavailable');
    return super.write(value);
  }
}

class _Cart extends Mock implements CartRepository {}

void main() {
  test(
    'Android vault has isolated keys and never enables destructive error recovery',
    () {
      final options = const SecureLocalAccountStore().storage.aOptions.toMap();
      expect(options['storageNamespace'], 'oma_local_identity_v1');
      expect(options['resetOnError'], 'false');
    },
  );
  test(
    'old cart mutation cannot publish into a new identity or leave it pending',
    () async {
      final repo = _Cart();
      var owner = 'a';
      when(repo.getCart).thenAnswer((_) async => const Cart());
      final pending = Completer<Cart>();
      when(() => repo.update('line-a', 2)).thenAnswer((_) => pending.future);
      final c = CartCubit(repo, principal: () => owner);
      await c.load();
      final update = c.update('line-a', 2);
      owner = 'b';
      c.identityChanged();
      await Future<void>.delayed(Duration.zero);
      pending.complete(const Cart(id: 'mutation-from-a'));
      expect(await update, isFalse);
      expect(c.state.cart.id, isNot('mutation-from-a'));
      expect(c.state.pending, isFalse);
      expect(c.state.message, isNull);
      await c.close();
    },
  );
  test(
    'vault queue preserves caller zone and advances after a failed transaction',
    () async {
      final lock = LocalAccountLock(), gate = Completer<void>();
      final seen = <String>[];
      final a = Zone.current
          .fork(zoneValues: {'actor': 'a'})
          .run(
            () => lock.run(() async {
              seen.add(Zone.current['actor'] as String);
              await gate.future;
              throw StateError('Fail transaction');
            }),
          );
      final b = Zone.current
          .fork(zoneValues: {'actor': 'b'})
          .run(
            () => lock.run(() async {
              seen.add(Zone.current['actor'] as String);
              return 2;
            }),
          );
      final failed = expectLater(a, throwsStateError);
      expect(seen, ['a']);
      gate.complete();
      await failed;
      expect(await b, 2);
      expect(seen, ['a', 'b']);
    },
  );
  test(
    'failed logout is reported and retry commits durable logout without deleting accounts',
    () async {
      final store = _FailStore();
      final auth = LocalAuthRepository(store: store);
      await _register(auth);
      store.fail = true;
      await expectLater(auth.logout(), throwsA(isA<ApiException>()));
      expect(auth.currentUser, isNull);
      store.fail = false;
      await auth.logout();
      expect(await LocalAuthRepository(store: store).restoreSession(), isNull);
      expect(jsonDecode(store.value!)['accounts'].length, 3);
    },
  );
  test(
    'profile email uniqueness is case-insensitive and failed edits preserve the stored account',
    () async {
      final store = MemoryLocalAccountStore();
      final auth = LocalAuthRepository(store: store);
      final a = await _register(auth);
      await _register(auth, second: true);
      await auth.logout();
      await auth.login(phoneNumber: a.user.phoneNumber, password: _passwordA);
      final previous = store.value;
      await expectLater(
        auth.updateUser(
          AuthUser(
            id: a.user.id,
            phoneNumber: a.user.phoneNumber,
            fullName: a.user.fullName,
            email: 'B.FIXTURE@EXAMPLE.INVALID',
          ),
        ),
        throwsA(isA<ApiException>()),
      );
      expect(store.value, previous);
      expect((await auth.restoreSession())!.email, a.user.email);
    },
  );
  test(
    'real PBKDF2 salts differ, correct password verifies and wrong one fails',
    () async {
      const h = Pbkdf2LocalPasswordHasher();
      final a = await h.hash(_passwordA), b = await h.hash(_passwordA);
      expect(a.salt, isNot(b.salt));
      expect(a.digest, isNot(b.digest));
      expect(await h.verify(_passwordA, a), isTrue);
      expect(await h.verify(_passwordB, a), isFalse);
      expect(a.toJson().toString(), isNot(contains(_passwordA)));
    },
  );
  test(
    'register has UUID, canonical phone and persistent profile/session; secure store has no plaintext',
    () async {
      final r = LocalAuthRepository();
      final first = await _register(r);
      expect(first.user.id, isNot('CUST-MOCK'));
      final raw = await const FlutterSecureStorage().read(
        key: SecureLocalAccountStore.key,
      );
      expect(raw, isNot(contains(_passwordA)));
      expect(jsonDecode(raw!)['accounts'].length, 3);
      final fresh = LocalAuthRepository();
      expect((await fresh.restoreSession())!.id, first.user.id);
      await fresh.logout();
      final logged = await fresh.login(
        phoneNumber: '+84901110001',
        password: _passwordA,
      );
      expect(logged.user, first.user);
      await fresh.updateUser(
        AuthUser(
          id: first.user.id,
          phoneNumber: first.user.phoneNumber,
          fullName: 'Local A Updated',
          email: first.user.email,
        ),
      );
      expect(
        (await LocalAuthRepository().restoreSession())!.fullName,
        'Local A Updated',
      );
    },
  );
  test(
    'duplicate phone canonicalization/email and concurrent register never create shared identities',
    () async {
      final store = MemoryLocalAccountStore();
      final a = LocalAuthRepository(store: store),
          b = LocalAuthRepository(store: store);
      final results = await Future.wait([
        _register(a).then<Object>((s) => s, onError: (Object e) => e),
        _register(b).then<Object>((s) => s, onError: (Object e) => e),
      ]);
      expect(results.whereType<AuthSession>().length, 1);
      expect(results.whereType<ApiException>().length, 1);
      await expectLater(
        a.register(
          phoneNumber: '+84901110001',
          password: _passwordA,
          fullName: 'Duplicate',
          email: 'different@example.invalid',
        ),
        throwsA(isA<ApiException>()),
      );
      await expectLater(
        a.register(
          phoneNumber: '0901110003',
          password: _passwordB,
          fullName: 'Duplicate',
          email: 'A.FIXTURE@EXAMPLE.INVALID',
        ),
        throwsA(isA<ApiException>()),
      );
      final second = await _register(b, second: true);
      expect(
        second.user.id,
        isNot((results.whereType<AuthSession>().single).user.id),
      );
      expect(jsonDecode(store.value!)['accounts'].length, 4);
    },
  );
  test(
    'wrong and unknown credentials denied, successful logout and expiry survive recreation',
    () async {
      var now = DateTime.utc(2026, 10, 31);
      final store = MemoryLocalAccountStore();
      final r = LocalAuthRepository(store: store, now: () => now);
      await _register(r);
      await r.logout();
      await expectLater(
        r.login(phoneNumber: '0901110001', password: _passwordB),
        throwsA(isA<ApiException>()),
      );
      expect(r.currentUser, isNull);
      expect(await LocalAuthRepository(store: store).restoreSession(), isNull);
      await expectLater(
        r.login(phoneNumber: '0901110099', password: _passwordA),
        throwsA(isA<ApiException>()),
      );
      await r.login(phoneNumber: '0901110001', password: _passwordA);
      now = now.add(const Duration(days: 8));
      expect(
        await LocalAuthRepository(
          store: store,
          now: () => now,
        ).restoreSession(),
        isNull,
      );
      expect(jsonDecode(store.value!)['accounts'].length, 3);
    },
  );
  test(
    'failed commits do not publish a new identity; corrupt data is retained and fails closed',
    () async {
      final store = _FailStore();
      final auth = LocalAuthRepository(store: store);
      await auth.restoreSession();
      final previous = store.value;
      store.fail = true;
      await expectLater(_register(auth), throwsStateError);
      expect(auth.currentUser, isNull);
      expect(store.value, previous);
      store.fail = false;
      store.value = '{invalid';
      await expectLater(auth.restoreSession(), throwsFormatException);
      await expectLater(
        auth.login(
          phoneNumber: '0905123456',
          password: LocalAuthRepository.fixturePassword,
        ),
        throwsFormatException,
      );
      expect(store.value, '{invalid');
      expect(auth.currentUser, isNull);
    },
  );
  test(
    'reserved fixture IDs require canonical credential; registration cannot gain B2B or mock recovery authority',
    () async {
      final r = LocalAuthRepository(store: MemoryLocalAccountStore());
      final b2b = await r.login(
        phoneNumber: MockAuthRepository.b2bFixturePhone,
        password: LocalAuthRepository.fixturePassword,
      );
      expect(b2b.user.id, MockAuthRepository.b2bFixtureId);
      await r.logout();
      await expectLater(
        r.register(
          phoneNumber: MockAuthRepository.b2bFixturePhone,
          password: _passwordA,
          fullName: 'Spoof',
          email: 'spoof@example.invalid',
        ),
        throwsA(isA<ApiException>()),
      );
      await expectLater(
        r.verifyRecovery('mock-token', idempotencyKey: 'fixture-key'),
        throwsA(isA<ApiException>()),
      );
      await expectLater(
        r.login(phoneNumber: '0905123456', password: 'arbitrary'),
        throwsA(isA<ApiException>()),
      );
    },
  );
  test(
    'two identities isolate cart, addresses, legacy/MVP order and operation data; guest cart only adopted once',
    () async {
      final d = RepositoryFactory.create(mode: DataSourceMode.mock);
      final auth = d.authRepository as LocalAuthRepository;
      final product = await d.catalogRepository.product('PROD-MX-GION-500');
      await d.cartRepository.add(product, product.variants.first, 1);
      final a = await _register(auth);
      expect((await d.cartRepository.getCart()).totalItems, 1);
      await d.addressRepository.save(
        const Address(
          id: 'a-address',
          recipientName: 'A',
          phoneNumber: '0901110001',
          streetAddress: 'A street',
          ward: 'A ward',
          province: 'A province',
          isDefault: true,
        ),
      );
      final cart = await d.mvpPurchaseRepository!.cart(),
          addresses = await d.mvpPurchaseRepository!.savedAddresses();
      final quote = await d.mvpPurchaseRepository!.quote(
        cartRevision: cart.revision,
        addressId: addresses.first.id,
      );
      final operation = await d.mvpCheckoutRuntime!.repository.accept(
        MvpCheckoutInput.fromQuote(quote),
        key: 'fixture-isolation-key',
      );
      await d.mvpCheckoutRuntime!.repository.operation(operation.id);
      final complete = await d.mvpCheckoutRuntime!.repository.operation(
        operation.id,
      );
      await auth.logout();
      expect((await d.cartRepository.getCart()).items, isEmpty);
      final b = await _register(auth, second: true);
      expect(a.user.id, isNot(b.user.id));
      expect((await d.cartRepository.getCart()).items, isEmpty);
      expect(await d.addressRepository.addresses(), isEmpty);
      expect(await d.ordersRepository.orders(), isEmpty);
      expect((await d.mvpOrdersRuntime!.repository.firstPage()).items, isEmpty);
      await expectLater(
        d.mvpOrdersRuntime!.repository.order(complete.orderId!),
        throwsA(isA<ApiException>()),
      );
      await expectLater(
        d.mvpCheckoutRuntime!.repository.operation(operation.id),
        throwsA(isA<ApiException>()),
      );
      await d.cartRepository.add(product, product.variants.first, 2);
      await auth.logout();
      await auth.login(phoneNumber: '0901110001', password: _passwordA);
      expect((await d.cartRepository.getCart()).totalItems, 1);
      expect((await d.addressRepository.addresses()).single.id, 'a-address');
      expect(
        (await d.mvpOrdersRuntime!.repository.firstPage()).items.single.id,
        complete.orderId,
      );
    },
  );
  test(
    'old cart request cannot replace new identity state after a switch',
    () async {
      final repo = _Cart();
      var owner = 'a';
      final pending = Completer<Cart>();
      when(repo.getCart).thenAnswer((_) => pending.future);
      final c = CartCubit(repo, principal: () => owner);
      final old = c.load();
      owner = 'b';
      when(repo.getCart).thenAnswer((_) async => const Cart());
      c.identityChanged();
      await Future<void>.delayed(Duration.zero);
      pending.complete(const Cart(id: 'old-owner-a'));
      await old;
      expect(c.state.cart.id, isNot('old-owner-a'));
      expect(c.state.pending, isFalse);
      await c.close();
    },
  );
}
