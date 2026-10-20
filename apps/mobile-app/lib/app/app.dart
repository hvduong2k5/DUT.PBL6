import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/app/router/app_router.dart';
import 'package:mobile_app/core/storage/onboarding_store.dart';
import 'package:mobile_app/core/theme/app_theme.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/catalog/domain/catalog_repository.dart';
import 'package:mobile_app/features/checkout/domain/checkout_repository.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/notifications/domain/app_notification.dart';
import 'package:mobile_app/features/promotions/domain/promotions_repository.dart';
import 'package:mobile_app/features/account/domain/customer_account_repository.dart';
import 'package:mobile_app/features/orders/domain/guest_orders_repository.dart';
import 'package:mobile_app/features/support/domain/support_repository.dart';
import 'package:mobile_app/features/auth/presentation/capabilities_cubit.dart';
import 'package:mobile_app/features/trace/domain/trace_repository.dart';

class OmaApp extends StatefulWidget {
  const OmaApp({
    super.key,
    this.onboardingStore,
    this.onboardingCompleted = true,
    this.initialLocation,
    this.dependencies,
  });

  final OnboardingStore? onboardingStore;
  final bool onboardingCompleted;
  final String? initialLocation;
  final AppDependencies? dependencies;

  @override
  State<OmaApp> createState() => _OmaAppState();
}

class _OmaAppState extends State<OmaApp> {
  late final AppDependencies _dependencies;
  late final OnboardingStore _onboardingStore;
  late final GoRouter _router;

  @override
  void initState() {
    super.initState();
    _dependencies = widget.dependencies ?? RepositoryFactory.create();
    _onboardingStore = widget.onboardingStore ?? OnboardingStore();
    _router = createAppRouter(
      onboardingStore: _onboardingStore,
      onboardingCompleted: widget.onboardingCompleted,
      initialLocation: widget.initialLocation,
    );
  }

  @override
  void dispose() {
    _router.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MultiRepositoryProvider(
    providers: [
      RepositoryProvider<TraceRepository>.value(
        value: _dependencies.traceRepository,
      ),
      RepositoryProvider<SupportRepository>.value(
        value: _dependencies.supportRepository,
      ),
      RepositoryProvider<GuestOrdersRepository>.value(
        value: _dependencies.guestOrdersRepository,
      ),
      RepositoryProvider<CustomerAccountRepository>.value(
        value: _dependencies.customerAccountRepository,
      ),
      RepositoryProvider<PromotionsRepository>.value(
        value: _dependencies.promotionsRepository,
      ),
      RepositoryProvider<CatalogRepository>.value(
        value: _dependencies.catalogRepository,
      ),
      RepositoryProvider<CartRepository>.value(
        value: _dependencies.cartRepository,
      ),
      RepositoryProvider<CheckoutRepository>.value(
        value: _dependencies.checkoutRepository,
      ),
      RepositoryProvider<OrdersRepository>.value(
        value: _dependencies.ordersRepository,
      ),
      RepositoryProvider<AddressRepository>.value(
        value: _dependencies.addressRepository,
      ),
      RepositoryProvider<AuthRepository>.value(
        value: _dependencies.authRepository,
      ),
      RepositoryProvider<NotificationsRepository>.value(
        value: _dependencies.notificationsRepository,
      ),
    ],
    child: MultiBlocProvider(
      providers: [
        BlocProvider(
          create: (_) => CartCubit(_dependencies.cartRepository)..load(),
        ),
        BlocProvider(
          create: (_) => AuthCubit(_dependencies.authRepository)..restore(),
        ),
        BlocProvider(
          lazy: false,
          create: (context) => CapabilitiesCubit(
            _dependencies.capabilitiesRepository,
            context.read<AuthCubit>(),
          ),
        ),
      ],
      child: MaterialApp.router(
        title: 'Ô Mạ',
        debugShowCheckedModeBanner: false,
        theme: AppTheme.light,
        routerConfig: _router,
      ),
    ),
  );
}
