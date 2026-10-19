import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';
import 'package:mobile_app/features/auth/presentation/auth_return_location.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/features/auth/presentation/customer_capability_gate.dart';

class CustomerAccountGate extends StatelessWidget {
  const CustomerAccountGate({
    super.key,
    required this.returnLocation,
    required this.child,
  });
  final String returnLocation;
  final Widget child;
  @override
  Widget build(BuildContext context) => BlocBuilder<AuthCubit, AuthState>(
    builder: (context, state) => state.status == AuthStatus.restoring
        ? const Scaffold(body: AppLoadingView())
        : state.sessionRestoreFailed
        ? Scaffold(
            appBar: const AppMobileHeader(
              title: 'Phiên tài khoản',
              showBack: true,
            ),
            body: AppErrorView(
              message: state.message!,
              onRetry: context.read<AuthCubit>().restore,
              action: TextButton(
                onPressed: () =>
                    context.go(authLocation(AppRoutes.login, returnLocation)),
                child: const Text('Đăng nhập lại'),
              ),
            ),
          )
        : state.status == AuthStatus.authenticated
        ? CustomerCapabilityGate(
            capability: switch (Uri.parse(returnLocation).path) {
              AppRoutes.loyalty => 'LOYALTY_VIEW',
              AppRoutes.orders => 'ORDER_HISTORY_VIEW',
              AppRoutes.addresses || AppRoutes.addressForm => 'ADDRESS_MANAGE',
              _ => 'PROFILE_MANAGE',
            },
            child: child,
          )
        : Scaffold(
            appBar: const AppMobileHeader(
              title: 'Tài khoản Tri Kỷ',
              showBack: true,
            ),
            body: AppEmptyView(
              title: 'Đăng nhập để tiếp tục',
              message:
                  'Thông tin tài khoản của bạn được bảo vệ trong phiên đăng nhập.',
              action: AppPrimaryButton(
                label: 'Đăng nhập',
                onPressed: () =>
                    context.go(authLocation(AppRoutes.login, returnLocation)),
              ),
            ),
          ),
  );
}
