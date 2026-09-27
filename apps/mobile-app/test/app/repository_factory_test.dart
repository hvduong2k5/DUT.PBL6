import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/features/account/data/in_memory_address_repository.dart';
import 'package:mobile_app/features/account/data/rest_address_repository.dart';
import 'package:mobile_app/features/auth/data/mock_auth_repository.dart';
import 'package:mobile_app/features/auth/data/rest_auth_repository.dart';
import 'package:mobile_app/features/cart/data/in_memory_cart_repository.dart';
import 'package:mobile_app/features/cart/data/rest_cart_repository.dart';
import 'package:mobile_app/features/catalog/data/mock_catalog_repository.dart';
import 'package:mobile_app/features/catalog/data/rest_catalog_repository.dart';
import 'package:mobile_app/features/checkout/data/mock_checkout_repository.dart';
import 'package:mobile_app/features/checkout/data/rest_checkout_repository.dart';
import 'package:mobile_app/features/notifications/data/local_notifications_repository.dart';
import 'package:mobile_app/features/orders/data/mock_orders_repository.dart';
import 'package:mobile_app/features/orders/data/rest_orders_repository.dart';

void main() {
  test('mock mode resolves only local implementations', () {
    final dependencies = RepositoryFactory.create(mode: DataSourceMode.mock);
    expect(dependencies.catalogRepository, isA<MockCatalogRepository>());
    expect(dependencies.cartRepository, isA<InMemoryCartRepository>());
    expect(dependencies.checkoutRepository, isA<MockCheckoutRepository>());
    expect(dependencies.ordersRepository, isA<MockOrdersRepository>());
    expect(dependencies.addressRepository, isA<InMemoryAddressRepository>());
    expect(dependencies.authRepository, isA<MockAuthRepository>());
    expect(
      dependencies.notificationsRepository,
      isA<LocalNotificationsRepository>(),
    );
  });

  for (final mode in [DataSourceMode.mockoon, DataSourceMode.backend]) {
    test('$mode resolves the shared REST stack where supported', () {
      final dependencies = RepositoryFactory.create(
        mode: mode,
        apiBaseUrl: 'http://localhost:4010',
      );
      expect(dependencies.catalogRepository, isA<RestCatalogRepository>());
      expect(dependencies.cartRepository, isA<RestCartRepository>());
      expect(dependencies.ordersRepository, isA<RestOrdersRepository>());
      expect(dependencies.addressRepository, isA<RestAddressRepository>());
      expect(dependencies.authRepository, isA<RestAuthRepository>());
      expect(dependencies.checkoutRepository, isA<RestCheckoutRepository>());
      expect(
        dependencies.notificationsRepository,
        isA<LocalNotificationsRepository>(),
      );
    });
  }
}
