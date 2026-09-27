import 'package:flutter/material.dart';
import 'package:mobile_app/app/router/safe_navigation.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_radius.dart';

class AppMobileHeader extends StatelessWidget implements PreferredSizeWidget {
  const AppMobileHeader({
    super.key,
    this.title = 'Ô Mạ',
    this.showBack = false,
    this.showMenu = false,
    this.onBack,
    this.cartCount,
    this.onCart,
    this.onSearch,
  });
  final String title;
  final bool showBack;
  final bool showMenu;
  final VoidCallback? onBack;
  final int? cartCount;
  final VoidCallback? onCart;
  final VoidCallback? onSearch;

  @override
  Size get preferredSize => const Size.fromHeight(56);
  @override
  Widget build(BuildContext context) {
    final branded = !showBack && title == 'Ô Mạ';
    return AppBar(
      automaticallyImplyLeading: false,
      leadingWidth: showBack || showMenu
          ? 64
          : branded
          ? 56
          : 16,
      leading: showBack
          ? IconButton(
              onPressed: onBack ?? () => safeBackOrHome(context),
              icon: const Icon(Icons.arrow_back_ios_new),
              tooltip: 'Quay lại',
            )
          : showMenu
          ? Builder(
              builder: (buttonContext) => IconButton(
                constraints: const BoxConstraints(minWidth: 48, minHeight: 48),
                onPressed: () => Scaffold.of(buttonContext).openDrawer(),
                icon: const Icon(Icons.menu),
                tooltip: 'Mở menu',
              ),
            )
          : branded
          ? Padding(
              padding: const EdgeInsets.only(left: 16, top: 8, bottom: 8),
              child: DecoratedBox(
                decoration: BoxDecoration(
                  color: AppColors.brandBrown,
                  borderRadius: BorderRadius.circular(AppRadius.medium),
                ),
                child: const Center(
                  child: Text(
                    'Ô',
                    style: TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ),
              ),
            )
          : null,
      titleSpacing: 8,
      title: branded
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: Theme.of(context).textTheme.titleMedium),
                const Text(
                  'Tinh hoa bánh mứt ngự trà',
                  style: TextStyle(fontSize: 9, color: AppColors.textMuted),
                ),
              ],
            )
          : Text(title, style: Theme.of(context).textTheme.titleLarge),
      bottom: PreferredSize(
        preferredSize: const Size.fromHeight(1),
        child: Container(
          height: 1,
          color: AppColors.border.withValues(alpha: .4),
        ),
      ),
      actions: [
        if (onSearch != null)
          SizedBox.square(
            dimension: 48,
            child: IconButton(
              onPressed: onSearch,
              icon: const Icon(Icons.search),
              tooltip: 'Tìm kiếm',
            ),
          ),
        if (onCart != null)
          SizedBox.square(
            dimension: 48,
            child: Badge(
              isLabelVisible: (cartCount ?? 0) > 0,
              label: Text('$cartCount'),
              backgroundColor: AppColors.darkRed,
              child: IconButton(
                onPressed: onCart,
                icon: const Icon(Icons.shopping_bag_outlined),
                tooltip: 'Giỏ hàng',
              ),
            ),
          ),
        const SizedBox(width: 8),
      ],
    );
  }
}
