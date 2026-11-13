import 'dart:convert';
import 'package:cryptography/cryptography.dart';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/features/auth/domain/auth_validation.dart';
import 'auth_cubit.dart';
import 'auth_return_location.dart';
import 'password_checklist.dart';

class IdentityAuthPage extends StatefulWidget {
  const IdentityAuthPage({
    super.key,
    required this.register,
    required this.returnLocation,
  });
  final bool register;
  final String returnLocation;
  @override
  State<IdentityAuthPage> createState() => _IdentityAuthPageState();
}

class _IdentityAuthPageState extends State<IdentityAuthPage> {
  final _first = TextEditingController(),
      _last = TextEditingController(),
      _email = TextEditingController(),
      _password = TextEditingController(),
      _confirm = TextEditingController();
  bool _terms = false,
      _submitted = false,
      _registered = false,
      _visible = false;
  String? _signature, _key;
  @override
  void dispose() {
    for (final controller in [_first, _last, _email, _password, _confirm]) {
      controller.dispose();
    }
    super.dispose();
  }

  Future<void> _login([String? provider]) async {
    final success = await context.read<AuthCubit>().signInWithBrowser(
      provider: provider,
    );
    if (success && mounted) {
      context.go(safeAuthReturnLocation(widget.returnLocation));
    }
  }

