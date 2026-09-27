import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_radius.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_bottom_navigation.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';

class CartPage extends StatelessWidget {
  const CartPage({super.key});
  @override
  Widget build(BuildContext context) => BlocBuilder<CartCubit, CartState>(
    builder: (context, state) {
      final cart = state.cart;
      return Scaffold(
        appBar: AppMobileHeader(
          title: cart.items.isEmpty
              ? 'Giỏ Hàng'
              : 'Giỏ Hàng (${cart.totalItems} món)',
          showBack: true,
        ),
        body: switch (state.status) {
          CartStatus.initial || CartStatus.loading => const AppLoadingView(),
          CartStatus.failure => AppErrorView(
            message: state.message ?? 'Không thể tải giỏ hàng.',
            onRetry: context.read<CartCubit>().load,
          ),
          _ when cart.items.isEmpty => AppEmptyView(
            title: 'Giỏ hàng trống',
            message:
                'Bạn chưa có sản phẩm nào trong giỏ. Khám phá những món quà mang hương vị xứ Huế.',
            icon: Icons.shopping_bag_outlined,
            action: SizedBox(
              width: 220,
              child: AppPrimaryButton(
                label: 'Khám phá sản phẩm',
                icon: Icons.arrow_forward,
                onPressed: () => context.go(AppRoutes.catalog),
              ),
            ),
          ),
          _ => ListView(
            padding: const EdgeInsets.fromLTRB(
              AppSpacing.md,
              AppSpacing.xs,
              AppSpacing.md,
              120,
            ),
            children: [
              for (final line in cart.items) _CartLineTile(line: line),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(AppSpacing.md),
                  child: Column(
                    children: [
                      const Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Text(
                            'Tóm tắt đơn hàng',
                            style: TextStyle(fontWeight: FontWeight.bold),
                          ),
                          Text('Tạm tính'),
                        ],
                      ),
                      const Divider(height: AppSpacing.lg),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Text('${cart.totalItems} sản phẩm'),
                          Text(cart.subtotal.format()),
                        ],
                      ),
                      const SizedBox(height: AppSpacing.xs),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          const Text(
                            'Tổng cộng',
                            style: TextStyle(fontWeight: FontWeight.bold),
                          ),
                          Text(
                            cart.subtotal.format(),
                            style: Theme.of(context).textTheme.titleLarge,
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        },
        bottomSheet: cart.items.isEmpty
            ? null
            : SafeArea(
                top: false,
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: AppSpacing.md,
                    vertical: AppSpacing.sm,
                  ),
                  decoration: const BoxDecoration(
                    color: AppColors.ivory,
                    border: Border(top: BorderSide(color: AppColors.border)),
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              'Tổng tạm tính (${cart.totalItems} món):',
                              style: Theme.of(context).textTheme.bodySmall,
                            ),
                            Text(
                              cart.subtotal.format(),
                              style: const TextStyle(
                                fontSize: 18,
                                fontWeight: FontWeight.bold,
                                color: AppColors.brickRed,
                              ),
                            ),
                          ],
                        ),
                      ),
                      SizedBox(
                        width: 150,
                        child: AppPrimaryButton(
                          label: 'Thanh Toán',
                          icon: Icons.arrow_forward,
                          onPressed: () => context.push(AppRoutes.checkout),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
        bottomNavigationBar: AppBottomNavigation(
          activeIndex: 2,
          cartCount: cart.totalItems,
          onDestinationSelected: (index) {
            if (index == 0) context.go(AppRoutes.home);
            if (index == 1) context.go(AppRoutes.catalog);
            if (index == 3) context.go(AppRoutes.account);
          },
        ),
      );
    },
  );
}

class _CartLineTile extends StatelessWidget {
  const _CartLineTile({required this.line});
  final CartLine line;
  @override
  Widget build(BuildContext context) => Card(
    margin: const EdgeInsets.only(bottom: AppSpacing.sm),
    child: Padding(
      padding: const EdgeInsets.all(AppSpacing.sm),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 80,
            height: 80,
            decoration: BoxDecoration(
              color: AppColors.muted,
              borderRadius: BorderRadius.circular(AppRadius.large),
            ),
            child: const Icon(
              Icons.redeem_outlined,
              color: AppColors.brandBrown,
            ),
          ),
          const SizedBox(width: AppSpacing.sm),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  line.product.name,
                  maxLines: 2,
                  style: const TextStyle(fontWeight: FontWeight.w600),
                ),
                Text(
                  line.variant.name,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
                Text(
                  line.subtotal.format(),
                  style: const TextStyle(
                    color: AppColors.brickRed,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                Row(
                  children: [
                    IconButton(
                      onPressed: () => context.read<CartCubit>().update(
                        line.itemId,
                        line.quantity - 1,
                      ),
                      icon: const Icon(Icons.remove),
                      constraints: const BoxConstraints.tightFor(
                        width: 44,
                        height: 44,
                      ),
                    ),
                    Text('${line.quantity}'),
                    IconButton(
                      onPressed: () => context.read<CartCubit>().update(
                        line.itemId,
                        line.quantity + 1,
                      ),
                      icon: const Icon(Icons.add),
                      constraints: const BoxConstraints.tightFor(
                        width: 44,
                        height: 44,
                      ),
                    ),
                    const Spacer(),
                    IconButton(
                      onPressed: () =>
                          context.read<CartCubit>().remove(line.itemId),
                      icon: const Icon(Icons.delete_outline),
                      tooltip: 'Xóa sản phẩm',
                      constraints: const BoxConstraints.tightFor(
                        width: 44,
                        height: 44,
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    ),
  );
}
