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
            message: 'Không thể tải chi tiết đơn hàng.',
            onRetry: () => setState(_load),
          );
        }
        final order = snapshot.data!;
        return ListView(
          padding: const EdgeInsets.all(16),
          children: [
            Align(
              alignment: Alignment.centerLeft,
              child: Chip(label: Text(order.status.label)),
            ),
            const SizedBox(height: 10),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Tiến trình đơn hàng',
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
                    Text(order.address),
                  ],
                ),
              ),
            ),
            Card(
              color: AppColors.softYellow,
              child: const ListTile(
                title: Text('Thông tin vận chuyển'),
                subtitle: Text(
                  'Đơn vị và mã theo dõi chỉ hiển thị khi API cung cấp.',
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
                          trailing: Text(item.lineTotal.format()),
                        ),
                    ],
                  ),
                ),
              ),
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
            const Padding(
              padding: EdgeInsets.all(12),
              child: Text(
                'Hành động hủy đơn hoặc hỗ trợ chỉ xuất hiện khi API cho phép.',
                textAlign: TextAlign.center,
                style: TextStyle(fontSize: 11, color: AppColors.textMuted),
              ),
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
  };
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
