import 'package:mobile_app/core/config/app_config.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_repository.dart';
import 'package:mobile_app/features/checkout/data/local_mvp_purchase_repository.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_repository.dart';
import 'package:mobile_app/features/checkout/data/local_mvp_checkout_repository.dart';
import 'package:mobile_app/features/checkout/data/mvp_checkout_store.dart';
import 'package:mobile_app/features/orders/domain/mvp_orders_repository.dart';
import 'package:mobile_app/features/orders/data/local_mvp_orders_repository.dart';
import 'package:mobile_app/features/orders/data/mvp_cancellation_store.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/storage/token_store.dart';
import 'package:mobile_app/features/account/data/in_memory_address_repository.dart';
import 'package:mobile_app/features/account/data/rest_address_repository.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';
import 'package:mobile_app/features/auth/data/mock_auth_repository.dart';
import 'package:mobile_app/features/auth/data/local_auth_repository.dart';
import 'package:mobile_app/features/auth/data/local_password_hasher.dart';
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
import 'package:mobile_app/features/promotions/domain/promotions_repository.dart';
import 'package:mobile_app/features/promotions/data/mock_promotions_repository.dart';
import 'package:mobile_app/features/promotions/data/rest_promotions_repository.dart';
import 'package:mobile_app/features/account/domain/customer_account_repository.dart';
import 'package:mobile_app/features/account/data/mock_customer_account_repository.dart';
import 'package:mobile_app/features/account/data/rest_customer_account_repository.dart';
import 'package:mobile_app/features/orders/domain/guest_orders_repository.dart';
import 'package:mobile_app/features/orders/data/mock_guest_orders_repository.dart';
import 'package:mobile_app/features/orders/data/rest_guest_orders_repository.dart';
import 'package:mobile_app/features/support/domain/support_repository.dart';
import 'package:mobile_app/features/support/data/rest_support_repository.dart';
import 'package:mobile_app/features/support/data/mock_support_repository.dart';
import 'package:mobile_app/features/auth/domain/customer_capabilities.dart';
import 'package:mobile_app/features/auth/data/customer_capabilities_repositories.dart';
import 'package:mobile_app/features/trace/domain/trace_repository.dart';
import 'package:mobile_app/features/trace/data/trace_repositories.dart';
import 'package:mobile_app/features/returns/domain/returns_repository.dart';
import 'package:mobile_app/features/returns/data/rest_returns_repository.dart';
import 'package:mobile_app/features/returns/data/mock_returns_repository.dart';
import 'package:mobile_app/features/b2b/domain/b2b_repository.dart';
import 'package:mobile_app/features/b2b/data/rest_b2b_repository.dart';
import 'package:mobile_app/features/b2b/data/mock_b2b_repository.dart';
import 'package:mobile_app/features/reviews/domain/reviews_repository.dart';
import 'package:mobile_app/features/reviews/data/rest_reviews_repository.dart';
import 'package:mobile_app/features/reviews/data/mock_reviews_repository.dart';

class AppDependencies {
  const AppDependencies({
    this.mvpPurchaseRepository,
    this.mvpCheckoutRuntime,
    this.mvpOrdersRuntime,
    required this.catalogRepository,
    required this.cartRepository,
    required this.checkoutRepository,
    required this.ordersRepository,
    required this.addressRepository,
    required this.authRepository,
    required this.notificationsRepository,
    required this.promotionsRepository,
    required this.customerAccountRepository,
    required this.guestOrdersRepository,
    required this.supportRepository,
    required this.capabilitiesRepository,
    required this.traceRepository,
    required this.returnsRepository,
    required this.b2bRepository,
    required this.reviewsRepository,
  });

  final CatalogRepository catalogRepository;
  final MvpPurchaseRepository? mvpPurchaseRepository;
  final MvpCheckoutRuntime? mvpCheckoutRuntime;
  final MvpOrdersRuntime? mvpOrdersRuntime;
  final CartRepository cartRepository;
  final CheckoutRepository checkoutRepository;
  final OrdersRepository ordersRepository;
  final AddressRepository addressRepository;
  final AuthRepository authRepository;
  final NotificationsRepository notificationsRepository;
  final PromotionsRepository promotionsRepository;
  final CustomerAccountRepository customerAccountRepository;
  final GuestOrdersRepository guestOrdersRepository;
  final SupportRepository supportRepository;
  final CustomerCapabilitiesRepository capabilitiesRepository;
  final TraceRepository traceRepository;
  final ReturnsRepository returnsRepository;
  final B2bRepository b2bRepository;
  final ReviewsRepository reviewsRepository;
}

