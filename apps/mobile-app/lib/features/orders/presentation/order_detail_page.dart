import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:intl/intl.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/presentation/capabilities_cubit.dart';
import 'package:mobile_app/features/orders/presentation/order_actions_pages.dart';

class OrderDetailPage extends StatefulWidget {
  const OrderDetailPage({super.key, required this.orderId});
  final String orderId;
  @override
  State<OrderDetailPage> createState() => _OrderDetailPageState();
}

class _OrderDetailPageState extends State<OrderDetailPage> {
  late Future<CustomerOrder> _future;
  @override
  void initState() {
    super.initState();
    _load();
  }

  void _load() =>
      _future = context.read<OrdersRepository>().order(widget.orderId);
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppMobileHeader(
      title: 'Đơn hàng #${widget.orderId}',
      showBack: true,
    ),
    body: FutureBuilder<CustomerOrder>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done) {
          return const AppLoadingView();
        }
        if (snapshot.hasError || snapshot.data == null) {
          return AppErrorView(
            message: snapshot.error is ApiException
                ? (snapshot.error as ApiException).failure.message
                : 'Không thể tải chi tiết đơn hàng.',
            onRetry: () => setState(_load),
            action: TextButton(
              onPressed: () => context.go(AppRoutes.guestOrderLookup),
              child: const Text('Xác minh đơn khách'),
            ),
          );
        }
        final order = snapshot.data!;
        return ListView(
          padding: const EdgeInsets.all(16),
          children: [
            Align(
              alignment: Alignment.centerLeft,
              child: Chip(
                label: Text(
                  order.status == OrderStatus.unknown
                      ? (order.rawStatus ?? order.status.label)
                      : order.status.label,
                ),
              ),
            ),
            if (const {
              'PENDING',
              'PENDING_PAYMENT',
              'AWAITING_CONFIRMATION',
            }.contains(order.rawStatus))
              BlocBuilder<CapabilitiesCubit, CapabilitiesState>(
                builder: (context, capabilities) =>
                    capabilities.allows('ORDER_HISTORY_VIEW')
                    ? TextButton(
                        onPressed: () async {
                          final status = await showOrderCancellation(
                            context,
                            order.id,
                          );
                          if (status != null && mounted) {
                            setState(() {
                              _future = Future.value(order.withStatus(status));
                            });
                          }
                        },
                        child: const Text('Yêu cầu hủy đơn'),
                      )
                    : const SizedBox.shrink(),
              ),
            const SizedBox(height: 10),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Giai đoạn theo trạng thái đơn',
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    const SizedBox(height: 14),
                    for (var index = 0; index < _steps(order).length; index++)
                      _TimelineRow(
                        label: _steps(order)[index],
                        completed: index <= _currentStep(order.status),
                        time: order.checkpoints
                            .where(
                              (point) => point.label == _steps(order)[index],
                            )
                            .firstOrNull
                            ?.time,
                      ),
                  ],
                ),
              ),
            ),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Địa chỉ giao hàng',
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 8),
                    if (order.recipientName != null) Text(order.recipientName!),
                    if (order.recipientPhone != null)
                      Text(order.recipientPhone!),
                    Text(order.address),
                  ],
                ),
              ),
            ),
            Card(
              color: AppColors.softYellow,
              child: ListTile(
                title: const Text('Thông tin vận chuyển'),
                subtitle: const Text(
                  'Xem hành trình, mã vận đơn và niêm phong.',
                ),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => context.push(
                  '/orders/${Uri.encodeComponent(order.id)}/tracking',
                ),
              ),
            ),
            if (order.items.isNotEmpty)
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Padding(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 4,
                          vertical: 8,
                        ),
                        child: Text(
                          'Sản phẩm trong đơn',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                      ),
                      for (final item in order.items)
                        ListTile(
                          contentPadding: EdgeInsets.zero,
                          leading: const CircleAvatar(
                            backgroundColor: AppColors.muted,
                            child: Icon(Icons.redeem_outlined),
                          ),
                          title: Text(item.name),
                          subtitle: Text(
                            '${item.variantName} • SL ${item.quantity}',
                          ),
                          trailing: Text(
                            item.unitPriceKnown
                                ? item.lineTotal.format()
                                : item.price.format(),
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            if (order.subtotal != null)
              _AmountRow('Tạm tính', order.subtotal!.format()),
            if (order.discount != null)
              _AmountRow('Ưu đãi', order.discount!.format()),
            if (order.shippingFee != null)
              _AmountRow('Phí vận chuyển', order.shippingFee!.format()),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(AppSpacing.md),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Expanded(
                      child: Text(
                        'Tổng thanh toán',
                        style: TextStyle(fontWeight: FontWeight.bold),
                      ),
                    ),
                    const SizedBox(width: AppSpacing.sm),
                    Flexible(
                      child: Text(
                        order.total.format(),
                        textAlign: TextAlign.end,
                        style: Theme.of(context).textTheme.titleLarge,
                      ),
                    ),
                  ],
                ),
              ),
            ),
            TextButton.icon(
              onPressed: () => context.push(AppRoutes.supportRequest),
              icon: const Icon(Icons.support_agent_outlined),
              label: const Text('Yêu cầu hỗ trợ'),
            ),
            if (order.status == OrderStatus.delivered)
              BlocBuilder<CapabilitiesCubit, CapabilitiesState>(
                builder: (context, capabilities) =>
                    capabilities.allows('ORDER_HISTORY_VIEW')
                    ? TextButton.icon(
                        icon: const Icon(Icons.assignment_return_outlined),
                        label: const Text('Tạo yêu cầu hậu mãi / đổi trả'),
                        onPressed: () => context.push(
                          '/orders/${Uri.encodeComponent(order.id)}/after-sales/request',
                        ),
                      )
                    : const SizedBox.shrink(),
              ),
          ],
        );
      },
    ),
  );
  List<String> _steps(CustomerOrder order) =>
      order.status == OrderStatus.cancelled
      ? ['Đã đặt', 'Đã hủy']
      : ['Đã đặt', 'Đã xác nhận', 'Đang giao', 'Đã giao'];
  int _currentStep(OrderStatus status) => switch (status) {
    OrderStatus.pending => 0,
    OrderStatus.confirmed => 1,
    OrderStatus.shipping => 2,
    OrderStatus.delivered => 3,
    OrderStatus.cancelled => 1,
    OrderStatus.unknown => -1,
  };
}

class _AmountRow extends StatelessWidget {
  const _AmountRow(this.label, this.value);
  final String label, value;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.all(12),
    child: Row(
      children: [
        Expanded(child: Text(label)),
        const SizedBox(width: 12),
        Flexible(child: Text(value, textAlign: TextAlign.end)),
      ],
    ),
  );
}

class _TimelineRow extends StatelessWidget {
  const _TimelineRow({required this.label, required this.completed, this.time});
  final String label;
  final bool completed;
  final DateTime? time;
  @override
  Widget build(BuildContext context) => SizedBox(
    height: 52,
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(
          completed ? Icons.check_circle : Icons.radio_button_unchecked,
          color: completed ? AppColors.honeyGold : AppColors.border,
          size: 20,
        ),
        const SizedBox(width: 12),
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(label, style: const TextStyle(fontWeight: FontWeight.w600)),
            if (time != null)
              Text(
                DateFormat('dd/MM • HH:mm').format(time!),
                style: Theme.of(context).textTheme.bodySmall,
              ),
          ],
        ),
      ],
    ),
  );
}
