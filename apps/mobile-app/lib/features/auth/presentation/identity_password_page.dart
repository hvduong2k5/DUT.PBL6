import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/features/auth/domain/auth_validation.dart';
import 'auth_cubit.dart';
import 'password_checklist.dart';

class IdentityPasswordPage extends StatefulWidget {
  const IdentityPasswordPage({super.key});
  @override
  State<IdentityPasswordPage> createState() => _IdentityPasswordPageState();
}

class _IdentityPasswordPageState extends State<IdentityPasswordPage> {
  final _current = TextEditingController(),
      _next = TextEditingController(),
      _confirm = TextEditingController();
  bool _pending = false, _done = false;
  String? _message;
  @override
  void dispose() {
    for (final c in [_current, _next, _confirm]) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _submit() async {
    if (_pending) return;
    if (_current.text.isEmpty ||
        !AuthValidation.password(_next.text) ||
        _confirm.text != _next.text) {
      setState(
        () => _message =
            'Vui lòng kiểm tra mật khẩu hiện tại, mật khẩu mới và xác nhận.',
      );
      return;
    }
    setState(() {
      _pending = true;
      _message = null;
    });
    try {
      await context.read<AuthCubit>().changeIdentityPassword(
        _current.text,
        _next.text,
      );
      if (!mounted) return;
      _current.clear();
      _next.clear();
      _confirm.clear();
      setState(() {
        _done = true;
        _message = 'Đã cập nhật mật khẩu.';
      });
    } catch (error) {
      if (mounted) {
        setState(
          () => _message = error is ApiException
              ? error.failure.message
              : 'Chưa thể cập nhật mật khẩu. Vui lòng thử lại.',
        );
      }
    } finally {
      if (mounted) setState(() => _pending = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Đổi mật khẩu', showBack: true),
    body: SafeArea(
      child: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          if (_message != null)
            Semantics(liveRegion: true, child: Text(_message!)),
          if (!_done) ...[
            const SizedBox(height: 16),
            AppTextField(
              controller: _current,
              label: 'Mật khẩu hiện tại',
              obscureText: true,
              enabled: !_pending,
              textInputAction: TextInputAction.next,
              autofillHints: const [AutofillHints.password],
            ),
            const SizedBox(height: 12),
            AppTextField(
              controller: _next,
              label: 'Mật khẩu mới',
              obscureText: true,
              enabled: !_pending,
              onChanged: (_) => setState(() {}),
              textInputAction: TextInputAction.next,
              autofillHints: const [AutofillHints.newPassword],
            ),
            PasswordChecklist(password: _next.text),
            const SizedBox(height: 12),
            AppTextField(
              controller: _confirm,
              label: 'Xác nhận mật khẩu mới',
              obscureText: true,
              enabled: !_pending,
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => _submit(),
            ),
            const SizedBox(height: 24),
            AppPrimaryButton(
              label: 'Cập nhật mật khẩu',
              loading: _pending,
              onPressed: _submit,
            ),
          ],
        ],
      ),
    ),
  );
}
