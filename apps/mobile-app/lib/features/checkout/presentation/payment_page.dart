import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_snackbar.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';

class PaymentPage extends StatefulWidget {
  const PaymentPage({super.key, required this.orderId, this.confirmation});
  final String orderId;
  final OrderConfirmation? confirmation;
  @override
  State<PaymentPage> createState() => _PaymentPageState();
}

class _PaymentPageState extends State<PaymentPage> {
  Timer? _timer;
  bool _checking = false;
  CustomerOrder? _order;
  String? _error;

  @override
  void initState() {
    super.initState();
    if (widget.confirmation?.paymentExpiresAt != null) {
      _timer = Timer.periodic(const Duration(seconds: 1), (_) {
        if (mounted) setState(() {});
      });
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  Future<void> _check() async {
    if (_checking) return;
    setState(() {
      _checking = true;
      _error = null;
    });
    try {
      final order = await context.read<OrdersRepository>().order(
        widget.orderId,
      );
      if (mounted) setState(() => _order = order);
    } catch (error) {
      if (mounted) {
        setState(
          () => _error = error is ApiException
              ? error.failure.message
              : 'Chưa thể kiểm tra trạng thái. Vui lòng thử lại.',
        );
      }
    } finally {
      if (mounted) setState(() => _checking = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final confirmation = widget.confirmation;
    if (confirmation == null) {
      return Scaffold(
        appBar: const AppMobileHeader(title: 'Thanh toán', showBack: true),
        body: AppEmptyView(
          title: 'Chưa có thông tin thanh toán',
          message: 'Hãy kiểm tra đơn hàng trước khi thực hiện thanh toán.',
          action: AppPrimaryButton(
            label: 'Xem đơn hàng',
            onPressed: () => context.go('/orders/${widget.orderId}'),
          ),
        ),
      );
    }
    final cod = confirmation.paymentMethod == PaymentMethod.cod;
    final expiry = confirmation.paymentExpiresAt;
    final remaining = expiry
        ?.difference(DateTime.now())
        .inSeconds
        .clamp(0, 86400);
    final paid = [
      'PAID',
      'PAYMENT_CONFIRMED',
    ].contains(_order?.rawStatus ?? confirmation.status);
    final cancelled = _order?.status == OrderStatus.cancelled;
    final qr = Uri.tryParse(confirmation.vietQrUrl ?? '');
    final validQr = qr != null && qr.scheme == 'https' && qr.host.isNotEmpty;
    return Scaffold(
      appBar: const AppMobileHeader(
        title: 'Xác nhận & thanh toán',
        showBack: true,
      ),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(20),
          children: [
            Text(
              paid
                  ? 'Thanh toán đã được ghi nhận'
                  : cancelled
                  ? 'Đơn hàng đã hủy'
                  : cod
                  ? 'Đơn COD đã được tiếp nhận'
                  : remaining == 0
                  ? 'Thời hạn thanh toán đã kết thúc'
                  : 'Đang chờ xác minh thanh toán',
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: 12),
            Text(
              cod
                  ? 'Bạn thanh toán khi nhận hàng. Đơn chưa được đánh dấu đã thu tiền.'
                  : 'Chỉ xác nhận thanh toán khi hệ thống ghi nhận thông tin hợp lệ. Không thanh toán lặp khi giao dịch chưa rõ kết quả.',
            ),
            const SizedBox(height: 20),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text('Mã đơn tham chiếu'),
                    SelectableText(
                      confirmation.orderId,
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    const SizedBox(height: 12),
                    const Text('Số tiền theo đơn'),
                    Text(
                      (_order?.total ?? confirmation.total).format(),
                      style: Theme.of(context).textTheme.headlineMedium,
                    ),
                    TextButton.icon(
                      onPressed: () async {
                        await Clipboard.setData(
                          ClipboardData(text: confirmation.orderId),
                        );
                        if (context.mounted) {
                          AppSnackBar.success(context, 'Đã sao chép mã đơn.');
                        }
                      },
                      icon: const Icon(Icons.copy_outlined),
                      label: const Text('Chép mã đơn'),
                    ),
                  ],
                ),
              ),
            ),
            if (!cod && !paid && !cancelled) ...[
              if (remaining != null)
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 12),
                  child: Text(
                    'Thời gian còn lại: ${remaining ~/ 60}:${(remaining % 60).toString().padLeft(2, '0')}',
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                ),
              if (remaining == 0)
                const Text(
                  'Mã đã hết hạn. Kiểm tra trạng thái đơn trước khi thực hiện giao dịch mới.',
                  style: TextStyle(color: AppColors.darkRed),
                )
              else if (validQr)
                Center(
                  child: SizedBox.square(
                    dimension: 260,
                    child: Image.network(
                      qr.toString(),
                      semanticLabel:
                          'Mã VietQR của đơn ${confirmation.orderId}',
                      errorBuilder: (_, _, _) => const Center(
                        child: Text(
                          'Chưa tải được mã QR. Hãy kiểm tra kết nối và thử lại.',
                        ),
                      ),
                    ),
                  ),
                )
              else
                const Padding(
                  padding: EdgeInsets.all(16),
                  child: Text(
                    'Chưa có mã QR khả dụng cho phiên này. Hãy kiểm tra đơn hàng hoặc liên hệ hỗ trợ.',
                  ),
                ),
            ],
            if (_order != null)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 16),
                child: Semantics(
                  liveRegion: true,
                  child: Text(
                    'Trạng thái đơn mới nhất: ${_order!.status.label}',
                  ),
                ),
              ),
            if (_error != null)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 16),
                child: Semantics(
                  liveRegion: true,
                  child: Text(
                    _error!,
                    style: const TextStyle(color: AppColors.darkRed),
                  ),
                ),
              ),
            const SizedBox(height: 20),
            AppPrimaryButton(
              label: 'Kiểm tra trạng thái',
              loading: _checking,
              onPressed: _check,
            ),
            const SizedBox(height: 12),
            OutlinedButton(
              onPressed: () => context.go('/orders/${widget.orderId}'),
              child: const Text('Xem chi tiết đơn'),
            ),
          ],
        ),
      ),
    );
  }
}