abstract final class RepositoryFactory {
  static AppDependencies create({
    DataSourceMode? mode,
    String? apiBaseUrl,
    TokenStore? tokenStore,
    bool localMvpCheckout = true,
    LocalPasswordHasher? localPasswordHasher,
  }) {
    final selectedMode = mode ?? AppConfig.dataSourceMode;
    if (!selectedMode.isRemote) {
      return _local(
        localMvpCheckout: localMvpCheckout,
        localPasswordHasher: localPasswordHasher,
      );
    }

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
      promotionsRepository: RestPromotionsRepository(client),
      customerAccountRepository: RestCustomerAccountRepository(client),
      guestOrdersRepository: RestGuestOrdersRepository(client),
      supportRepository: RestSupportRepository(client),
      capabilitiesRepository: RestCustomerCapabilitiesRepository(client),
      traceRepository: RestTraceRepository(client),
      b2bRepository: RestB2bRepository(client),
      reviewsRepository: RestReviewsRepository(client),
      returnsRepository: RestReturnsRepository(
        client,
        RestOrdersRepository(client),
      ),
    );
  }

  static AppDependencies _local({
    required bool localMvpCheckout,
    LocalPasswordHasher? localPasswordHasher,
  }) {
    final auth = localMvpCheckout
        ? LocalAuthRepository(hasher: localPasswordHasher)
        : MockAuthRepository();
    final catalog = MockCatalogRepository();
    final cart = InMemoryCartRepository(
      principal: localMvpCheckout ? () => auth.currentUser?.id : null,
    );
    final purchase = LocalMvpPurchaseRepository(
      cart,
      () => auth.currentUser != null,
      principal: () => auth.currentUser?.id,
    );
    final orders = MockOrdersRepository(
      authenticated: () => auth.currentUser != null,
      principal: localMvpCheckout ? () => auth.currentUser?.id : null,
    );
    final mvpOrders = LocalMvpOrdersRepository(() => auth.currentUser?.id);
    return AppDependencies(
      catalogRepository: catalog,
      mvpPurchaseRepository: purchase,
      mvpOrdersRuntime: localMvpCheckout
          ? MvpOrdersRuntime(
              repository: mvpOrders,
              cancellations: MemoryMvpCancellationStore(),
              scope: 'local-process-mvp-v1',
              local: true,
            )
          : null,
      mvpCheckoutRuntime: localMvpCheckout
          ? MvpCheckoutRuntime(
              repository: LocalMvpCheckoutRepository(
                purchase,
                () => auth.currentUser?.id,
                onPlaced: mvpOrders.record,
              ),
              store: MemoryMvpCheckoutStore(),
              scope: 'local-process-mvp-v1',
              local: true,
            )
          : null,
      reviewsRepository: MockReviewsRepository(
        catalog: catalog,
        orders: orders,
        currentUser: () => auth.currentUser,
      ),
      cartRepository: cart,
      checkoutRepository: MockCheckoutRepository(onCreated: orders.record),
      ordersRepository: orders,
      addressRepository: InMemoryAddressRepository(
        principal: localMvpCheckout ? () => auth.currentUser?.id : null,
      ),
      authRepository: auth,
      notificationsRepository: LocalNotificationsRepository(),
      promotionsRepository: MockPromotionsRepository(),
      customerAccountRepository: MockCustomerAccountRepository(
        currentUser: () => auth.currentUser,
        onUpdate: auth.updateUser,
      ),
      guestOrdersRepository: MockGuestOrdersRepository(orders),
      traceRepository: MockTraceRepository(),
      b2bRepository: MockB2bRepository(
        currentUser: () => auth.currentUser,
        onAccepted: orders.recordBusinessOrder,
      ),
      returnsRepository: MockReturnsRepository(
        orders,
        () => auth.currentUser?.id,
      ),
      capabilitiesRepository: MockCustomerCapabilitiesRepository(
        () => auth.currentUser,
      ),
      supportRepository: MockSupportRepository(
        currentUser: () => auth.currentUser,
        orders: orders,
      ),
    );
  }
}
