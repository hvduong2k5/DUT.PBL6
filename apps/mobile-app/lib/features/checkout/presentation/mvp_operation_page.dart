import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/checkout/presentation/mvp_checkout_cubit.dart';

class MvpOperationPage extends StatefulWidget {
  const MvpOperationPage({super.key, this.localPreview = false});
  final bool localPreview;
  @override
  State<MvpOperationPage> createState() => _MvpOperationPageState();
}

class _MvpOperationPageState extends State<MvpOperationPage>
    with WidgetsBindingObserver {
  Timer? _timer;
  bool _foreground = true;
  int _polls = 0;
  String? _reloaded;
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _foreground =
        WidgetsBinding.instance.lifecycleState == null ||
        WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed;
    _timer = Timer.periodic(const Duration(seconds: 3), (_) {
      if (!_foreground || _polls >= 20) return;
      final controller = context.read<MvpCheckoutCubit>();
      if (!controller.state.busy &&
          controller.state.message == null &&
          (controller.state.operation?.automaticPoll ?? false)) {
        _polls++;
        unawaited(controller.recover(automatic: true));
      }
    });
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _reloadCart(context.read<MvpCheckoutCubit>().state);
    });
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) =>
      _foreground = state == AppLifecycleState.resumed;
  void _reloadCart(MvpCheckoutState state) {
    final operation = state.operation;
    if (operation?.status == 'SUCCEEDED' && _reloaded != operation!.id) {
      _reloaded = operation.id;
      unawaited(context.read<CartCubit>().load());
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Yêu cầu đặt hàng', showBack: true),
    body: SafeArea(
      child: BlocConsumer<MvpCheckoutCubit, MvpCheckoutState>(
        listener: (_, state) => _reloadCart(state),
        builder: (context, state) {
          if (!state.ready) return const AppLoadingView();
          final operation = state.operation;
          final status = operation?.status;
          final title = switch (status) {
            'SUCCEEDED' => 'Đã tạo đơn hàng',
            'FAILED' => 'Chưa thể tạo đơn',
            'MANUAL_REVIEW' => 'Yêu cầu cần được kiểm tra',
            'WAITING_RETRY' => 'Hệ thống đang chờ xử lý lại',
            'COMPENSATING' => 'Đang hoàn tất xử lý yêu cầu',
            'ACCEPTED' || 'PROCESSING' => 'Đang tạo đơn hàng',
            _ =>
              state.attempt == null
                  ? 'Chưa có yêu cầu đặt hàng'
                  : 'Đang chờ xác nhận kết quả',
          };
          return ListView(
            padding: const EdgeInsets.all(20),
            children: [
              if (widget.localPreview)
                const Text(
                  'Đơn thử nghiệm local. Không có giao dịch tiền thật. Dữ liệu chỉ giữ trong phiên chạy app.',
                ),
              const SizedBox(height: 20),
              Semantics(
                liveRegion: true,
                child: Text(
                  title,
                  style: Theme.of(context).textTheme.headlineMedium,
                ),
              ),
              const SizedBox(height: 12),
              Text(
                status == 'SUCCEEDED'
                    ? 'Đơn đã được ghi nhận. Trạng thái thanh toán sẽ được kiểm tra riêng.'
                    : status == 'FAILED'
                    ? 'Yêu cầu này đã kết thúc không thành công. Bạn có thể xem lại giỏ và lấy báo giá mới.'
                    : status == 'MANUAL_REVIEW'
                    ? 'Chưa thể kết luận kết quả. Giữ mã yêu cầu và liên hệ hỗ trợ; không đặt lại đơn này.'
                    : state.attempt == null
                    ? 'Quay lại giỏ để kiểm tra và xác nhận báo giá.'
                    : 'Bạn có thể rời màn hình và quay lại kiểm tra. Quay lại không hủy yêu cầu; không gửi thêm đơn trong lúc chờ.',
              ),
              if (operation != null) ...[
                const SizedBox(height: 20),
                const Text('Mã yêu cầu'),
                SelectableText(operation.id),
                if (operation.orderId != null) ...[
                  const SizedBox(height: 12),
                  const Text('Mã tham chiếu đơn'),
                  SelectableText(operation.orderId!),
                ],
                if (status == 'FAILED') Text('Mã lỗi: ${operation.errorCode}'),
              ],
              if (state.message != null) ...[
                const SizedBox(height: 16),
                Semantics(liveRegion: true, child: Text(state.message!)),
              ],
              const SizedBox(height: 24),
              if (state.attempt != null && !(operation?.terminal ?? false) ||
                  state.storageBlocked)
                AppPrimaryButton(
                  label: state.storageBlocked
                      ? 'Khôi phục yêu cầu'
                      : 'Kiểm tra lại yêu cầu',
                  loading: state.busy,
                  onPressed: () => context.read<MvpCheckoutCubit>().recover(),
                ),
              if (status == 'SUCCEEDED')
                const Text(
                  'Thông tin thanh toán đang chờ bước tích hợp tiếp theo.',
                ),
              const SizedBox(height: 12),
              OutlinedButton(
                onPressed: () => context.go(AppRoutes.cart),
                child: const Text('Về giỏ hàng'),
              ),
              if (status == 'MANUAL_REVIEW' || state.message != null)
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
