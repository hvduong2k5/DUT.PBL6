import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/theme/app_radius.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';

class AuthPage extends StatefulWidget {
  const AuthPage({super.key, required this.register});
  final bool register;
  @override
  State<AuthPage> createState() => _AuthPageState();
}

class _AuthPageState extends State<AuthPage> {
  final _name = TextEditingController();
  final _identity = TextEditingController();
  final _email = TextEditingController();
  final _password = TextEditingController();
  final _confirmation = TextEditingController();
  bool _showPassword = false;
  bool _submitted = false;

  String? _required(String value) =>
      _submitted && value.trim().isEmpty ? 'Bắt buộc' : null;
  String? get _phoneError {
    if (!_submitted) return null;
    if (_identity.text.trim().isEmpty) return 'Bắt buộc';
    return RegExp(r'^0\d{9}$').hasMatch(_identity.text.trim())
        ? null
        : 'Không hợp lệ';
  }

  String? get _passwordError {
    if (!_submitted) return null;
    if (_password.text.length < 8) return 'Tối thiểu 8 ký tự';
    if (widget.register && _confirmation.text != _password.text) {
      return 'Mật khẩu không khớp';
    }
    return null;
  }

  Future<void> _submit() async {
    setState(() => _submitted = true);
    if (_phoneError != null ||
        _passwordError != null ||
        (widget.register && _name.text.trim().isEmpty)) {
      return;
    }
    final cubit = context.read<AuthCubit>();
    final success = widget.register
        ? await cubit.register(
            name: _name.text,
            phone: _identity.text,
            password: _password.text,
            email: _email.text,
          )
        : await cubit.login(_identity.text, _password.text);
    if (success && mounted) context.go(AppRoutes.account);
  }

  @override
  void dispose() {
    for (final controller in [
      _name,
      _identity,
      _email,
      _password,
      _confirmation,
    ]) {
      controller.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppMobileHeader(
      title: widget.register ? 'Đăng Ký' : 'Đăng Nhập',
      showBack: true,
    ),
    body: SafeArea(
      child: BlocBuilder<AuthCubit, AuthState>(
        builder: (context, state) => ListView(
          padding: const EdgeInsets.all(AppSpacing.lg),
          children: [
            const SizedBox(height: AppSpacing.xl),
            const CircleAvatar(
              radius: 44,
              backgroundColor: AppColors.cream,
              child: Icon(
                Icons.person_outline,
                size: 32,
                color: AppColors.brandBrown,
              ),
            ),
            const SizedBox(height: AppSpacing.md),
            Text(
              'Ô Mạ',
              style: Theme.of(context).textTheme.headlineMedium,
              textAlign: TextAlign.center,
            ),
            Text(
              widget.register
                  ? 'Tạo tài khoản'
                  : 'Đăng nhập để quản lý đơn hàng, địa chỉ và thông báo.',
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: AppSpacing.xl),
            if (state.status == AuthStatus.failure)
              Container(
                margin: const EdgeInsets.only(bottom: AppSpacing.md),
                padding: const EdgeInsets.all(AppSpacing.sm),
                decoration: BoxDecoration(
                  color: AppColors.alertPink,
                  border: Border.all(color: AppColors.darkRed),
                  borderRadius: BorderRadius.circular(AppRadius.large),
                ),
                child: Text(
                  state.message ?? 'Có lỗi xảy ra.',
                  style: const TextStyle(color: AppColors.darkRed),
                ),
              ),
            if (widget.register) ...[
              AppTextField(
                controller: _name,
                label: 'Họ và tên',
                errorText: _required(_name.text),
              ),
              const SizedBox(height: AppSpacing.sm),
              AppTextField(
                controller: _email,
                label: 'Email (không bắt buộc)',
                keyboardType: TextInputType.emailAddress,
              ),
              const SizedBox(height: AppSpacing.sm),
            ],
            AppTextField(
              controller: _identity,
              label: 'Số điện thoại',
              keyboardType: TextInputType.phone,
              errorText: _phoneError,
            ),
            const SizedBox(height: AppSpacing.sm),
            AppTextField(
              controller: _password,
              label: 'Mật khẩu',
              obscureText: !_showPassword,
              prefixIcon: Icons.lock_outline,
              errorText: _passwordError,
              suffixIcon: IconButton(
                constraints: const BoxConstraints(minWidth: 44, minHeight: 44),
                onPressed: () => setState(() => _showPassword = !_showPassword),
                icon: Icon(
                  _showPassword ? Icons.visibility_off : Icons.visibility,
                ),
              ),
            ),
            if (widget.register) ...[
              const SizedBox(height: AppSpacing.sm),
              AppTextField(
                controller: _confirmation,
                label: 'Xác nhận mật khẩu',
                obscureText: !_showPassword,
                errorText: _submitted && _confirmation.text != _password.text
                    ? 'Mật khẩu không khớp'
                    : null,
              ),
            ],
            const SizedBox(height: AppSpacing.lg),
            AppPrimaryButton(
              label: widget.register ? 'Đăng ký' : 'Đăng nhập',
              loading: state.status == AuthStatus.submitting,
              onPressed: _submit,
            ),
            const SizedBox(height: AppSpacing.sm),
            TextButton(
              onPressed: () => context.go(
                widget.register ? AppRoutes.login : AppRoutes.register,
              ),
              child: Text(
                widget.register
                    ? 'Đã có tài khoản? Đăng nhập'
                    : 'Chưa có tài khoản? Đăng ký',
              ),
            ),
            if (!widget.register)
              const Text(
                'Mua hàng với tư cách khách vẫn được hỗ trợ.',
                textAlign: TextAlign.center,
                style: TextStyle(fontSize: 11, color: AppColors.textMuted),
              ),
          ],
        ),
      ),
    ),
  );
}