  Future<void> _register() async {
    final auth = context.read<AuthCubit>();
    if (auth.state.pending) return;
    setState(() => _submitted = true);
    if (_first.text.trim().isEmpty ||
        _last.text.trim().isEmpty ||
        !AuthValidation.email(_email.text) ||
        !AuthValidation.password(_password.text) ||
        _confirm.text != _password.text ||
        !_terms) {
      return;
    }
    // Keep the same key after an uncertain outcome with unchanged input.
    final firstName = _first.text.trim(),
        lastName = _last.text.trim(),
        email = _email.text.trim().toLowerCase(),
        password = _password.text;
    final signature = (await Sha256().hash(
      utf8.encode(jsonEncode([firstName, lastName, email, password])),
    )).bytes.toString();
    if (_signature != signature) {
      _signature = signature;
      _key = const Uuid().v4();
    }
    final registered = await auth.registerIdentity(
      firstName: firstName,
      lastName: lastName,
      email: email,
      password: password,
      idempotencyKey: _key!,
    );
    if (!mounted || !registered) return;
    setState(() => _registered = true);
    _password.clear();
    _confirm.clear();
    await _login();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppMobileHeader(
      title: widget.register ? 'Đăng Ký' : 'Đăng Nhập',
      showBack: true,
    ),
    body: SafeArea(
      child: BlocBuilder<AuthCubit, AuthState>(
        builder: (context, state) {
          final auth = context.read<AuthCubit>();
          return ListView(
            padding: const EdgeInsets.all(24),
            children: [
              const SizedBox(height: 20),
              Text(
                widget.register
                    ? 'Trở thành Tri Kỷ'
                    : 'Chào mừng Tri Kỷ trở lại',
                style: Theme.of(context).textTheme.headlineLarge,
              ),
              const SizedBox(height: 12),
              Text(
                widget.register
                    ? 'Tạo tài khoản khách hàng bằng email để tiếp tục hành trình cùng Ô Mạ.'
                    : 'Tiếp tục đăng nhập trong trình duyệt bảo mật. Bạn sẽ được đưa trở lại ứng dụng khi hoàn tất.',
              ),
              const SizedBox(height: 24),
              if (_registered) ...[
                const Text(
                  'Đã đăng ký thành công. Tiếp tục đăng nhập để mở tài khoản.',
                ),
                const SizedBox(height: 16),
              ],
              if (state.message != null) ...[
                Semantics(
                  liveRegion: true,
                  child: Text(
                    state.message!,
                    style: TextStyle(
                      color: state.status == AuthStatus.failure
                          ? AppColors.darkRed
                          : AppColors.text,
                    ),
                  ),
                ),
                const SizedBox(height: 16),
              ],
              if (widget.register && !_registered) ...[
                AppTextField(
                  controller: _last,
                  label: 'Họ',
                  enabled: !state.pending,
                  textInputAction: TextInputAction.next,
                  autofillHints: const [AutofillHints.familyName],
                  errorText: _submitted && _last.text.trim().isEmpty
                      ? 'Vui lòng nhập họ.'
                      : null,
                ),
                const SizedBox(height: 12),
                AppTextField(
                  controller: _first,
                  label: 'Tên',
                  enabled: !state.pending,
                  textInputAction: TextInputAction.next,
                  autofillHints: const [AutofillHints.givenName],
                  errorText: _submitted && _first.text.trim().isEmpty
                      ? 'Vui lòng nhập tên.'
                      : null,
                ),
                const SizedBox(height: 12),
                AppTextField(
                  controller: _email,
                  label: 'Email',
                  enabled: !state.pending,
                  keyboardType: TextInputType.emailAddress,
                  textInputAction: TextInputAction.next,
                  autofillHints: const [AutofillHints.email],
                  errorText: _submitted && !AuthValidation.email(_email.text)
                      ? 'Vui lòng nhập email hợp lệ.'
                      : null,
                ),
                const SizedBox(height: 12),
                AppTextField(
                  controller: _password,
                  label: 'Mật khẩu',
                  enabled: !state.pending,
                  obscureText: !_visible,
                  textInputAction: TextInputAction.next,
                  autofillHints: const [AutofillHints.newPassword],
                  onChanged: (_) => setState(() {}),
                  errorText:
                      _submitted && !AuthValidation.password(_password.text)
                      ? 'Mật khẩu chưa đáp ứng điều kiện.'
                      : null,
                  suffixIcon: IconButton(
                    tooltip: _visible ? 'Ẩn mật khẩu' : 'Hiện mật khẩu',
                    onPressed: state.pending
                        ? null
                        : () => setState(() => _visible = !_visible),
                    icon: Icon(
                      _visible
                          ? Icons.visibility_off_outlined
                          : Icons.visibility_outlined,
                    ),
                  ),
                ),
                PasswordChecklist(password: _password.text),
                const SizedBox(height: 12),
                AppTextField(
                  controller: _confirm,
                  label: 'Xác nhận mật khẩu',
                  enabled: !state.pending,
                  obscureText: !_visible,
                  textInputAction: TextInputAction.done,
                  onSubmitted: (_) => _register(),
                  errorText: _submitted && _confirm.text != _password.text
                      ? 'Mật khẩu không khớp.'
                      : null,
                ),
                CheckboxListTile(
                  contentPadding: EdgeInsets.zero,
                  controlAffinity: ListTileControlAffinity.leading,
                  value: _terms,
                  onChanged: state.pending
                      ? null
                      : (v) => setState(() => _terms = v ?? false),
                  title: const Text(
                    'Tôi đồng ý với Quy chế Thẻ Tri Kỷ và Chính sách Bảo mật.',
                  ),
                ),
                if (_submitted && !_terms)
                  const Text(
                    'Bạn cần chấp nhận điều khoản.',
                    style: TextStyle(color: AppColors.darkRed),
                  ),
                const SizedBox(height: 12),
              ],
              AppPrimaryButton(
                label: widget.register && !_registered
                    ? 'Đăng ký'
                    : 'Tiếp tục đăng nhập',
                loading: state.pending,
                onPressed: widget.register && !_registered
                    ? _register
                    : () => _login(),
              ),
              for (final provider in auth.identityProviders) ...[
                const SizedBox(height: 12),
                OutlinedButton(
                  onPressed: state.pending ? null : () => _login(provider),
                  child: Text(
                    '${widget.register ? "Đăng ký" : "Đăng nhập"} bằng ${provider == "google" ? "Google" : "Zalo"}',
                  ),
                ),
              ],
              const SizedBox(height: 12),
              TextButton(
                onPressed: state.pending
                    ? null
                    : () => context.go(
                        authLocation(
                          widget.register
                              ? AppRoutes.login
                              : AppRoutes.register,
                          widget.returnLocation,
                        ),
                      ),
                child: Text(
                  widget.register
                      ? 'Đã có tài khoản? Đăng nhập'
                      : 'Chưa có tài khoản? Đăng ký',
                ),
              ),
              if (!widget.register) ...[
                const Text(
                  'Bạn có thể chọn quên mật khẩu trên trang đăng nhập bảo mật.',
                ),
                TextButton(
                  onPressed: state.pending
                      ? null
                      : () => context.go(AppRoutes.home),
                  child: const Text('Tiếp tục khám phá với tư cách khách'),
                ),
              ],
            ],
          );
        },
      ),
    ),
  );
}
