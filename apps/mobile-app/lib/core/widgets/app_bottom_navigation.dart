import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_icons.dart';

/// Four top-level destinations. Every tab navigates to a screen; none opens a
/// sheet. Notifications and a cart shortcut live in the top bar.
enum AppNavTab { home, categories, cart, account }

class AppBottomNavigation extends StatelessWidget {
  const AppBottomNavigation({super.key, this.active, this.cartCount = 0});

  /// `null` on screens that are not a tab root.
  final AppNavTab? active;
  final int cartCount;

  static String route(AppNavTab tab) => switch (tab) {
    AppNavTab.home => AppRoutes.home,
    AppNavTab.categories => AppRoutes.catalog,
    AppNavTab.cart => AppRoutes.cart,
    AppNavTab.account => AppRoutes.account,
  };

  @override
  Widget build(BuildContext context) => Material(
    color: HeritageColors.nauCungDinh,
    child: DecoratedBox(
      decoration: const BoxDecoration(
        border: Border(top: BorderSide(color: HeritageColors.mat, width: 1)),
      ),
      child: SafeArea(
        top: false,
        child: SizedBox(
          height: 64,
          child: Row(
            children: [
              _item(
                context,
                AppNavTab.home,
                'Trang chủ',
                const HeritageIcon(HeritageIconKind.home),
              ),
              _item(
                context,
                AppNavTab.categories,
                'Danh mục',
                const HeritageIcon(HeritageIconKind.grid),
              ),
              _item(
                context,
                AppNavTab.cart,
                'Giỏ hàng',
                const HeritageIcon(HeritageIconKind.cart),
                badge: cartCount,
              ),
              _item(
                context,
                AppNavTab.account,
                'Tài khoản',
                const Icon(Icons.person_outline, size: 24),
              ),
            ],
          ),
        ),
      ),
    ),
  );

  Widget _item(
    BuildContext context,
    AppNavTab tab,
    String label,
    Widget icon, {
    int badge = 0,
  }) {
    final selected = active == tab;
    final color = selected ? HeritageColors.matSang : HeritageColors.kem;
    return Expanded(
      child: Semantics(
        button: true,
        selected: selected,
        label: badge > 0 ? '$label, $badge sản phẩm' : label,
        child: InkWell(
          onTap: selected ? null : () => context.go(route(tab)),
          child: ExcludeSemantics(
            child: IconTheme(
              data: IconThemeData(color: color),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Badge(
                    isLabelVisible: badge > 0,
                    label: Text(badge > 99 ? '99+' : '$badge'),
                    backgroundColor: HeritageColors.mat,
                    textColor: HeritageColors.nauCungDinh,
                    child: icon,
                  ),
                  const SizedBox(height: 4),
                  FittedBox(
                    fit: BoxFit.scaleDown,
                    child: Text(
                      label,
                      maxLines: 1,
                      style: TextStyle(
                        fontSize: 13,
                        fontWeight: selected
                            ? FontWeight.w700
                            : FontWeight.w500,
                        color: color,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
