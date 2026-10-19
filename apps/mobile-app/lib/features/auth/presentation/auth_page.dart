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
import 'package:mobile_app/features/auth/domain/auth_validation.dart';
import 'package:mobile_app/features/auth/presentation/auth_return_location.dart';
import 'package:mobile_app/features/auth/presentation/password_checklist.dart';

class AuthPage extends StatefulWidget {
  const AuthPage({
    super.key,
    required this.register,
    this.returnLocation = AppRoutes.account,
  });
  final bool register;
  final String returnLocation;
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
  bool _terms = false;

  String? _required(String value) =>
      _submitted && value.trim().isEmpty ? 'Bắt buộc' : null;
  String? get _phoneError {
    if (!_submitted) return null;
    if (_identity.text.trim().isEmpty) return 'Bắt buộc';
    return AuthValidation.phone(_identity.text) ? null : 'Không hợp lệ';
  }

  String? get _passwordError {
    if (!_submitted) return null;
    if (_password.text.isEmpty) return 'Vui lòng nhập mật khẩu.';
    if (widget.register && !AuthValidation.password(_password.text)) {
      return 'Mật khẩu chưa đáp ứng đầy đủ điều kiện.';
    }
    return null;
  }

  Future<void> _submit() async {
    if (context.read<AuthCubit>().state.status == AuthStatus.submitting) return;
    setState(() => _submitted = true);
    if (_phoneError != null ||
        _passwordError != null ||
        (widget.register &&
            (_name.text.trim().isEmpty ||
                !AuthValidation.email(_email.text) ||
                !_terms ||
                _confirmation.text != _password.text))) {
      return;
    }
    final cubit = context.read<AuthCubit>();
    FocusScope.of(context).unfocus();
    final success = widget.register
        ? await cubit.register(
            name: _name.text,
            phone: _identity.text,
            password: _password.text,
            email: _email.text,
          )
        : await cubit.login(_identity.text, _password.text);
    if (success && mounted) {
      context.go(safeAuthReturnLocation(widget.returnLocation));
    }
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
            const Icon(
              Icons.local_florist_outlined,
              size: 48,
              color: AppColors.headingBrown,
            ),
            const SizedBox(height: AppSpacing.md),
            Text(
              widget.register ? 'Trở thành Tri Kỷ' : 'Chào mừng Tri Kỷ trở lại',
              style: Theme.of(context).textTheme.headlineLarge,
              textAlign: TextAlign.center,
            ),
            Text(
              widget.register
                  ? 'Tạo tài khoản để lưu địa chỉ, theo dõi đơn và nhận ưu đãi.'
                  : 'Đăng nhập để tiếp tục hành trình cùng những thức quà Huế.',
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
                enabled: state.status != AuthStatus.submitting,
                autofillHints: const [AutofillHints.name],
                textInputAction: TextInputAction.next,
              ),
              const SizedBox(height: AppSpacing.sm),
              AppTextField(
                controller: _email,
                label: 'Email khôi phục',
                keyboardType: TextInputType.emailAddress,
                enabled: state.status != AuthStatus.submitting,
                autofillHints: const [AutofillHints.email],
                errorText: _submitted && !AuthValidation.email(_email.text)
                    ? 'Vui lòng nhập email hợp lệ.'
                    : null,
                textInputAction: TextInputAction.next,
              ),
              const SizedBox(height: AppSpacing.sm),
            ],
            AppTextField(
              controller: _identity,
              label: 'Số điện thoại',
              keyboardType: TextInputType.phone,
              errorText: _phoneError,
              enabled: state.status != AuthStatus.submitting,
              autofillHints: const [AutofillHints.telephoneNumber],
              textInputAction: TextInputAction.next,
            ),
            const SizedBox(height: AppSpacing.sm),
            AppTextField(
              controller: _password,
              label: 'Mật khẩu',
              obscureText: !_showPassword,
              prefixIcon: Icons.lock_outline,
              errorText: _passwordError,
              enabled: state.status != AuthStatus.submitting,
              onChanged: (_) => setState(() {}),
              autofillHints: [
                widget.register
                    ? AutofillHints.newPassword
                    : AutofillHints.password,
              ],
              textInputAction: widget.register
                  ? TextInputAction.next
                  : TextInputAction.done,
              onSubmitted: widget.register ? null : (_) => _submit(),
              suffixIcon: IconButton(
                tooltip: _showPassword ? 'Ẩn mật khẩu' : 'Hiện mật khẩu',
                constraints: const BoxConstraints(minWidth: 44, minHeight: 44),
                onPressed: () => setState(() => _showPassword = !_showPassword),
                icon: Icon(
                  _showPassword ? Icons.visibility_off : Icons.visibility,
                ),
              ),
            ),
            if (widget.register) ...[
              const SizedBox(height: AppSpacing.sm),
              PasswordChecklist(password: _password.text),
              const SizedBox(height: AppSpacing.sm),
              AppTextField(
                controller: _confirmation,
                label: 'Xác nhận mật khẩu',
                obscureText: !_showPassword,
                enabled: state.status != AuthStatus.submitting,
                textInputAction: TextInputAction.done,
                onSubmitted: (_) => _submit(),
                errorText: _submitted && _confirmation.text != _password.text
                    ? 'Mật khẩu không khớp'
                    : null,
              ),
              CheckboxListTile(
                contentPadding: EdgeInsets.zero,
                controlAffinity: ListTileControlAffinity.leading,
                value: _terms,
                title: const Text(
                  'Tôi đồng ý với Quy chế Thẻ Tri Kỷ và Chính sách Bảo mật.',
                  style: TextStyle(fontSize: 13),
                ),
                onChanged: state.status == AuthStatus.submitting
                    ? null
                    : (value) => setState(() => _terms = value ?? false),
              ),
              if (_submitted && !_terms)
                const Text(
                  'Bạn cần chấp nhận điều khoản.',
                  style: TextStyle(color: AppColors.brickRed),
                ),
            ],
            if (!widget.register)
              Align(
                alignment: Alignment.centerRight,
                child: TextButton(
                  onPressed: state.status == AuthStatus.submitting
                      ? null
                      : () => context.push(
                          authLocation(
                            AppRoutes.recovery,
                            widget.returnLocation,
                          ),
                        ),
                  child: const Text('Quên mật khẩu?'),
                ),
              ),
            const SizedBox(height: AppSpacing.lg),
            AppPrimaryButton(
              label: widget.register ? 'Đăng ký' : 'Đăng nhập',
              loading: state.status == AuthStatus.submitting,
              onPressed: _submit,
            ),
            const SizedBox(height: AppSpacing.sm),
            TextButton(
              onPressed: state.status == AuthStatus.submitting
                  ? null
                  : () => context.go(
                      authLocation(
                        widget.register ? AppRoutes.login : AppRoutes.register,
                        widget.returnLocation,
                      ),
                    ),
              child: Text(
                widget.register
                    ? 'Đã có tài khoản? Đăng nhập'
                    : 'Chưa có tài khoản? Đăng ký',
              ),
            ),
            if (!widget.register)
              TextButton(
                onPressed: state.status == AuthStatus.submitting
                    ? null
                    : () => context.go(
                        Uri.parse(widget.returnLocation).path ==
                                AppRoutes.checkout
                            ? widget.returnLocation
                            : AppRoutes.home,
                      ),
                child: Text(
                  Uri.parse(widget.returnLocation).path == AppRoutes.checkout
                      ? 'Tiếp tục thanh toán với tư cách khách'
                      : 'Tiếp tục khám phá với tư cách khách',
                ),
              ),
          ],
        ),
      ),
    ),
  );
}
