import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/config/app_config.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';
import 'package:mobile_app/features/auth/presentation/auth_return_location.dart';
import 'package:mobile_app/features/auth/presentation/password_checklist.dart';
import 'package:mobile_app/features/auth/presentation/recovery_cubit.dart';

class RecoveryPage extends StatefulWidget {
  const RecoveryPage({
    super.key,
    required this.returnLocation,
    this.recoveryToken,
  });
  final String returnLocation;
  final String? recoveryToken;
  @override
  State<RecoveryPage> createState() => _RecoveryPageState();
}

class _RecoveryPageState extends State<RecoveryPage> {
  final _email = TextEditingController();
  final _password = TextEditingController();
  final _confirmation = TextEditingController();
  bool _visible = false;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    if (widget.recoveryToken case final String token when token.isNotEmpty) {
      context.read<RecoveryCubit>().verify(token);
    }
  }

  void _listen(BuildContext context, RecoveryState state) {
    _timer?.cancel();
    if (state.retryAt != null) {
      _timer = Timer.periodic(const Duration(seconds: 1), (timer) {
        if (!mounted) {
          timer.cancel();
          return;
        }
        setState(() {});
        if (!state.retryAt!.isAfter(DateTime.now())) timer.cancel();
      });
    }
    if (state.phase == RecoveryPhase.success) {
      _password.clear();
      _confirmation.clear();
      context.read<AuthCubit>().resetToGuest();
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    _email.dispose();
    _password.dispose();
    _confirmation.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppMobileHeader(
      title: 'Khôi phục mật khẩu',
      showBack: true,
      onBack: () =>
          context.go(authLocation(AppRoutes.login, widget.returnLocation)),
    ),
    body: SafeArea(
      child: BlocConsumer<RecoveryCubit, RecoveryState>(
        listener: _listen,
        builder: (context, state) {
          final cubit = context.read<RecoveryCubit>();
          final seconds = state.retryAt == null
              ? 0
              : state.retryAt!.difference(DateTime.now()).inSeconds + 1;
          return ListView(
            padding: const EdgeInsets.all(24),
            children: [
              const SizedBox(height: 24),
              Icon(
                state.phase == RecoveryPhase.success
                    ? Icons.check_circle_outline
                    : Icons.mark_email_read_outlined,
                size: 52,
                color: AppColors.headingBrown,
              ),
              const SizedBox(height: 24),
              Text(switch (state.phase) {
                RecoveryPhase.sent => 'Hãy kiểm tra email',
                RecoveryPhase.verifying => 'Đang kiểm tra liên kết…',
                RecoveryPhase.reset => 'Tạo mật khẩu mới',
                RecoveryPhase.success => 'Mật khẩu đã được cập nhật',
                _ => 'Quên mật khẩu?',
              }, style: Theme.of(context).textTheme.headlineMedium),
              const SizedBox(height: 12),
              if (state.message != null) ...[
                Semantics(
                  liveRegion: true,
                  child: Container(
                    padding: const EdgeInsets.all(14),
                    decoration: BoxDecoration(
                      color: AppColors.errorPale,
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(
                      state.message!,
                      style: const TextStyle(color: AppColors.darkRed),
                    ),
                  ),
                ),
                const SizedBox(height: 16),
              ],
              if (seconds > 0)
                Padding(
                  padding: const EdgeInsets.only(bottom: 16),
                  child: Text('Vui lòng chờ $seconds giây trước khi thử lại.'),
                ),
              if (state.phase == RecoveryPhase.request) ...[
                const Text(
                  'Nhập email đã đăng ký để nhận hướng dẫn khôi phục tài khoản.',
                ),
                const SizedBox(height: 20),
                AppTextField(
                  controller: _email,
                  label: 'Email đã đăng ký',
                  enabled: !state.pending,
                  keyboardType: TextInputType.emailAddress,
                  autofillHints: const [AutofillHints.email],
                  textInputAction: TextInputAction.done,
                  onSubmitted: (_) => cubit.request(_email.text),
                  errorText: state.fieldErrors['email'],
                ),
                const SizedBox(height: 12),
                const Text(
                  'Thông báo được giữ giống nhau để bảo vệ thông tin tài khoản.',
                  style: TextStyle(color: AppColors.textMuted, fontSize: 12),
                ),
                const SizedBox(height: 24),
                AppPrimaryButton(
                  label: 'Gửi liên kết khôi phục',
                  loading: state.pending,
                  onPressed: cubit.canSubmit
                      ? () => cubit.request(_email.text)
                      : null,
                ),
              ],
              if (state.phase == RecoveryPhase.sent) ...[
                Text(
                  'Nếu ${state.email} phù hợp với tài khoản, hướng dẫn khôi phục sẽ được gửi đến địa chỉ đó.',
                ),
                const SizedBox(height: 16),
                const Text(
                  'Thông báo này không xác nhận tài khoản có tồn tại hay không.',
                ),
                const SizedBox(height: 24),
                OutlinedButton(
                  onPressed: cubit.newRequest,
                  child: const Text('Gửi một yêu cầu khác'),
                ),
                if (AppConfig.dataSourceMode == DataSourceMode.mock) ...[
                  const SizedBox(height: 16),
                  const Text('Chế độ thử nghiệm: không gửi email thật.'),
                  TextButton(
                    onPressed: () => cubit.verify('mock-token'),
                    child: const Text('Mở liên kết thử nghiệm'),
                  ),
                ],
              ],
              if (state.phase == RecoveryPhase.verifying)
                const Padding(
                  padding: EdgeInsets.all(24),
                  child: Center(child: CircularProgressIndicator()),
                ),
              if (state.phase == RecoveryPhase.reset) ...[
                const Text(
                  'Liên kết hợp lệ. Hãy đặt mật khẩu mới cho tài khoản.',
                ),
                const SizedBox(height: 20),
                AppTextField(
                  controller: _password,
                  label: 'Mật khẩu mới',
                  obscureText: !_visible,
                  enabled: !state.pending,
                  autofillHints: const [AutofillHints.newPassword],
                  onChanged: (_) => setState(() {}),
                  errorText: state.fieldErrors['newPassword'],
                  suffixIcon: IconButton(
                    tooltip: _visible ? 'Ẩn mật khẩu' : 'Hiện mật khẩu',
                    onPressed: () => setState(() => _visible = !_visible),
                    icon: Icon(
                      _visible ? Icons.visibility_off : Icons.visibility,
                    ),
                  ),
                ),
                const SizedBox(height: 12),
                PasswordChecklist(password: _password.text),
                const SizedBox(height: 16),
                AppTextField(
                  controller: _confirmation,
                  label: 'Xác nhận mật khẩu mới',
                  enabled: !state.pending,
                  obscureText: !_visible,
                  errorText: state.fieldErrors['newPasswordConfirmation'],
                  textInputAction: TextInputAction.done,
                  onSubmitted: (_) =>
                      cubit.reset(_password.text, _confirmation.text),
                ),
                const SizedBox(height: 24),
                AppPrimaryButton(
                  label: 'Đặt lại mật khẩu',
                  loading: state.pending,
                  onPressed: cubit.canSubmit
                      ? () => cubit.reset(_password.text, _confirmation.text)
                      : null,
                ),
              ],
              if (state.phase == RecoveryPhase.success) ...[
                const Text(
                  'Tất cả phiên cũ được thu hồi bởi hệ thống. Hãy đăng nhập lại bằng mật khẩu mới.',
                ),
                const SizedBox(height: 24),
                AppPrimaryButton(
                  label: 'Quay lại đăng nhập',
                  onPressed: () => context.go(
                    authLocation(AppRoutes.login, widget.returnLocation),
                  ),
                ),
              ],
              if (state.phase != RecoveryPhase.success)
                TextButton(
                  onPressed: state.pending
                      ? null
                      : () => context.go(
                          authLocation(AppRoutes.login, widget.returnLocation),
                        ),
                  child: const Text('Quay lại đăng nhập Tri Kỷ'),
                ),
            ],
          );
        },
      ),
    ),
  );
}
