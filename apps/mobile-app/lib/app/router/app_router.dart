import 'package:flutter/widgets.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/app/router/safe_navigation.dart';
import 'package:mobile_app/core/storage/onboarding_store.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';
import 'package:mobile_app/features/account/presentation/account_page.dart';
import 'package:mobile_app/features/account/presentation/address_form_page.dart';
import 'package:mobile_app/features/account/presentation/addresses_cubit.dart';
import 'package:mobile_app/features/account/presentation/addresses_page.dart';
import 'package:mobile_app/features/auth/presentation/auth_page.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/auth/presentation/auth_return_location.dart';
import 'package:mobile_app/features/auth/presentation/recovery_cubit.dart';
import 'package:mobile_app/features/auth/presentation/recovery_page.dart';
import 'package:mobile_app/features/promotions/domain/promotions_repository.dart';
import 'package:mobile_app/features/promotions/presentation/promotions_cubit.dart';
import 'package:mobile_app/features/promotions/presentation/promotions_page.dart';
import 'package:mobile_app/features/cart/presentation/cart_page.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/catalog/domain/catalog_repository.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_page.dart';
import 'package:mobile_app/features/catalog/presentation/home_page.dart';
import 'package:mobile_app/features/catalog/presentation/product_detail_page.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_repository.dart';
import 'package:mobile_app/features/checkout/presentation/checkout_cubit.dart';
import 'package:mobile_app/features/checkout/presentation/checkout_page.dart';
import 'package:mobile_app/features/checkout/presentation/order_confirmation_page.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/orders/presentation/order_detail_page.dart';
import 'package:mobile_app/features/orders/presentation/orders_cubit.dart';
import 'package:mobile_app/features/orders/presentation/orders_page.dart';
import 'package:mobile_app/features/notifications/domain/app_notification.dart';
import 'package:mobile_app/features/notifications/presentation/notifications_cubit.dart';
import 'package:mobile_app/features/notifications/presentation/notifications_page.dart';
import 'package:mobile_app/features/onboarding/presentation/onboarding_page.dart';

GoRouter createAppRouter({
  required OnboardingStore onboardingStore,
  bool onboardingCompleted = true,
  String? initialLocation,
}) => GoRouter(
  initialLocation:
      initialLocation ??
      (onboardingCompleted ? AppRoutes.home : AppRoutes.onboarding),
  routes: [
    GoRoute(
      path: AppRoutes.promotions,
      builder: (context, state) => SafeBackScope(
        child: BlocProvider(
          create: (_) => PromotionsCubit(
            context.read<PromotionsRepository>(),
            subtotal:
                int.tryParse(state.uri.queryParameters['subtotal'] ?? '') ?? 0,
          )..load(),
          child: PromotionsPage(
            checkoutSubtotal: int.tryParse(
              state.uri.queryParameters['subtotal'] ?? '',
            ),
          ),
        ),
      ),
    ),
    GoRoute(
      path: AppRoutes.onboarding,
      builder: (context, state) => OnboardingPage(store: onboardingStore),
    ),
    GoRoute(
      path: AppRoutes.home,
      builder: (context, state) => BlocProvider(
        create: (_) => CatalogCubit(context.read<CatalogRepository>())..load(),
        child: const HomePage(),
      ),
    ),
    GoRoute(
      path: AppRoutes.catalog,
      builder: (context, state) => SafeBackScope(
        child: BlocProvider(
          create: (_) =>
              CatalogCubit(context.read<CatalogRepository>())..load(),
          child: const CatalogPage(),
        ),
      ),
    ),
    GoRoute(
      path: AppRoutes.product,
      builder: (context, state) => SafeBackScope(
        child: ProductDetailPage(productId: state.pathParameters['id']!),
      ),
    ),
    GoRoute(
      path: AppRoutes.cart,
      builder: (context, state) => const SafeBackScope(child: CartPage()),
    ),
    GoRoute(
      path: AppRoutes.checkout,
      builder: (context, state) => SafeBackScope(
        child: BlocProvider(
          create: (_) {
            final cubit = CheckoutCubit(
              context.read<CheckoutRepository>(),
              Cart(
                id: context.read<CartCubit>().state.cart.id,
                items: context
                    .read<CartCubit>()
                    .state
                    .cart
                    .items
                    .where(
                      (line) =>
                          (state.uri.queryParameters['items'] ?? '')
                              .split(',')
                              .contains(line.itemId) &&
                          line.product.inStock &&
                          line.variant.stock > 0,
                    )
                    .toList(),
              ),
            );
            cubit.loadLocations();
            if (context.read<AuthCubit>().state.status ==
                AuthStatus.authenticated) {
              cubit.loadSavedAddress(context.read<AddressRepository>());
            }
            return cubit;
          },
          child: const CheckoutPage(),
        ),
      ),
    ),
    GoRoute(
      path: AppRoutes.orderConfirmation,
      builder: (context, state) => SafeBackScope(
        child: OrderConfirmationPage(
          orderId: state.pathParameters['id']!,
          confirmation: state.extra as OrderConfirmation?,
        ),
      ),
    ),
    GoRoute(
      path: AppRoutes.orders,
      builder: (context, state) => SafeBackScope(
        child: BlocProvider(
          create: (_) => OrdersCubit(context.read<OrdersRepository>())..load(),
          child: const OrdersPage(),
        ),
      ),
    ),
    GoRoute(
      path: AppRoutes.orderDetail,
      builder: (context, state) => SafeBackScope(
        child: OrderDetailPage(orderId: state.pathParameters['id']!),
      ),
    ),
    GoRoute(
      path: AppRoutes.account,
      builder: (context, state) => const SafeBackScope(child: AccountPage()),
    ),
    GoRoute(
      path: AppRoutes.addresses,
      builder: (context, state) => SafeBackScope(
        child: BlocProvider(
          create: (_) =>
              AddressesCubit(context.read<AddressRepository>())..load(),
          child: const AddressesPage(),
        ),
      ),
    ),
    GoRoute(
      path: AppRoutes.addressForm,
      builder: (context, state) => SafeBackScope(
        child: AddressFormPage(address: state.extra as Address?),
      ),
    ),
    GoRoute(
      path: AppRoutes.notifications,
      builder: (context, state) => SafeBackScope(
        child: BlocProvider(
          create: (_) =>
              NotificationsCubit(context.read<NotificationsRepository>())
                ..load(),
          child: const NotificationsPage(),
        ),
      ),
    ),
    GoRoute(
      path: AppRoutes.login,
      builder: (context, state) => SafeBackScope(
        child: AuthPage(
          register: false,
          returnLocation: safeAuthReturnLocation(
            state.uri.queryParameters['returnUrl'],
          ),
        ),
      ),
    ),
    GoRoute(
      path: AppRoutes.register,
      builder: (context, state) => SafeBackScope(
        child: AuthPage(
          register: true,
          returnLocation: safeAuthReturnLocation(
            state.uri.queryParameters['returnUrl'],
          ),
        ),
      ),
    ),
    GoRoute(
      path: AppRoutes.recovery,
      builder: (context, state) => SafeBackScope(
        child: BlocProvider(
          key: ValueKey(state.uri.toString()),
          create: (_) => RecoveryCubit(context.read<AuthRepository>()),
          child: RecoveryPage(
            returnLocation: safeAuthReturnLocation(
              state.uri.queryParameters['returnUrl'],
            ),
            recoveryToken: state.uri.queryParameters['recoveryToken'],
          ),
        ),
      ),
    ),
  ],
);
