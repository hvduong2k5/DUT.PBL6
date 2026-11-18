import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/widgets/app_bottom_navigation.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_product_image.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_frame.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_search_bar.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_surface.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_cubit.dart';

/// Categories tab (Tiki/Shopee pattern): category rail on the left, products
/// of the selected category on the right. Data comes from the catalog API.
class CategoriesPage extends StatelessWidget {
  const CategoriesPage({super.key});

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const HeritageSearchHeader(),
    body: HeritageSurface(
      child: BlocBuilder<CatalogCubit, CatalogState>(
        builder: (context, state) => switch (state.status) {
          CatalogStatus.initial ||
          CatalogStatus.loading => const AppLoadingView(),
          CatalogStatus.failure => AppErrorView(
            message: state.message ?? 'Không thể tải danh mục.',
            onRetry: context.read<CatalogCubit>().load,
          ),
          _ => Row(
            key: const Key('categories-content'),
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _Rail(state: state),
              const VerticalDivider(width: 1, color: HeritageColors.vien),
              Expanded(child: _Products(state: state)),
            ],
          ),
        },
      ),
    ),
    bottomNavigationBar: BlocBuilder<CartCubit, CartState>(
      builder: (context, cart) => AppBottomNavigation(
        active: AppNavTab.categories,
        cartCount: cart.cart.totalItems,
      ),
    ),
  );
}

class _Rail extends StatelessWidget {
  const _Rail({required this.state});
  final CatalogState state;

  @override
  Widget build(BuildContext context) {
    final width = (MediaQuery.sizeOf(context).width * .28).clamp(92.0, 120.0);
    return SizedBox(
      width: width,
      child: ColoredBox(
        color: HeritageColors.kemAm,
        child: ListView(
          key: const Key('categories-rail'),
          children: [
            for (final category in state.categories)
              _RailItem(
                category: category,
                selected: state.categoryId == category.id,
                onTap: () =>
                    context.read<CatalogCubit>().selectCategory(category.id),
              ),
          ],
        ),
      ),
    );
  }
}

class _RailItem extends StatelessWidget {
  const _RailItem({
    required this.category,
    required this.selected,
    required this.onTap,
  });
  final Category category;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) => Semantics(
    selected: selected,
    button: true,
    child: InkWell(
      onTap: onTap,
      child: Container(
        constraints: const BoxConstraints(minHeight: 64),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 12),
        decoration: BoxDecoration(
          color: selected ? HeritageColors.kem : Colors.transparent,
          border: Border(
            left: BorderSide(
              color: selected ? HeritageColors.mat : Colors.transparent,
              width: 3,
            ),
          ),
        ),
        alignment: Alignment.centerLeft,
        child: Text(
          category.name,
          style: TextStyle(
            fontSize: 13,
            height: 1.35,
            fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
            color: selected ? HeritageColors.nauDam : HeritageColors.phu,
          ),
        ),
      ),
    ),
  );
}

class _Products extends StatelessWidget {
  const _Products({required this.state});
  final CatalogState state;

  @override
  Widget build(BuildContext context) {
    final products = state.products;
    final current = state.categories
        .where((c) => c.id == state.categoryId)
        .map((c) => c.name)
        .firstOrNull;
    if (products.isEmpty) {
      return const AppEmptyView(
        title: 'Chưa có sản phẩm',
        message: 'Danh mục này chưa có sản phẩm đang bán.',
      );
    }
    final scale = MediaQuery.textScalerOf(context).scale(1).clamp(1.0, 2.0);
    return CustomScrollView(
      slivers: [
        SliverPadding(
          padding: const EdgeInsets.fromLTRB(12, 14, 12, 10),
          sliver: SliverToBoxAdapter(
            child: Text(
              current ?? 'Sản phẩm',
              style: Theme.of(context).textTheme.titleLarge,
            ),
          ),
        ),
        SliverPadding(
          padding: const EdgeInsets.fromLTRB(12, 0, 12, 24),
          sliver: SliverGrid.builder(
            itemCount: products.length,
            gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: 2,
              crossAxisSpacing: 10,
              mainAxisSpacing: 10,
              childAspectRatio: .62 / scale,
            ),
            itemBuilder: (context, i) => _Tile(product: products[i]),
          ),
        ),
      ],
    );
  }
}

/// Compact product tile for the narrow right pane: image, name, price.
class _Tile extends StatelessWidget {
  const _Tile({required this.product});
  final Product product;

  @override
  Widget build(BuildContext context) => Semantics(
    button: true,
    label: product.name,
    child: InkWell(
      onTap: () => context.push('/product/${Uri.encodeComponent(product.id)}'),
      child: HeritageFrame(
        level: HeritageFrameLevel.card,
        padding: const EdgeInsets.all(8),
        child: ExcludeSemantics(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: SizedBox(
                  width: double.infinity,
                  child: AppProductImage(
                    url: product.imageUrl,
                    label: product.name,
                  ),
                ),
              ),
              const SizedBox(height: 6),
              Text(
                product.name,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 13,
                  height: 1.3,
                  fontWeight: FontWeight.w600,
                  color: HeritageColors.muc,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                product.price.format(),
                style: GoogleFonts.playfairDisplay(
                  fontSize: 15,
                  fontWeight: FontWeight.w700,
                  color: HeritageColors.doChuSa,
                  fontFeatures: const [FontFeature.liningFigures()],
                ),
              ),
            ],
          ),
        ),
      ),
    ),
  );
}
