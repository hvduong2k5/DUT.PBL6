import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';

class AppNavigationDrawer extends StatelessWidget {
  const AppNavigationDrawer({super.key, required this.activeRoute});

  final String activeRoute;

  void _open(BuildContext context, String route) {
    Navigator.of(context).pop();
    if (route != activeRoute) context.go(route);
  }

  @override
  Widget build(BuildContext context) => Drawer(
    child: SafeArea(
      child: BlocBuilder<AuthCubit, AuthState>(
        builder: (context, authState) => ListView(
          padding: EdgeInsets.zero,
          children: [
            Container(
              padding: const EdgeInsets.fromLTRB(20, 20, 20, 18),
              color: AppColors.brandBrown,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const CircleAvatar(
                    radius: 24,
                    backgroundColor: AppColors.honeyGold,
                    child: Text(
                      'Ô',
                      style: TextStyle(
                        color: AppColors.ctaText,
                        fontSize: 20,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                  const SizedBox(height: 12),
                  const Text(
                    'Ô Mạ',
                    style: TextStyle(
                      color: Colors.white,
                      fontSize: 20,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  Text(
                    authState.status == AuthStatus.authenticated
                        ? authState.user!.fullName
                        : 'Mua sắm với tư cách khách',
                    style: const TextStyle(color: Colors.white70),
                  ),
                ],
              ),
            ),
            _DrawerDestination(
              icon: Icons.home_outlined,
              label: 'Trang Chủ',
              selected: activeRoute == AppRoutes.home,
              onTap: () => _open(context, AppRoutes.home),
            ),
            _DrawerDestination(
              icon: Icons.restaurant_menu,
              label: 'Thực Đơn',
              selected: activeRoute == AppRoutes.catalog,
              onTap: () => _open(context, AppRoutes.catalog),
            ),
            BlocBuilder<CartCubit, CartState>(
              builder: (context, cartState) => _DrawerDestination(
                icon: Icons.shopping_bag_outlined,
                label: cartState.cart.totalItems == 0
                    ? 'Giỏ Hàng'
                    : 'Giỏ Hàng (${cartState.cart.totalItems})',
                selected: activeRoute == AppRoutes.cart,
                onTap: () => _open(context, AppRoutes.cart),
              ),
            ),
            _DrawerDestination(
              icon: Icons.person_outline,
              label: 'Tài Khoản',
              selected: activeRoute == AppRoutes.account,
              onTap: () => _open(context, AppRoutes.account),
            ),
            if (authState.status == AuthStatus.authenticated) ...[
              const Divider(),
              _DrawerDestination(
                icon: Icons.receipt_long_outlined,
                label: 'Đơn hàng của tôi',
                onTap: () => _open(context, AppRoutes.orders),
              ),
              _DrawerDestination(
                icon: Icons.location_on_outlined,
                label: 'Địa chỉ giao hàng',
                onTap: () => _open(context, AppRoutes.addresses),
              ),
              _DrawerDestination(
                icon: Icons.notifications_none,
                label: 'Thông báo',
                onTap: () => _open(context, AppRoutes.notifications),
              ),
            ],
            const Divider(),
            if (authState.status == AuthStatus.authenticated)
              _DrawerDestination(
                icon: Icons.logout,
                label: 'Đăng xuất',
                onTap: () async {
                  Navigator.of(context).pop();
                  await context.read<AuthCubit>().logout();
                  if (context.mounted) context.go(AppRoutes.home);
                },
              )
            else ...[
              _DrawerDestination(
                icon: Icons.login,
                label: 'Đăng nhập',
                onTap: () => _open(context, AppRoutes.login),
              ),
              _DrawerDestination(
                icon: Icons.person_add_outlined,
                label: 'Đăng ký',
                onTap: () => _open(context, AppRoutes.register),
              ),
            ],
          ],
        ),
      ),
    ),
  );
}

class _DrawerDestination extends StatelessWidget {
  const _DrawerDestination({
    required this.icon,
    required this.label,
    required this.onTap,
    this.selected = false,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;
  final bool selected;

  @override
  Widget build(BuildContext context) => ListTile(
    minTileHeight: 48,
    selected: selected,
    selectedColor: AppColors.brandBrown,
    selectedTileColor: AppColors.cream,
    leading: Icon(icon),
    title: Text(label),
    trailing: selected ? const Icon(Icons.check, size: 18) : null,
    onTap: onTap,
  );
}
