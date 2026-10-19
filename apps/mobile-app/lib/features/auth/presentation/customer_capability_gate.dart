import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/features/auth/presentation/capabilities_cubit.dart';

class CustomerCapabilityGate extends StatelessWidget {
  const CustomerCapabilityGate({
    super.key,
    required this.capability,
    required this.child,
    this.requireB2b = false,
  });
  final String capability;
  final bool requireB2b;
  final Widget child;
  @override
  Widget build(
    BuildContext context,
  ) => BlocBuilder<CapabilitiesCubit, CapabilitiesState>(
    builder: (context, state) {
      if (!state.loading &&
          state.allows(capability) &&
          (!requireB2b || state.projection?.actor == 'B2B')) {
        return child;
      }
      return Scaffold(
        appBar: const AppMobileHeader(title: 'Quyền truy cập', showBack: true),
        body: state.loading
            ? const AppLoadingView()
            : state.error != null
            ? AppErrorView(
                message: state.error!,
                onRetry: context.read<CapabilitiesCubit>().load,
              )
            : AppEmptyView(
                title: 'Chưa có quyền truy cập',
                message: requireB2b
                    ? 'Tính năng dành cho tổ chức B2B đã được xác minh.'
                    : 'Tài khoản hiện tại chưa được cấp quyền sử dụng tính năng này.',
              ),
      );
    },
  );
}
