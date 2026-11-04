import 'dart:convert';
import 'dart:io';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/features/auth/data/identity_auth_repository.dart';
import 'package:mobile_app/features/auth/data/identity_purchase_guards.dart';
import 'package:mobile_app/features/auth/data/identity_session_store.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/cart/data/mvp_cart_repository.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';
import 'package:mobile_app/features/orders/domain/mvp_order_models.dart';
import 'package:uuid/uuid.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  final path = Platform.environment['MVP_LIVE_CONFIG'];
  test(
    'native repositories use real Identity, Catalog, Profile, Order and Inventory',
    () async {
      HttpOverrides.global = null;
      FlutterSecureStorage.setMockInitialValues({});
      final config = Map<String, dynamic>.from(
        jsonDecode(await File(path!).readAsString()) as Map,
      );
      final store = MemoryIdentitySessionStore();
      await store.write(
        IdentitySessionRecord.fromJson(
          Map<String, dynamic>.from(config['session'] as Map),
        ),
      );
      final dependencies = RepositoryFactory.create(
        mode: DataSourceMode.backend,
        apiBaseUrl: config['gateway'] as String,
        identityAuth: true,
        identityBaseUrl: '${config['gateway']}/api/v1',
        identityStore: store,
        identityAuthorizationOrigins: {
          Uri.parse(config['issuer'] as String).origin,
        },
      );
      final auth = dependencies.authRepository as IdentityAuthRepository;
      final user = await auth.restoreSession();
      expect(user?.canPurchase, isTrue);
      expect(dependencies.mvpCheckoutRuntime?.local, isFalse);
      expect(dependencies.mvpOrdersRuntime?.local, isFalse);
      final cart = dependencies.cartRepository;
      final client =
          ((cart as IdentityCartGuard).inner as MvpCartRepository).client;
      // Functional tests share the Gateway's normal rate budget with Web fan-out.
      client.dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (request, handler) async {
            await Future<void>.delayed(const Duration(seconds: 1));
            handler.next(request);
          },
          onError: (error, handler) {
            stdout.writeln(
              'Live API failure: ${error.requestOptions.method} ${error.requestOptions.uri.path} HTTP ${error.response?.statusCode} (${error.type.name})',
            );
            handler.next(error);
          },
        ),
      );
      final locations = await dependencies.checkoutRepository.locations();
      expect(locations, isNotEmpty);
      final product = await dependencies.catalogRepository.product(
        'test-product',
      );
      expect(product.variants.single.sku, 'PEER');
      final search = await dependencies.catalogRepository.productPage(
        query: 'Synthetic',
        pageSize: 5,
      );
      expect(search.items.any((item) => item.id == product.id), isTrue);
      final addresses = dependencies.addressRepository;
      final before = await addresses.addresses();
      expect(before, isNotEmpty);
      final location = locations.first;
      await addresses.save(
        Address(
          id: 'native-client-draft',
          recipientName: 'Native live fixture',
          phoneNumber: '0900000001',
          streetAddress: 'Native fixture street',
          province: location.name,
          ward: location.wards.first.name,
          provinceCode: location.code,
          wardCode: location.wards.first.code,
        ),
      );
      final created = (await addresses.addresses()).firstWhere(
        (item) => item.recipientName == 'Native live fixture',
      );
      expect(created.id, isNot('native-client-draft'));
      await addresses.save(created.copyWith(isDefault: true));
      expect(
        (await addresses.addresses())
            .firstWhere((item) => item.id == created.id)
            .isDefault,
        isTrue,
      );
      await addresses.delete(created.id);
      expect(
        (await addresses.addresses()).any((item) => item.id == created.id),
        isFalse,
      );
      await cart.clear();
      final added = await cart.add(product, product.variants.single, 1);
      expect(added.totalItems, 1);
      expect(added.subtotal.units, BigInt.from(1000));
      final doubled = await cart.update(added.items.single.itemId, 2);
      expect(doubled.subtotal.units, BigInt.from(2000));
      expect((await cart.remove(doubled.items.single.itemId)).items, isEmpty);
      await cart.add(product, product.variants.single, 1);
      final purchase = dependencies.mvpPurchaseRepository!;
      final selection = await purchase.cart();
      final saved = (await purchase.savedAddresses()).firstWhere(
        (address) => address.isDefault,
      );
      final quote = await purchase.quote(
        cartRevision: selection.revision,
        addressId: saved.id,
      );
      expect(quote.paymentMethod, 'VIETQR');
      final checkout = dependencies.mvpCheckoutRuntime!.repository;
      final input = MvpCheckoutInput.fromQuote(quote);
      final key = const Uuid().v4();
      var operation = await checkout.accept(input, key: key);
      final replay = await checkout.accept(input, key: key);
      expect(replay.id, operation.id);
      for (
        var attempt = 0;
        attempt < 40 && operation.status != 'SUCCEEDED';
        attempt++
      ) {
        await Future<void>.delayed(const Duration(milliseconds: 200));
        operation = await checkout.operation(operation.id);
      }
      expect(operation.status, 'SUCCEEDED');
      final orders = dependencies.mvpOrdersRuntime!;
      final order = await orders.repository.order(operation.orderId!);
      expect(order.total.units, quote.total.units);
      expect(
        (await orders.repository.firstPage()).items.any(
          (item) => item.id == order.id,
        ),
        isTrue,
      );
      expect((await orders.repository.payment(order.id)).orderId, order.id);
      await orders.repository.instructions(order.id);
      await orders.repository.cancel(
        MvpCancellationCommand(
          owner: user!.id,
          scope: orders.scope,
          orderId: order.id,
          key: const Uuid().v4(),
          reason: 'Native TEST cancellation',
          version: order.version,
        ),
      );
      var cancelled = await orders.repository.order(order.id);
      for (
        var attempt = 0;
        attempt < 40 && cancelled.status != 'CANCELLED_BY_USER';
        attempt++
      ) {
        await Future<void>.delayed(const Duration(milliseconds: 200));
        cancelled = await orders.repository.order(order.id);
      }
      expect(cancelled.status, 'CANCELLED_BY_USER');
      await auth.logout();
    },
    skip: path == null,
    timeout: const Timeout(Duration(minutes: 3)),
  );
}
