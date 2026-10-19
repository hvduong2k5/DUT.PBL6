import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';
import 'package:mobile_app/features/auth/presentation/auth_return_location.dart';

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
    builder: (context, state) => state.status == AuthStatus.authenticated
        ? child
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
