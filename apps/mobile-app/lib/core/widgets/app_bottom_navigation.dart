import 'package:flutter/material.dart';
import 'package:mobile_app/core/theme/app_colors.dart';

class AppBottomNavigation extends StatelessWidget {
  const AppBottomNavigation({
    super.key,
    required this.activeIndex,
    this.cartCount = 0,
    this.onDestinationSelected,
  });
  final int activeIndex;
  final int cartCount;
  final ValueChanged<int>? onDestinationSelected;
  @override
  Widget build(BuildContext context) => DecoratedBox(
    decoration: const BoxDecoration(
      border: Border(top: BorderSide(color: AppColors.border, width: 0.5)),
    ),
    child: NavigationBar(
      height: 64,
      selectedIndex: activeIndex,
      onDestinationSelected: onDestinationSelected,
      backgroundColor: AppColors.ivory.withValues(alpha: .97),
      indicatorColor: Colors.transparent,
      labelBehavior: NavigationDestinationLabelBehavior.alwaysShow,
      destinations: [
        const NavigationDestination(
          icon: Icon(Icons.home_outlined),
          selectedIcon: Icon(Icons.home),
          label: 'Trang Chủ',
        ),
        const NavigationDestination(
          icon: Icon(Icons.restaurant_menu_outlined),
          selectedIcon: Icon(Icons.restaurant_menu),
          label: 'Thực Đơn',
        ),
        NavigationDestination(
          icon: Badge(
            isLabelVisible: cartCount > 0,
            label: Text('$cartCount'),
            child: const Icon(Icons.shopping_bag_outlined),
          ),
          selectedIcon: Badge(
            isLabelVisible: cartCount > 0,
            label: Text('$cartCount'),
            child: const Icon(Icons.shopping_bag),
          ),
          label: 'Giỏ Hàng',
        ),
        const NavigationDestination(
          icon: Icon(Icons.person_outline),
          selectedIcon: Icon(Icons.person),
          label: 'Tài Khoản',
        ),
      ],
    ),
  );
}
