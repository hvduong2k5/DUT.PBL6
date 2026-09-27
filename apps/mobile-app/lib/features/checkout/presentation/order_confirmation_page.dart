import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_secondary_button.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';

class OrderConfirmationPage extends StatelessWidget {
  const OrderConfirmationPage({
    super.key,
    required this.orderId,
    this.confirmation,
  });
  final String orderId;
  final OrderConfirmation? confirmation;
  @override
  Widget build(BuildContext context) => Scaffold(
    body: SafeArea(
      child: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Column(
            children: [
              Text(
                'Ô Mạ',
                style: TextStyle(
                  fontFamily: GoogleFonts.notoSerif().fontFamily,
                  fontSize: 22,
                  color: AppColors.headingBrown,
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(height: 36),
              Container(
                width: 72,
                height: 72,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: AppColors.success.withValues(alpha: .16),
                  border: Border.all(color: AppColors.success),
                ),
                child: const Icon(
                  Icons.check,
                  color: AppColors.success,
                  size: 34,
                ),
              ),
              const SizedBox(height: AppSpacing.lg),
              Text(
                'Đặt hàng thành công!',
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: AppSpacing.xs),
              const Text(
                'Đơn hàng của bạn đã được ghi nhận. Ô Mạ sẽ thông báo khi có cập nhật mới.',
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: AppSpacing.lg),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(AppSpacing.md),
                  child: Column(
                    children: [
                      const Text(
                        'MÃ ĐƠN HÀNG',
                        style: TextStyle(
                          fontSize: 10,
                          color: AppColors.textMuted,
                        ),
                      ),
                      const SizedBox(height: 6),
                      Text(
                        orderId,
                        style: Theme.of(context).textTheme.titleLarge,
                      ),
                      if (confirmation != null) ...[
                        const SizedBox(height: 10),
                        Text(confirmation!.total.format()),
                      ],
                    ],
                  ),
                ),
              ),
              const SizedBox(height: AppSpacing.lg),
              SizedBox(
                width: double.infinity,
                child: AppSecondaryButton(
                  label: 'Theo dõi đơn hàng',
                  onPressed: () => context.go('/orders/$orderId'),
                ),
              ),
              const SizedBox(height: AppSpacing.sm),
              SizedBox(
                width: double.infinity,
                child: AppPrimaryButton(
                  label: 'Tiếp tục mua sắm',
                  onPressed: () => context.go(AppRoutes.catalog),
                ),
              ),
            ],
          ),
        ),
      ),
    ),
  );
}
