import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/core/config/app_config.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/features/orders/presentation/guest_lookup_cubit.dart';

class GuestLookupPage extends StatefulWidget {
  const GuestLookupPage({super.key});
  @override
  State<GuestLookupPage> createState() => _GuestLookupPageState();
}

class _GuestLookupPageState extends State<GuestLookupPage> {
  final _order = TextEditingController();
  final _contact = TextEditingController();
  final _otp = TextEditingController();
  Timer? _timer;
  @override
  void dispose() {
    _timer?.cancel();
    _order.dispose();
    _contact.dispose();
    _otp.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Tra cứu đơn khách', showBack: true),
    body: SafeArea(
      child: BlocConsumer<GuestLookupCubit, GuestLookupState>(
        listener: (context, state) {
          _timer?.cancel();
          if (state.retryAt != null && state.retryAt!.isAfter(DateTime.now())) {
            _timer = Timer.periodic(const Duration(seconds: 1), (timer) {
              if (!mounted) {
                timer.cancel();
                return;
              }
              setState(() {});
              if (!state.retryAt!.isAfter(DateTime.now())) timer.cancel();
            });
          }
          if (state.access != null) {
            context.go('/orders/${Uri.encodeComponent(state.access!.orderId)}');
          }
        },
        builder: (context, state) {
          final cubit = context.read<GuestLookupCubit>();
          final seconds = state.retryAt == null
              ? 0
              : state.retryAt!.difference(DateTime.now()).inSeconds + 1;
          return ListView(
            padding: const EdgeInsets.all(24),
            children: [
              Text(
                'Xác minh quyền sở hữu đơn',
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: 12),
              const Text(
                'Mã đơn đơn lẻ không đủ để xem thông tin. Hãy xác minh qua liên hệ đã dùng đặt hàng.',
              ),
              const SizedBox(height: 24),
              if (state.error != null)
                Padding(
                  padding: const EdgeInsets.only(bottom: 16),
                  child: Semantics(
                    liveRegion: true,
                    child: Text(
                      state.error!,
                      style: const TextStyle(color: AppColors.darkRed),
                    ),
                  ),
                ),
              if (state.challenge == null) ...[
                AppTextField(
                  controller: _order,
                  label: 'Mã đơn hàng',
                  enabled: !state.pending,
                ),
                const SizedBox(height: 16),
                AppTextField(
                  controller: _contact,
                  label: 'Số điện thoại hoặc email đặt hàng',
                  enabled: !state.pending,
                ),
                const SizedBox(height: 24),
                AppPrimaryButton(
                  label: seconds > 0 ? 'Chờ $seconds giây' : 'Gửi mã xác minh',
                  loading: state.pending,
                  onPressed: cubit.canRequest
                      ? () => cubit.request(_order.text, _contact.text)
                      : null,
                ),
              ] else ...[
                Text(state.challenge!.message),
                const SizedBox(height: 8),
                Text(state.challenge!.maskedDestination),
                const SizedBox(height: 20),
                AppTextField(
                  controller: _otp,
                  label: 'Mã xác minh 6 chữ số',
                  enabled: !state.pending,
                  inputFormatters: [
                    FilteringTextInputFormatter.digitsOnly,
                    LengthLimitingTextInputFormatter(6),
                  ],
                  keyboardType: TextInputType.number,
                  autofillHints: const [AutofillHints.oneTimeCode],
                  textInputAction: TextInputAction.done,
                  onSubmitted: (_) => cubit.verify(_otp.text),
                ),
                const SizedBox(height: 24),
                AppPrimaryButton(
                  label: state.verificationThrottled && seconds > 0
                      ? 'Chờ $seconds giây'
                      : 'Xác minh & mở đơn',
                  loading: state.pending,
                  onPressed: state.verificationThrottled && seconds > 0
                      ? null
                      : () => cubit.verify(_otp.text),
                ),
                TextButton(
                  onPressed: cubit.canRequest
                      ? () {
                          _otp.clear();
                          cubit.resend(_order.text, _contact.text);
                        }
                      : null,
                  child: Text(
                    seconds > 0 ? 'Gửi lại sau $seconds giây' : 'Gửi lại mã',
                  ),
                ),
                TextButton(
                  onPressed: state.pending
                      ? null
                      : () {
                          _otp.clear();
                          cubit.useOtherInformation();
                        },
                  child: const Text('Dùng thông tin khác'),
                ),
              ],
              if (AppConfig.dataSourceMode == DataSourceMode.mock)
                const Padding(
                  padding: EdgeInsets.only(top: 20),
                  child: Text(
                    'Thử nghiệm local: đơn OM-8921, liên hệ 0905123456, mã 789214. Không gửi SMS/email thật.',
                    style: TextStyle(fontSize: 12, color: AppColors.textMuted),
                  ),
                ),
            ],
          );
        },
      ),
    ),
  );
}
