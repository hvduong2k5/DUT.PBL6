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
  const CartPage({super.key, this.quoteEnabled = false});
  final bool quoteEnabled;
  Future<void> _clear(BuildContext context) async {
    final cubit = context.read<CartCubit>();
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Dọn giỏ hàng?'),
        content: const Text(
          'Bạn có chắc muốn xóa toàn bộ sản phẩm khỏi giỏ hàng?',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Giữ lại'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Xóa toàn bộ'),
          ),
        ],
      ),
    );
    if (confirmed == true &&
        context.mounted &&
        !cubit.isClosed &&
        !cubit.state.pending) {
      await cubit.clear();
    }
  }

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
              CheckboxListTile(
                contentPadding: EdgeInsets.zero,
                title: const Text('Chọn tất cả sản phẩm khả dụng'),
                subtitle: Text(
                  'Đã chọn ${state.selectedIds.length}/${state.availableIds.length} dòng',
                ),
                value:
                    state.availableIds.isNotEmpty &&
                    state.selectedIds.length == state.availableIds.length,
                onChanged: state.pending
                    ? null
                    : (_) => context.read<CartCubit>().toggleAll(),
              ),
              Align(
                alignment: Alignment.centerRight,
                child: TextButton.icon(
                  onPressed: state.pending ? null : () => _clear(context),
                  icon: const Icon(Icons.delete_outline),
                  label: const Text('Dọn giỏ hàng'),
                ),
              ),
              if (state.message != null)
                Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: Semantics(
                    liveRegion: true,
                    child: Text(
                      state.message!,
                      style: const TextStyle(color: AppColors.darkRed),
                    ),
                  ),
                ),
              if (quoteEnabled)
                Padding(
                  padding: const EdgeInsets.only(bottom: 16),
                  child: OutlinedButton(
                    onPressed: state.pending
                        ? null
                        : () => context.push(AppRoutes.quoteReview),
                    child: const Text('Xem báo giá cả giỏ'),
                  ),
                ),
              for (final line in cart.items)
                _CartLineTile(
                  line: line,
                  selected: state.selectedIds.contains(line.itemId),
                  available: state.availableIds.contains(line.itemId),
                  pending: state.pending,
                ),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(AppSpacing.md),
                  child: Column(
                    children: [
                      const Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Expanded(
                            child: Text(
                              'Tóm tắt đơn hàng',
                              style: TextStyle(fontWeight: FontWeight.bold),
                            ),
                          ),
                          Text('Tạm tính'),
                        ],
                      ),
                      const Divider(height: AppSpacing.lg),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Expanded(child: Text('${cart.totalItems} sản phẩm')),
                          Flexible(
                            child: Text(
                              cart.subtotal.format(),
                              textAlign: TextAlign.end,
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: AppSpacing.xs),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          const Expanded(
                            child: Text(
                              'Hàng đã chọn',
                              style: TextStyle(fontWeight: FontWeight.bold),
                            ),
                          ),
                          Flexible(
                            child: Text(
                              state.selectedCart.subtotal.format(),
                              style: Theme.of(context).textTheme.titleLarge,
                              textAlign: TextAlign.end,
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 12),
                      const Text(
                        'Tạm tính chưa gồm phí vận chuyển và ưu đãi. Giá và khả dụng được kiểm tra lại khi đặt hàng.',
                        style: TextStyle(
                          fontSize: 12,
                          color: AppColors.textMuted,
                        ),
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
                              'Đã chọn ${state.selectedCart.totalItems} món:',
                              style: Theme.of(context).textTheme.bodySmall,
                            ),
                            Text(
                              state.selectedCart.subtotal.format(),
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
                          onPressed: state.pending || state.selectedIds.isEmpty
                              ? null
                              : () => context.push(
                                  Uri(
                                    path: AppRoutes.checkout,
                                    queryParameters: {
                                      'items': state.selectedIds.join(','),
                                    },
                                  ).toString(),
                                ),
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
  const _CartLineTile({
    required this.line,
    required this.selected,
    required this.available,
    required this.pending,
  });
  final CartLine line;
  final bool selected;
  final bool available;
  final bool pending;
  @override
  Widget build(BuildContext context) => Card(
    margin: const EdgeInsets.only(bottom: AppSpacing.sm),
    child: Padding(
      padding: const EdgeInsets.all(AppSpacing.sm),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Column(
            children: [
              Container(
                width: 64,
                height: 64,
                decoration: BoxDecoration(
                  color: AppColors.muted,
                  borderRadius: BorderRadius.circular(AppRadius.large),
                ),
                clipBehavior: Clip.antiAlias,
                child: line.product.imageUrl.isEmpty
                    ? const Icon(
                        Icons.redeem_outlined,
                        color: AppColors.brandBrown,
                      )
                    : Image.network(
                        line.product.imageUrl,
                        fit: BoxFit.cover,
                        errorBuilder: (_, _, _) =>
                            const Icon(Icons.image_not_supported_outlined),
                      ),
              ),
              Checkbox(
                value: selected,
                semanticLabel: 'Chọn ${line.product.name} để thanh toán',
                onChanged: pending || !available
                    ? null
                    : (_) => context.read<CartCubit>().toggle(line.itemId),
              ),
            ],
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
                if (!available)
                  const Text(
                    'Quy cách hiện không khả dụng.',
                    style: TextStyle(color: AppColors.darkRed, fontSize: 12),
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
                      onPressed: pending || !available || line.quantity <= 1
                          ? null
                          : () => context.read<CartCubit>().update(
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
                      onPressed:
                          pending ||
                              !available ||
                              line.quantity >= maxCartItemQuantity
                          ? null
                          : () => context.read<CartCubit>().update(
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
                      onPressed: pending
                          ? null
                          : () => context.read<CartCubit>().remove(line.itemId),
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
