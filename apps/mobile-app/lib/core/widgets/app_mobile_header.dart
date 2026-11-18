import 'package:flutter/material.dart';
import 'package:mobile_app/app/router/safe_navigation.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/widgets/header_actions.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_emblem.dart';

class AppMobileHeader extends StatelessWidget implements PreferredSizeWidget {
  const AppMobileHeader({
    super.key,
    this.title = 'Gói Huế',
    this.showBack = false,
    this.onBack,
    this.cartCount,
    this.onCart,
    this.onSearch,
    this.showCart = false,
    this.showNotifications = false,
  });
  final String title;
  final bool showBack;
  final VoidCallback? onBack;
  final int? cartCount;
  final VoidCallback? onCart;
  final VoidCallback? onSearch;

  /// Shared top-bar shortcuts (cart badge from CartCubit, notifications).
  final bool showCart;
  final bool showNotifications;

  @override
  Size get preferredSize => const Size.fromHeight(56);
  @override
  Widget build(BuildContext context) {
    final branded = !showBack && title == 'Gói Huế';
    return AppBar(
      automaticallyImplyLeading: false,
      leadingWidth: showBack
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
          : branded
          ? Padding(
              padding: const EdgeInsets.only(left: 16, top: 8, bottom: 8),
              child: const HeritageEmblem(size: 38),
            )
          : null,
      titleSpacing: 8,
      title: title == 'Gói Huế'
          ? Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (showBack) ...[
                  const HeritageEmblem(size: 32),
                  const SizedBox(width: 8),
                ],
                Text(
                  title,
                  style: Theme.of(context).textTheme.titleLarge?.copyWith(
                    color: HeritageColors.kem,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ],
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
        if (showCart || onCart != null) const HeaderCartAction(),
        if (showNotifications) const HeaderNotificationsAction(),
        const SizedBox(width: 8),
      ],
    );
  }
}
