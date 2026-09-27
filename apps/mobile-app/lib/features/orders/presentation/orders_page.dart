import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/theme/app_radius.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/presentation/orders_cubit.dart';

class OrdersPage extends StatelessWidget {
  const OrdersPage({super.key});

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Đơn hàng của tôi', showBack: true),
    body: BlocBuilder<OrdersCubit, OrdersState>(
      builder: (context, state) => Column(
        children: [
          _OrderFilters(selected: state.filter),
          Expanded(child: _OrderResults(state: state)),
        ],
      ),
    ),
  );
}

class _OrderFilters extends StatelessWidget {
  const _OrderFilters({required this.selected});
  final OrderStatus? selected;

  @override
  Widget build(BuildContext context) => SizedBox(
    height: 54,
    child: ListView(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      scrollDirection: Axis.horizontal,
      children: [
        FilterChip(
          label: const Text('Tất cả'),
          selected: selected == null,
          onSelected: (_) => context.read<OrdersCubit>().filter(null),
        ),
        const SizedBox(width: 8),
        for (final status in OrderStatus.values) ...[
          FilterChip(
            label: Text(status.label),
            selected: selected == status,
            onSelected: (_) => context.read<OrdersCubit>().filter(status),
          ),
          const SizedBox(width: 8),
        ],
      ],
    ),
  );
}

class _OrderResults extends StatelessWidget {
  const _OrderResults({required this.state});
  final OrdersState state;

  @override
  Widget build(BuildContext context) {
    if (state.status == OrdersStatus.initial ||
        state.status == OrdersStatus.loading) {
      return const AppLoadingView();
    }
    if (state.status == OrdersStatus.failure) {
      return AppErrorView(
        message: state.message ?? 'Không thể tải đơn hàng.',
        onRetry: context.read<OrdersCubit>().load,
      );
    }
    if (state.visible.isEmpty) {
      return AppEmptyView(
        title: 'Chưa có đơn hàng',
        message: 'Khi bạn đặt hàng, lịch sử đơn sẽ xuất hiện tại đây.',
        icon: Icons.receipt_long_outlined,
        action: FilledButton(
          onPressed: () => context.go('/catalog'),
          child: const Text('Khám phá sản phẩm'),
        ),
      );
    }
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: state.visible.length,
      itemBuilder: (context, index) => _OrderCard(order: state.visible[index]),
    );
  }
}

class _OrderCard extends StatelessWidget {
  const _OrderCard({required this.order});
  final CustomerOrder order;

  @override
  Widget build(BuildContext context) => Card(
    margin: const EdgeInsets.only(bottom: 12),
    child: InkWell(
      onTap: () => context.push('/orders/${order.id}'),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    '#${order.id}',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
                _StatusBadge(status: order.status),
              ],
            ),
            const SizedBox(height: 4),
            Align(
              alignment: Alignment.centerLeft,
              child: Text(
                DateFormat('dd/MM/yyyy').format(order.createdAt),
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ),
            const Divider(height: 24),
            Row(
              children: [
                const CircleAvatar(
                  backgroundColor: AppColors.muted,
                  child: Icon(
                    Icons.redeem_outlined,
                    color: AppColors.brandBrown,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(child: Text('${order.totalItems} sản phẩm')),
                Text(
                  order.total.format(),
                  style: const TextStyle(fontWeight: FontWeight.bold),
                ),
                const Icon(Icons.chevron_right),
              ],
            ),
          ],
        ),
      ),
    ),
  );
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.status});
  final OrderStatus status;

  @override
  Widget build(BuildContext context) {
    final cancelled = status == OrderStatus.cancelled;
    return Container(
      padding: const EdgeInsets.symmetric(
        horizontal: AppSpacing.sm,
        vertical: 5,
      ),
      decoration: BoxDecoration(
        color: cancelled ? AppColors.alertPink : AppColors.softYellow,
        border: Border.all(
          color: cancelled ? AppColors.darkRed : AppColors.honeyGold,
        ),
        borderRadius: BorderRadius.circular(AppRadius.large),
      ),
      child: Text(
        status.label,
        style: TextStyle(
          fontSize: 10,
          color: cancelled ? AppColors.darkRed : AppColors.brandBrown,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}
