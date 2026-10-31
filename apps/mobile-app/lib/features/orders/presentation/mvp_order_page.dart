import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/features/orders/presentation/mvp_order_cubit.dart';

class MvpOrderPage extends StatelessWidget {
  const MvpOrderPage({super.key, this.local = false});
  final bool local;
  Future<void> _cancel(BuildContext context) async {
    final reason = await showDialog<String>(
      context: context,
      builder: (_) => const _CancelDialog(),
    );
    if (context.mounted && reason != null) {
      await context.read<MvpOrderCubit>().cancel(reason);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Chi tiết đơn hàng', showBack: true),
    body: SafeArea(
      child: BlocBuilder<MvpOrderCubit, MvpOrderState>(
        builder: (context, state) {
          final cubit = context.read<MvpOrderCubit>(), order = state.order;
          if (state.loading && order == null) return const AppLoadingView();
          if (order == null) {
            return AppErrorView(
              message: state.message ?? 'Chưa thể tải đơn hàng.',
              onRetry: cubit.load,
            );
          }
          final canAct = !state.busy && !state.loading && state.message == null;
          return RefreshIndicator(
            onRefresh: cubit.load,
            child: ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.all(16),
              children: [
                if (local)
                  const Text(
                    'Đơn thử nghiệm local. Không có giao dịch tiền thật.',
                  ),
                Text(
                  order.code,
                  style: Theme.of(context).textTheme.headlineMedium,
                ),
                Semantics(
                  liveRegion: true,
                  child: Text(
                    order.statusLabel,
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                ),
                const SizedBox(height: 16),
                Text(
                  'Tình trạng tiền: ${const {'UNPAID': 'Chưa ghi nhận thanh toán', 'PENDING': 'Đang chờ xử lý', 'CONFIRMED': 'Đã ghi nhận tiền', 'RECONCILIATION_REQUIRED': 'Cần đối soát', 'PARTIALLY_REFUNDED': 'Đã hoàn một phần', 'REFUNDED': 'Đã hoàn tiền'}[order.paymentStatus] ?? 'Đang cập nhật'}',
                ),
                if (order.cancellationStatus != 'NONE') ...[
                  const SizedBox(height: 12),
                  Text(
                    'Yêu cầu hủy: ${const {'PENDING': 'Đang xử lý', 'SUCCEEDED': 'Đã xác nhận', 'REJECTED': 'Không được chấp nhận', 'MANUAL_REVIEW': 'Cần kiểm tra'}[order.cancellationStatus] ?? 'Đang cập nhật'}',
                  ),
                  if (order.compensationStatus == 'PENDING')
                    const Text(
                      'Đơn đang được hoàn tất xử lý. Xem trạng thái tiền riêng nếu bạn đã thanh toán.',
                    ),
                  if (order.compensationStatus == 'MANUAL_REVIEW')
                    const Text('Một phần xử lý cần được hỗ trợ kiểm tra.'),
                ],
                if (state.command != null && state.command!.result == null)
                  const Text(
                    'Yêu cầu hủy chưa có kết quả xác nhận. Vui lòng kiểm tra lại yêu cầu cũ.',
                  ),
                if (state.message != null) ...[
                  const SizedBox(height: 12),
                  Semantics(liveRegion: true, child: Text(state.message!)),
                ],
                if (state.loading || state.busy)
                  const LinearProgressIndicator(),
                const SizedBox(height: 20),
                Text('Sản phẩm', style: Theme.of(context).textTheme.titleLarge),
                if (order.snapshot == null)
                  const Text('Chưa có chi tiết sản phẩm.')
                else ...[
                  for (final item in order.snapshot!.items)
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(16),
                        child: Text(
                          '${item.name ?? item.sku}\nSố lượng: ${item.quantity}\n${item.unitPrice == null ? 'Chưa có đơn giá chi tiết' : 'Đơn giá: ${item.unitPrice!.format()}'}',
                        ),
                      ),
                    ),
                  const SizedBox(height: 16),
                  Text(
                    'Người nhận',
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                  Text(
                    '${order.snapshot!.address.recipient}\n${order.snapshot!.address.phone}\n${order.snapshot!.address.street}',
                  ),
                ],
                const SizedBox(height: 20),
                Text('Tiền hàng: ${order.subtotal.format()}'),
                Text('Phí giao hàng: ${order.shippingFee.format()}'),
                Text(
                  'Tổng đơn: ${order.total.format()}',
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                const SizedBox(height: 20),
                if (order.canPay)
                  AppPrimaryButton(
                    label: 'Xem thanh toán',
                    onPressed: canAct
                        ? () => context.push('/payment/${order.id}')
                        : null,
                  ),
                if (state.canCancel)
                  OutlinedButton(
                    onPressed: () => _cancel(context),
                    child: const Text('Yêu cầu hủy đơn'),
                  ),
                if (state.command != null || state.storageBlocked)
                  OutlinedButton(
                    onPressed: state.busy || state.loading
                        ? null
                        : cubit.recoverCancellation,
                    child: const Text('Kiểm tra yêu cầu hủy'),
                  ),
                if (order.actions.contains('REQUEST_CANCELLATION'))
                  const Text(
                    'Yêu cầu hủy đơn đã thanh toán cần được bộ phận hỗ trợ xem xét.',
                  ),
                if (order.actions.contains('VIEW_TRACKING')) ...[
                  const SizedBox(height: 16),
                  const Text('Chưa có thông tin vận chuyển chi tiết.'),
                  if (order.shipmentId != null)
                    Text('Mã tham chiếu vận chuyển: ${order.shipmentId}'),
                ],
                if (order.actions.contains('CONTACT_SUPPORT') ||
                    order.actions.contains('REQUEST_CANCELLATION') ||
                    state.message != null)
                  TextButton(
                    onPressed: () => context.push(AppRoutes.supportRequest),
                    child: const Text('Liên hệ hỗ trợ'),
                  ),
                TextButton(
                  onPressed: state.busy ? null : cubit.load,
                  child: const Text('Tải lại đơn hàng'),
                ),
                OutlinedButton(
                  onPressed: () => context.go(AppRoutes.orders),
                  child: const Text('Danh sách đơn hàng'),
                ),
              ],
            ),
          );
        },
      ),
    ),
  );
}

class _CancelDialog extends StatefulWidget {
  const _CancelDialog();
  @override
  State<_CancelDialog> createState() => _CancelDialogState();
}

class _CancelDialogState extends State<_CancelDialog> {
  final _reason = TextEditingController();
  @override
  void dispose() {
    _reason.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
    title: const Text('Yêu cầu hủy đơn'),
    content: SingleChildScrollView(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Text(
            'Kết quả hủy sẽ được xác nhận sau khi hệ thống xử lý. Hủy đơn không đồng nghĩa đã hoàn tiền.',
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _reason,
            maxLength: 255,
            minLines: 2,
            maxLines: 4,
            decoration: const InputDecoration(
              labelText: 'Lý do hủy',
              hintText: 'Nhập lý do của bạn',
            ),
          ),
        ],
      ),
    ),
    actions: [
      TextButton(
        onPressed: () => Navigator.pop(context),
        child: const Text('Giữ đơn'),
      ),
      ValueListenableBuilder(
        valueListenable: _reason,
        builder: (_, value, _) => FilledButton(
          onPressed: value.text.trim().isEmpty
              ? null
              : () => Navigator.pop(context, value.text.trim()),
          child: const Text('Gửi yêu cầu hủy'),
        ),
      ),
    ],
  );
}
