import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/features/account/presentation/profile_cubit.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';

class ProfilePage extends StatefulWidget {
  const ProfilePage({super.key});
  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  final _name = TextEditingController();
  final _email = TextEditingController();
  bool _initialized = false;
  void _sync(ProfileState state) {
    _name.text = state.profile!.user.fullName;
    _email.text = state.profile!.user.email ?? '';
  }

  @override
  void dispose() {
    _name.dispose();
    _email.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Hồ sơ cá nhân', showBack: true),
    body: BlocConsumer<ProfileCubit, ProfileState>(
      listener: (context, state) {
        if (state.loading) _initialized = false;
        if (state.profile != null && (!_initialized || state.saved)) {
          _sync(state);
          _initialized = true;
        }
        if (state.saved) {
          context.read<AuthCubit>().updateUser(state.profile!.user);
        }
      },
      builder: (context, state) {
        if (state.loading) return const AppLoadingView();
        if (state.profile == null) {
          return AppErrorView(
            message: state.message ?? 'Chưa thể mở hồ sơ.',
            onRetry: context.read<ProfileCubit>().load,
          );
        }
        if (!_initialized) {
          _sync(state);
          _initialized = true;
        }
        return ListView(
          padding: const EdgeInsets.all(20),
          children: [
            Text(
              'Thông tin định danh cơ bản',
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: 12),
            const Text(
              'Cập nhật thông tin liên hệ để nhận quà và theo dõi đơn thuận tiện hơn.',
            ),
            const SizedBox(height: 24),
            AppTextField(
              controller: _name,
              label: 'Họ và tên',
              enabled: !state.saving,
              errorText: state.errors['full_name'],
              autofillHints: const [AutofillHints.name],
            ),
            const SizedBox(height: 16),
            ListTile(
              contentPadding: EdgeInsets.zero,
              title: const Text('Số điện thoại tài khoản'),
              subtitle: Text(state.profile!.user.phoneNumber),
              trailing: const Icon(Icons.lock_outline),
            ),
            const Text(
              'Số điện thoại tài khoản chưa thể thay đổi tại đây.',
              style: TextStyle(color: AppColors.textMuted, fontSize: 12),
            ),
            const SizedBox(height: 16),
            AppTextField(
              controller: _email,
              label: 'Email liên hệ',
              enabled: !state.saving,
              keyboardType: TextInputType.emailAddress,
              errorText: state.errors['email'],
              autofillHints: const [AutofillHints.email],
            ),
            if (state.message != null)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 16),
                child: Semantics(
                  liveRegion: true,
                  child: Text(
                    state.message!,
                    style: TextStyle(
                      color: state.saved
                          ? AppColors.success
                          : AppColors.darkRed,
                    ),
                  ),
                ),
              ),
            if (state.conflict)
              TextButton(
                onPressed: context.read<ProfileCubit>().load,
                child: const Text('Tải lại hồ sơ mới nhất'),
              ),
            const SizedBox(height: 24),
            AppPrimaryButton(
              label: 'Lưu thay đổi',
              loading: state.saving,
              onPressed: () =>
                  context.read<ProfileCubit>().save(_name.text, _email.text),
            ),
            TextButton(
              onPressed: state.saving
                  ? null
                  : () => setState(() => _sync(state)),
              child: const Text('Hủy thay đổi'),
            ),
          ],
        );
      },
    ),
  );
}
