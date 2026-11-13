import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/features/checkout/presentation/mvp_payment_cubit.dart';

class MvpPaymentPage extends StatefulWidget {
  const MvpPaymentPage({super.key, this.local = false});
  final bool local;
  @override
  State<MvpPaymentPage> createState() => _MvpPaymentPageState();
}

class _MvpPaymentPageState extends State<MvpPaymentPage>
    with WidgetsBindingObserver {
  Timer? _timer;
  int _polls = 0;
  bool _foreground = true;
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _foreground =
        WidgetsBinding.instance.lifecycleState == null ||
        WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed;
    _timer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (!_foreground) return;
      final cubit = context.read<MvpPaymentCubit>();
      if (_polls < 20 && cubit.shouldPoll) {
        _polls++;
        unawaited(cubit.load());
      }
      if (mounted) setState(() {});
    });
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) =>
      _foreground = state == AppLifecycleState.resumed;
  @override
  void dispose() {
    _timer?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Thanh toán', showBack: true),
    body: SafeArea(
      child: BlocBuilder<MvpPaymentCubit, MvpPaymentState>(
        builder: (context, state) {
          final cubit = context.read<MvpPaymentCubit>(),
              status = state.status,
              instructions = state.instructions;
          if (state.loading && status == null) return const AppLoadingView();
          return ListView(
            padding: const EdgeInsets.all(20),
            children: [
              if (widget.local)
                const Text(
                  'Thanh toán thử nghiệm local: chưa có tài khoản nhận tiền hoặc mã QR thật.',
                ),
              const SizedBox(height: 16),
              Semantics(
                liveRegion: true,
                child: Text(
                  status?.label ?? 'Chưa có thông tin thanh toán',
                  style: Theme.of(context).textTheme.headlineMedium,
                ),
              ),
              if (status != null) ...[
                const SizedBox(height: 16),
                Text('Giá trị cần thanh toán: ${status.expected.format()}'),
                Text('Tiền đã ghi nhận cho đơn: ${status.allocated.format()}'),
                Text('Tiền đã hoàn: ${status.refunded.format()}'),
                if (const {
                  'VERIFYING',
                  'VERIFICATION_DELAYED',
                  'FINALIZING',
                }.contains(status.displayState))
                  const Padding(
                    padding: EdgeInsets.symmetric(vertical: 12),
                    child: Text(
                      'Hệ thống đang xác minh hoặc hoàn tất giao dịch. Không chuyển thêm tiền trong lúc chờ.',
                    ),
                  ),
                if (status.reconciliation ||
                    status.displayState == 'RECONCILIATION_REQUIRED')
                  const Text(
                    'Có thông tin cần đối soát. Liên hệ hỗ trợ trước khi thực hiện giao dịch khác.',
                  ),
                if (status.supportReference != null)
                  Text('Mã hỗ trợ: ${status.supportReference}'),
                if (status.verificationDueAt != null &&
                    const {
                      'VERIFYING',
                      'VERIFICATION_DELAYED',
                    }.contains(status.displayState))
                  Text(
                    'Mốc kiểm tra: ${DateFormat('HH:mm:ss dd/MM/yyyy').format(status.verificationDueAt!.toLocal())}',
                  ),
              ],
              if (state.message != null) ...[
                const SizedBox(height: 16),
                Text(state.message!),
              ],
              if (cubit.instructionsUsable && instructions != null) ...[
                const SizedBox(height: 20),
                Text(
                  instructions.amount.format(),
                  style: Theme.of(context).textTheme.headlineMedium,
                ),
                const SizedBox(height: 12),
                Image.network(
                  instructions.qrUrl.toString(),
                  height: 230,
                  errorBuilder: (_, _, _) => const Text(
                    'Chưa thể tải QR. Dùng đúng thông tin chuyển khoản bên dưới.',
                  ),
                ),
                SelectableText(
                  '${instructions.bankName}\n${instructions.account}\n${instructions.accountHolder}',
                ),
                const SizedBox(height: 12),
                const Text('Nội dung chuyển khoản'),
                SelectableText(instructions.transferContent),
                TextButton(
                  onPressed: () {
                    if (cubit.instructionsUsable) {
                      Clipboard.setData(
                        ClipboardData(text: instructions.transferContent),
                      );
                    }
                  },
                  child: const Text('Sao chép nội dung'),
                ),
                Text(
                  'Hạn thanh toán: ${DateFormat('HH:mm:ss dd/MM/yyyy').format(instructions.expiresAt.toLocal())}',
                ),
              ] else if (!state.loading &&
                  status?.displayState == 'WAITING_PAYMENT') ...[
                const SizedBox(height: 12),
                const Text(
                  'Hướng dẫn chuyển khoản hiện chưa khả dụng hoặc đã gần hết hạn. Kiểm tra lại trước khi chuyển tiền.',
                ),
              ],
              const SizedBox(height: 24),
              AppPrimaryButton(
                label: 'Kiểm tra trạng thái thanh toán',
                loading: state.loading,
                onPressed: cubit.load,
              ),
              const SizedBox(height: 12),
              OutlinedButton(
                onPressed: () => context.go('/orders/${cubit.orderId}'),
                child: const Text('Xem đơn hàng'),
              ),
              TextButton(
                onPressed: () => context.push(AppRoutes.supportRequest),
                child: const Text('Liên hệ hỗ trợ'),
              ),
            ],
          );
        },
      ),
    ),
  );
}
