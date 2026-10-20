import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:intl/intl.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/core/widgets/public_https_button.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/orders/presentation/order_cancellation_cubit.dart';

Future<String?> showOrderCancellation(BuildContext context, String orderId) =>
    showDialog<String>(
      context: context,
      barrierDismissible: false,
      builder: (_) => BlocProvider(
        create: (_) =>
            OrderCancellationCubit(context.read<OrdersRepository>(), orderId),
        child: const _CancellationDialog(),
      ),
    );

class _CancellationDialog extends StatefulWidget {
  const _CancellationDialog();
  @override
  State<_CancellationDialog> createState() => _CancellationDialogState();
}

class _CancellationDialogState extends State<_CancellationDialog> {
  final _reason = TextEditingController();
  @override
  void dispose() {
    _reason.dispose();
    super.dispose();
  }

  @override
  Widget build(
    BuildContext context,
  ) => BlocConsumer<OrderCancellationCubit, OrderCancellationState>(
    listener: (context, state) {
      if (state.status != null) Navigator.of(context).pop(state.status);
    },
    builder: (context, state) => PopScope(
      canPop: !state.pending,
      child: AlertDialog(
        title: const Text('Xác nhận hủy đơn'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Text(
                'Máy chủ sẽ kiểm tra điều kiện hủy tại thời điểm gửi. Hủy đơn không đồng nghĩa đã hoàn tiền.',
              ),
              const SizedBox(height: 16),
              AppTextField(
                controller: _reason,
                label: 'Lý do hủy',
                maxLines: 3,
                maxLength: 300,
                enabled: !state.pending,
              ),
              if (state.error != null)
                Semantics(liveRegion: true, child: Text(state.error!)),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: state.pending ? null : () => Navigator.of(context).pop(),
            child: const Text('Giữ đơn'),
          ),
          AppPrimaryButton(
            label: 'Xác nhận hủy',
            loading: state.pending,
            onPressed: () =>
                context.read<OrderCancellationCubit>().cancel(_reason.text),
          ),
        ],
      ),
    ),
  );
}

class OrderTrackingPage extends StatefulWidget {
  const OrderTrackingPage({super.key, required this.orderId});
  final String orderId;
  @override
  State<OrderTrackingPage> createState() => _OrderTrackingPageState();
}

class _OrderTrackingPageState extends State<OrderTrackingPage> {
  late Future<OrderTracking> _future;
  @override
  void initState() {
    super.initState();
    _load();
  }

  void _load() =>
      _future = context.read<OrdersRepository>().tracking(widget.orderId);
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(
      title: 'Hành trình giao nhận',
      showBack: true,
    ),
    body: FutureBuilder<OrderTracking>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done) {
          return const AppLoadingView();
        }
        if (!snapshot.hasData) {
          return AppErrorView(
            message: snapshot.error is ApiException
                ? (snapshot.error as ApiException).failure.message
                : 'Chưa thể tải hành trình giao nhận.',
            onRetry: () => setState(_load),
          );
        }
        final data = snapshot.data!;
        return RefreshIndicator(
          onRefresh: () async {
            setState(_load);
            try {
              await _future;
            } catch (_) {
              /* FutureBuilder presents retry/error state. */
            }
          },
          child: ListView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.all(20),
            children: [
              Text(
                '#${data.orderId}',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 16),
              _TrackingCard(
                title: 'Thông tin vận chuyển',
                children: [
                  Text(
                    data.carrier.isEmpty
                        ? 'Chưa có đơn vị giao nhận'
                        : data.carrier,
                  ),
                  if (data.code.isNotEmpty)
                    SelectableText('Mã vận đơn: ${data.code}'),
                  Text('Trạng thái vận chuyển: ${data.statusLabel}'),
                  if (data.shipper.isNotEmpty)
                    Text('Người giao: ${data.shipper}'),
                  if (data.phone.isNotEmpty)
                    SelectableText('Liên hệ giao nhận: ${data.phone}'),
                ],
              ),
              if (data.seal.isNotEmpty || data.video.isNotEmpty)
                _TrackingCard(
                  title: 'Đóng gói và niêm phong',
                  children: [
                    if (data.seal.isNotEmpty)
                      SelectableText('Mã niêm phong: ${data.seal}'),
                    if (data.video.isNotEmpty)
                      PublicHttpsButton(
                        url: data.video,
                        label: 'Xem video đóng gói',
                      ),
                  ],
                ),
              _TrackingCard(
                title: 'Các mốc công khai',
                children: [
                  if (data.checkpoints.isEmpty)
                    const Text('Chưa có mốc vận chuyển được công bố.'),
                  for (final point in data.checkpoints)
                    Padding(
                      padding: const EdgeInsets.symmetric(vertical: 10),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            DateFormat(
                              'dd/MM/yyyy HH:mm',
                            ).format(point.time.toLocal()),
                            style: Theme.of(context).textTheme.bodySmall,
                          ),
                          Text(
                            point.statusLabel,
                            style: const TextStyle(fontWeight: FontWeight.w600),
                          ),
                          Text(point.location),
                          Text(point.description),
                        ],
                      ),
                    ),
                ],
              ),
            ],
          ),
        );
      },
    ),
  );
}

class _TrackingCard extends StatelessWidget {
  const _TrackingCard({required this.title, required this.children});
  final String title;
  final List<Widget> children;
  @override
  Widget build(BuildContext context) => Card(
    margin: const EdgeInsets.symmetric(vertical: 12),
    child: Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 12),
          ...children,
        ],
      ),
    ),
  );
}
