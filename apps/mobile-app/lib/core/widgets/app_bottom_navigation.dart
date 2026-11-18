import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_bottom_sheet.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_icons.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/features/catalog/domain/catalog_repository.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

/// Kit bottom navigation (same order as web): Trang chủ, Danh mục (sheet),
/// Ưu đãi, Tìm kiếm (sheet), Giỏ hàng. Navigates by itself.
enum AppNavTab { home, categories, promotions, search, cart }

class AppBottomNavigation extends StatelessWidget {
  const AppBottomNavigation({super.key, this.active, this.cartCount = 0});

  /// `null` on pages that are not a nav destination (e.g. account).
  final AppNavTab? active;
  final int cartCount;

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
                HeritageIconKind.home,
                () => context.go(AppRoutes.home),
              ),
              _item(
                context,
                AppNavTab.categories,
                'Danh mục',
                HeritageIconKind.grid,
                () => _categories(context),
              ),
              _item(
                context,
                AppNavTab.promotions,
                'Ưu đãi',
                HeritageIconKind.gift,
                () => context.go(AppRoutes.promotions),
              ),
              _item(
                context,
                AppNavTab.search,
                'Tìm kiếm',
                HeritageIconKind.search,
                () => _search(context),
              ),
              _item(
                context,
                AppNavTab.cart,
                'Giỏ hàng',
                HeritageIconKind.cart,
                () => context.go(AppRoutes.cart),
                badge: cartCount,
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
    HeritageIconKind icon,
    VoidCallback onTap, {
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
          onTap: onTap,
          child: ExcludeSemantics(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Badge(
                  isLabelVisible: badge > 0,
                  label: Text(badge > 99 ? '99+' : '$badge'),
                  backgroundColor: HeritageColors.mat,
                  textColor: HeritageColors.nauCungDinh,
                  child: HeritageIcon(icon, color: color),
                ),
                const SizedBox(height: 4),
                Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
                    color: color,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  void _categories(BuildContext context) {
    final repository = context.read<CatalogRepository>();
    final router = GoRouter.of(context);
    showHeritageSheet<void>(
      context,
      title: 'Bộ sưu tập',
      builder: (sheetContext) => FutureBuilder<List<Category>>(
        future: repository.categories(),
        builder: (_, snapshot) {
          final categories = snapshot.data ?? const <Category>[];
          return ListView(
            shrinkWrap: true,
            padding: const EdgeInsets.fromLTRB(8, 0, 8, 16),
            children: [
              for (final c in categories)
                ListTile(
                  title: Text(c.name),
                  onTap: () {
                    Navigator.of(sheetContext).pop();
                    router.go(
                      '${AppRoutes.catalog}?category=${Uri.encodeQueryComponent(c.id)}',
                    );
                  },
                ),
              ListTile(
                title: const Text(
                  'Xem tất cả sản phẩm',
                  style: TextStyle(fontWeight: FontWeight.w700),
                ),
                subtitle: const Text('Toàn bộ bộ sưu tập Gói Huế'),
                onTap: () {
                  Navigator.of(sheetContext).pop();
                  router.go(AppRoutes.catalog);
                },
              ),
            ],
          );
        },
      ),
    );
  }

  void _search(BuildContext context) {
    final router = GoRouter.of(context);
    final controller = TextEditingController();
    showHeritageSheet<void>(
      context,
      title: 'Tìm sản phẩm',
      builder: (sheetContext) {
        void submit() {
          final q = controller.text.trim();
          Navigator.of(sheetContext).pop();
          router.go(
            q.isEmpty
                ? AppRoutes.catalog
                : '${AppRoutes.catalog}?q=${Uri.encodeQueryComponent(q)}',
          );
        }

        return Padding(
          padding: const EdgeInsets.fromLTRB(20, 8, 20, 20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              TextField(
                controller: controller,
                autofocus: true,
                textInputAction: TextInputAction.search,
                onSubmitted: (_) => submit(),
                decoration: const InputDecoration(
                  labelText: 'Nhập tên sản phẩm',
                  hintText: 'Tìm mè xửng, trà sen…',
                ),
              ),
              const SizedBox(height: 12),
              FilledButton(onPressed: submit, child: const Text('Tìm kiếm')),
            ],
          ),
        );
      },
    ).whenComplete(controller.dispose);
  }
}
