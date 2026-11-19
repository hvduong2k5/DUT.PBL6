import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/core/widgets/app_bottom_navigation.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_search_bar.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_surface.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/widgets/product_card.dart';
import 'package:mobile_app/features/catalog/presentation/widgets/product_filter_bar.dart';

/// "Danh mục" tab: the single browse screen for every entry point (bottom
/// nav, Home banner, "Xem tất cả", collection cards). Filter-based, no rail.
class CatalogPage extends StatelessWidget {
  const CatalogPage({super.key, this.initialQuery = ''});
  final String initialQuery;

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: HeritageSearchHeader(
      query: context.select((CatalogCubit c) => c.state.query),
    ),
    body: HeritageSurface(
      child: Column(
        key: const Key('catalog-browse'),
        children: [
          const ProductFilterBar(),
          Expanded(
            child: BlocBuilder<CatalogCubit, CatalogState>(
              builder: (context, state) => switch (state.status) {
                CatalogStatus.initial ||
                CatalogStatus.loading => const AppLoadingView(),
                CatalogStatus.failure => AppErrorView(
                  message: state.message ?? 'Không thể tải danh sách sản phẩm.',
                  onRetry: context.read<CatalogCubit>().load,
                ),
                _ => CatalogResults(state),
              },
            ),
          ),
        ],
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

/// Result count + clear, product grid and pagination. Shared with search.
class CatalogResults extends StatelessWidget {
  const CatalogResults(this.state, {super.key});
  final CatalogState state;
  @override
  Widget build(BuildContext context) {
    final products = state.products;
    return CustomScrollView(
      slivers: [
        SliverPadding(
          padding: const EdgeInsets.fromLTRB(16, 8, 8, 4),
          sliver: SliverToBoxAdapter(
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    state.complete
                        ? '${products.length} sản phẩm phù hợp'
                        : '${products.length} phù hợp trong ${state.loadedProducts.length}/${state.total} kết quả đã tải',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                ),
                if (state.filtering || state.query.isNotEmpty)
                  TextButton(
                    onPressed: context.read<CatalogCubit>().clearAll,
                    child: const Text('Xóa tất cả'),
                  ),
              ],
            ),
          ),
        ),
        if (products.isEmpty)
          SliverToBoxAdapter(
            child: AppEmptyView(
              title: 'Không tìm thấy sản phẩm',
              message: state.hasMore
                  ? 'Thử bớt bộ lọc hoặc tải thêm kết quả.'
                  : 'Thử từ khóa khác hoặc bớt bộ lọc.',
              icon: Icons.search_off,
            ),
          ),
        SliverPadding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          sliver: SliverGrid.builder(
            itemCount: products.length,
            gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: 2,
              childAspectRatio:
                  .60 / MediaQuery.textScalerOf(context).scale(1).clamp(1, 2),
              crossAxisSpacing: 12,
              mainAxisSpacing: 12,
            ),
            itemBuilder: (context, index) {
              final product = products[index];
              return ProductCard(
                product: product,
                onOpen: () =>
                    context.push('/product/${Uri.encodeComponent(product.id)}'),
                onAdd: () =>
                    context.push('/product/${Uri.encodeComponent(product.id)}'),
              );
            },
          ),
        ),
        SliverToBoxAdapter(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              children: [
                if (state.pageMessage != null)
                  Semantics(liveRegion: true, child: Text(state.pageMessage!)),
                if (state.hasMore)
                  OutlinedButton(
                    onPressed: state.loadingMore
                        ? null
                        : context.read<CatalogCubit>().loadMore,
                    child: state.loadingMore
                        ? const Text('Đang tải…')
                        : Text(
                            state.pageMessage == null
                                ? 'Tải thêm sản phẩm'
                                : 'Thử tải thêm lần nữa',
                          ),
                  ),
                if (state.filtering && !state.complete)
                  const Text(
                    'Bộ lọc và thứ tự giá áp dụng trên các sản phẩm đã tải.',
                  ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}
