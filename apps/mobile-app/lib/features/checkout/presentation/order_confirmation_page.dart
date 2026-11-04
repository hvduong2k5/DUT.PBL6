import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_secondary_button.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/network/api_exception.dart';

class OrderConfirmationPage extends StatelessWidget {
  const OrderConfirmationPage({
    super.key,
    required this.orderId,
    this.confirmation,
  });
  final String orderId;
  final OrderConfirmation? confirmation;
  @override
  Widget build(BuildContext context) {
    if (confirmation == null) return _ConfirmationLookup(orderId: orderId);
    return Scaffold(
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
                  confirmation!.status == 'PENDING_PAYMENT'
                      ? 'Đơn đã tạo, chờ thanh toán'
                      : confirmation!.paymentMethod == PaymentMethod.cod
                      ? 'Đã tiếp nhận đơn COD'
                      : 'Xác nhận đơn hàng',
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
                BlocBuilder<CartCubit, CartState>(
                  builder: (_, state) => state.cleanupFailed
                      ? const Padding(
                          padding: EdgeInsets.all(12),
                          child: Text(
                            'Đơn đã tạo nhưng giỏ chưa đồng bộ. Hãy kiểm tra đơn trước khi đặt lại.',
                            style: TextStyle(color: AppColors.darkRed),
                          ),
                        )
                      : const SizedBox.shrink(),
                ),
                if (confirmation!.paymentMethod != null) ...[
                  SizedBox(
                    width: double.infinity,
                    child: AppPrimaryButton(
                      label: 'Xem thông tin thanh toán',
                      onPressed: () => context.push(
                        '/payment/$orderId',
                        extra: confirmation,
                      ),
                    ),
                  ),
                  const SizedBox(height: 12),
                ],
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
}

class _ConfirmationLookup extends StatefulWidget {
  const _ConfirmationLookup({required this.orderId});
  final String orderId;
  @override
  State<_ConfirmationLookup> createState() => _ConfirmationLookupState();
}

class _ConfirmationLookupState extends State<_ConfirmationLookup> {
  late Future<OrderConfirmation> _future;
  @override
  void initState() {
    super.initState();
    _load();
  }

  void _load() {
    _future = context
        .read<OrdersRepository>()
        .order(widget.orderId)
        .then(
          (order) => OrderConfirmation(
            orderId: order.id,
            status: order.rawStatus ?? order.status.name,
            total: order.total,
          ),
        );
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    body: FutureBuilder<OrderConfirmation>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done) {
          return const AppLoadingView();
        }
        if (snapshot.hasError || snapshot.data == null) {
          return AppErrorView(
            message: snapshot.error is ApiException
                ? (snapshot.error as ApiException).failure.message
                : 'Chưa thể mở xác nhận đơn hàng.',
            onRetry: () => setState(_load),
          );
        }
        return OrderConfirmationPage(
          orderId: widget.orderId,
          confirmation: snapshot.data,
        );
      },
    ),
  );
}
