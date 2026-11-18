import 'package:flutter/material.dart';
import 'package:mobile_app/app/router/safe_navigation.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_emblem.dart';

class AppMobileHeader extends StatelessWidget implements PreferredSizeWidget {
  const AppMobileHeader({
    super.key,
    this.title = 'Gói Huế',
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
    final branded = !showBack && title == 'Gói Huế';
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
              child: const HeritageEmblem(size: 38),
            )
          : null,
      titleSpacing: 8,
      title: branded
          ? Text(
              title,
              style: Theme.of(context).textTheme.titleLarge?.copyWith(
                color: HeritageColors.kem,
                fontWeight: FontWeight.w700,
              ),
            )
          : Text(title),
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
              backgroundColor: HeritageColors.mat,
              textColor: HeritageColors.nauCungDinh,
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
