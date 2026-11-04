import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'auth_cubit.dart';

class PurchaseAccessBanner extends StatelessWidget {
  const PurchaseAccessBanner({super.key});
  @override
  Widget build(BuildContext context) => BlocBuilder<AuthCubit, AuthState>(
    builder: (context, state) {
      final message = state.purchaseRestriction;
      if (message == null) return const SizedBox.shrink();
      return Container(
        margin: const EdgeInsets.all(16),
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: AppColors.alertPink,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Semantics(
          liveRegion: true,
          child: Text(
            message,
            style: const TextStyle(color: AppColors.darkRed),
          ),
        ),
      );
    },
  );
}
