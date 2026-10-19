import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_secondary_button.dart';
import 'package:mobile_app/core/widgets/app_bottom_navigation.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_navigation_drawer.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';

class AccountPage extends StatelessWidget {
  const AccountPage({super.key});

  @override
  Widget build(BuildContext context) => BlocBuilder<AuthCubit, AuthState>(
    builder: (context, authState) => BlocBuilder<CartCubit, CartState>(
      builder: (context, cartState) {
        final isGuest = authState.status != AuthStatus.authenticated;
        return Scaffold(
          appBar: const AppMobileHeader(title: 'Tài Khoản', showMenu: true),
          drawer: const AppNavigationDrawer(activeRoute: AppRoutes.account),
          body: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              if (authState.sessionRestoreFailed)
                Card(
                  child: ListTile(
                    title: Text(authState.message!),
                    trailing: IconButton(
                      icon: const Icon(Icons.refresh),
                      tooltip: 'Khôi phục phiên',
                      onPressed: context.read<AuthCubit>().restore,
                    ),
                  ),
                ),
              if (isGuest)
                _GuestCard(
                  onLogin: () => context.push(AppRoutes.login),
                  onRegister: () => context.push(AppRoutes.register),
                )
              else
                _ProfileCard(user: authState.user!),
              if (!isGuest) ...[
                const SizedBox(height: AppSpacing.lg),
                const _SectionLabel('TÀI KHOẢN'),
                _MenuRow(
                  icon: Icons.person_outline,
                  label: 'Hồ sơ cá nhân',
                  onTap: () => context.push(AppRoutes.profile),
                ),
                _MenuRow(
                  icon: Icons.loyalty_outlined,
                  label: 'Ví điểm & đặc quyền',
                  onTap: () => context.push(AppRoutes.loyalty),
                ),
                _MenuRow(
                  icon: Icons.receipt_long_outlined,
                  label: 'Đơn hàng của tôi',
                  onTap: () => context.push(AppRoutes.orders),
                ),
                _MenuRow(
                  icon: Icons.location_on_outlined,
                  label: 'Địa chỉ giao hàng',
                  onTap: () => context.push(AppRoutes.addresses),
                ),
                _MenuRow(
                  icon: Icons.notifications_none,
                  label: 'Thông báo',
                  onTap: () => context.push(AppRoutes.notifications),
                ),
              ],
              const SizedBox(height: AppSpacing.md),
              const _SectionLabel('KHÁC'),
              _MenuRow(
                icon: Icons.local_offer_outlined,
                label: 'Ưu đãi Tri Kỷ',
                onTap: () => context.push(AppRoutes.promotions),
              ),
              _MenuRow(
                icon: Icons.help_outline,
                label: 'Hỗ trợ',
                onTap: () => context.push(AppRoutes.supportRequest),
              ),
              const _MenuRow(icon: Icons.info_outline, label: 'Về Ô Mạ'),
              if (!isGuest) ...[
                const SizedBox(height: AppSpacing.md),
                OutlinedButton.icon(
                  onPressed: () => context.read<AuthCubit>().logout(),
                  icon: const Icon(Icons.logout),
                  label: const Text('Đăng xuất'),
                  style: OutlinedButton.styleFrom(
                    minimumSize: const Size.fromHeight(48),
                    foregroundColor: AppColors.darkRed,
                    side: const BorderSide(color: AppColors.darkRed),
                  ),
                ),
              ],
            ],
          ),
          bottomNavigationBar: AppBottomNavigation(
            activeIndex: 3,
            cartCount: cartState.cart.totalItems,
            onDestinationSelected: (index) {
              if (index == 0) context.go(AppRoutes.home);
              if (index == 1) context.go(AppRoutes.catalog);
              if (index == 2) context.go(AppRoutes.cart);
            },
          ),
        );
      },
    ),
  );
}

class _ProfileCard extends StatelessWidget {
  const _ProfileCard({required this.user});
  final AuthUser user;

  String get _initials {
    final parts = user.fullName.split(' ').where((s) => s.isNotEmpty).toList();
    if (parts.isEmpty) return 'NA';
    if (parts.length == 1) return parts[0].characters.first.toUpperCase();
    return '${parts.first.characters.first}${parts.last.characters.first}'
        .toUpperCase();
  }

  @override
  Widget build(BuildContext context) => Card(
    child: Padding(
      padding: const EdgeInsets.all(AppSpacing.md),
      child: Row(
        children: [
          CircleAvatar(radius: 34, child: Text(_initials)),
          const SizedBox(width: AppSpacing.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  user.fullName,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                Text(user.phoneNumber),
                const Text(
                  'Thông tin hồ sơ người dùng',
                  style: TextStyle(fontSize: 11),
                ),
              ],
            ),
          ),
        ],
      ),
    ),
  );
}

class _GuestCard extends StatelessWidget {
  const _GuestCard({required this.onLogin, required this.onRegister});
  final VoidCallback onLogin;
  final VoidCallback onRegister;
  @override
  Widget build(BuildContext context) => Column(
    children: [
      const CircleAvatar(radius: 34, child: Icon(Icons.person_outline)),
      const SizedBox(height: AppSpacing.md),
      Text(
        'Đăng nhập để quản lý đơn hàng',
        style: Theme.of(context).textTheme.titleLarge,
        textAlign: TextAlign.center,
      ),
      const SizedBox(height: AppSpacing.md),
      AppPrimaryButton(onPressed: onLogin, label: 'Đăng nhập'),
      AppSecondaryButton(onPressed: onRegister, label: 'Đăng ký'),
    ],
  );
}

class _SectionLabel extends StatelessWidget {
  const _SectionLabel(this.label);
  final String label;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: AppSpacing.xs),
    child: Text(
      label,
      style: const TextStyle(
        fontSize: 10,
        color: AppColors.textMuted,
        fontWeight: FontWeight.bold,
      ),
    ),
  );
}

class _MenuRow extends StatelessWidget {
  const _MenuRow({required this.icon, required this.label, this.onTap});
  final IconData icon;
  final String label;
  final VoidCallback? onTap;
  @override
  Widget build(BuildContext context) => Card(
    margin: const EdgeInsets.only(bottom: 8),
    child: ListTile(
      minTileHeight: 56,
      leading: Icon(icon),
      title: Text(label),
      trailing: const Icon(Icons.chevron_right),
      onTap: onTap,
    ),
  );
}
