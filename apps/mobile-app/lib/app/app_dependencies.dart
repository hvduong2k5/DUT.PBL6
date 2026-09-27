import 'package:mobile_app/core/config/app_config.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/storage/token_store.dart';
import 'package:mobile_app/features/account/data/in_memory_address_repository.dart';
import 'package:mobile_app/features/account/data/rest_address_repository.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';
import 'package:mobile_app/features/auth/data/mock_auth_repository.dart';
import 'package:mobile_app/features/auth/data/rest_auth_repository.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/cart/data/in_memory_cart_repository.dart';
import 'package:mobile_app/features/cart/data/rest_cart_repository.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/catalog/data/mock_catalog_repository.dart';
import 'package:mobile_app/features/catalog/data/rest_catalog_remote_data_source.dart';
import 'package:mobile_app/features/catalog/data/rest_catalog_repository.dart';
import 'package:mobile_app/features/catalog/domain/catalog_repository.dart';
import 'package:mobile_app/features/checkout/data/mock_checkout_repository.dart';
import 'package:mobile_app/features/checkout/data/rest_checkout_repository.dart';
import 'package:mobile_app/features/checkout/domain/checkout_repository.dart';
import 'package:mobile_app/features/notifications/data/local_notifications_repository.dart';
import 'package:mobile_app/features/notifications/domain/app_notification.dart';
import 'package:mobile_app/features/orders/data/mock_orders_repository.dart';
import 'package:mobile_app/features/orders/data/rest_orders_repository.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';

class AppDependencies {
  const AppDependencies({
    required this.catalogRepository,
    required this.cartRepository,
    required this.checkoutRepository,
    required this.ordersRepository,
    required this.addressRepository,
    required this.authRepository,
    required this.notificationsRepository,
  });

  final CatalogRepository catalogRepository;
  final CartRepository cartRepository;
  final CheckoutRepository checkoutRepository;
  final OrdersRepository ordersRepository;
  final AddressRepository addressRepository;
  final AuthRepository authRepository;
  final NotificationsRepository notificationsRepository;
}

abstract final class RepositoryFactory {
  static AppDependencies create({
    DataSourceMode? mode,
    String? apiBaseUrl,
    TokenStore? tokenStore,
  }) {
    final selectedMode = mode ?? AppConfig.dataSourceMode;
    if (!selectedMode.isRemote) return _local();

    final baseUrl = apiBaseUrl ?? AppConfig.apiBaseUrl;
    if (baseUrl.isEmpty) {
      throw StateError('Remote data source requires a non-empty API base URL.');
    }
    final tokens = tokenStore ?? TokenStore();
    final client = ApiClient(baseUrl: baseUrl, tokenStore: tokens);
    return AppDependencies(
      catalogRepository: RestCatalogRepository(
        RestCatalogRemoteDataSource(client),
      ),
      cartRepository: RestCartRepository(client),
      checkoutRepository: RestCheckoutRepository(client),
      ordersRepository: RestOrdersRepository(client),
      addressRepository: RestAddressRepository(client),
      authRepository: RestAuthRepository(client, tokens),
      // No notification endpoint exists in the supplied contracts.
      notificationsRepository: LocalNotificationsRepository(),
    );
  }

  static AppDependencies _local() => AppDependencies(
    catalogRepository: MockCatalogRepository(),
    cartRepository: InMemoryCartRepository(),
    checkoutRepository: MockCheckoutRepository(),
    ordersRepository: MockOrdersRepository(),
    addressRepository: InMemoryAddressRepository(),
    authRepository: MockAuthRepository(),
    notificationsRepository: LocalNotificationsRepository(),
  );
}
