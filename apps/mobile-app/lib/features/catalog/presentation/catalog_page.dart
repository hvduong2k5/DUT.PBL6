import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_radius.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_bottom_navigation.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_navigation_drawer.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/widgets/product_card.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_filters_sheet.dart';

class CatalogPage extends StatefulWidget {
  const CatalogPage({super.key, this.initialQuery = ''});
  final String initialQuery;
  @override
  State<CatalogPage> createState() => _CatalogPageState();
}

class _CatalogPageState extends State<CatalogPage> {
  Timer? _debounce;
  late final _search = TextEditingController(text: widget.initialQuery);
  @override
  void dispose() {
    _debounce?.cancel();
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => BlocBuilder<CartCubit, CartState>(
    builder: (context, cartState) => Scaffold(
      appBar: AppMobileHeader(
        title: 'Thực Đơn Ô Mạ',
        showMenu: true,
        cartCount: cartState.cart.totalItems,
        onCart: () => context.go(AppRoutes.cart),
      ),
      drawer: const AppNavigationDrawer(activeRoute: AppRoutes.catalog),
      body: BlocBuilder<CatalogCubit, CatalogState>(
        builder: (context, state) => Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(
                AppSpacing.md,
                AppSpacing.xs,
                AppSpacing.md,
                AppSpacing.xs,
              ),
              child: Container(
                height: 44,
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(AppRadius.large),
                  border: Border.all(color: AppColors.border),
                  boxShadow: [
                    BoxShadow(
                      color: Colors.black.withValues(alpha: 0.04),
                      blurRadius: 8,
                      offset: const Offset(0, 2),
                    ),
                  ],
                ),
                child: TextField(
                  controller: _search,
                  textInputAction: TextInputAction.search,
                  onSubmitted: (value) {
                    _debounce?.cancel();
                    context.read<CatalogCubit>().search(value);
                  },
                  onChanged: (value) {
                    _debounce?.cancel();
                    _debounce = Timer(
                      const Duration(milliseconds: 350),
                      () => context.read<CatalogCubit>().search(value),
                    );
                  },
                  decoration: const InputDecoration(
                    isDense: true,
                    border: InputBorder.none,
                    hintText: 'Tìm mè xửng, quà Huế...',
                    prefixIcon: Icon(Icons.search, size: 20),
                    contentPadding: EdgeInsets.symmetric(vertical: 10),
                  ),
                ),
              ),
            ),
            if (state.searchError != null)
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                child: Text(
                  state.searchError!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ),
            if (state.categories.isNotEmpty &&
                MediaQuery.viewInsetsOf(context).bottom == 0)
              SizedBox(
                height: 42,
                child: ListView.separated(
                  padding: const EdgeInsets.symmetric(
                    horizontal: AppSpacing.md,
                  ),
                  scrollDirection: Axis.horizontal,
                  itemBuilder: (context, index) {
                    final category = state.categories[index];
                    return ChoiceChip(
                      label: Text(category.name),
                      selected: state.categoryId == category.id,
                      selectedColor: AppColors.brandBrown,
                      backgroundColor: Colors.white,
                      labelStyle: TextStyle(
                        color: state.categoryId == category.id
                            ? Colors.white
                            : AppColors.text,
                      ),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(AppRadius.medium),
                        side: BorderSide(
                          color: state.categoryId == category.id
                              ? AppColors.brandBrown
                              : AppColors.border,
                        ),
                      ),
                      showCheckmark: false,
                      onSelected: (_) => context
                          .read<CatalogCubit>()
                          .selectCategory(category.id),
                    );
                  },
                  separatorBuilder: (_, _) =>
                      const SizedBox(width: AppSpacing.xs),
                  itemCount: state.categories.length,
                ),
              ),
            const SizedBox(height: AppSpacing.xs),
            if (MediaQuery.viewInsetsOf(context).bottom == 0)
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                child: Row(
                  children: [
                    TextButton.icon(
                      icon: const Icon(Icons.tune),
                      label: const Text('Bộ lọc'),
                      onPressed: () async {
                        final filters = await showCatalogFilters(
                          context,
                          state.filters,
                        );
                        if (context.mounted && filters != null) {
                          context.read<CatalogCubit>().setFilters(filters);
                        }
                      },
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: DropdownButton<CatalogSort>(
                        value: state.filters.sort,
                        isExpanded: true,
                        items: [
                          DropdownMenuItem(
                            value: CatalogSort.curated,
                            child: Text(
                              state.query.isEmpty
                                  ? 'Tuyển chọn'
                                  : 'Liên quan nhất',
                            ),
                          ),
                          const DropdownMenuItem(
                            value: CatalogSort.priceAscending,
                            child: Text('Giá thấp đến cao'),
                          ),
                          const DropdownMenuItem(
                            value: CatalogSort.priceDescending,
                            child: Text('Giá cao đến thấp'),
                          ),
                        ],
                        onChanged: (sort) {
                          if (sort != null) {
                            context.read<CatalogCubit>().setFilters(
                              CatalogFilters(
                                minimum: state.filters.minimum,
                                maximum: state.filters.maximum,
                                stars: state.filters.stars,
                                availableOnly: state.filters.availableOnly,
                                sort: sort,
                              ),
                            );
                          }
                        },
                      ),
                    ),
                  ],
                ),
              ),
            if (state.status == CatalogStatus.success &&
                MediaQuery.viewInsetsOf(context).bottom == 0)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        state.complete
                            ? '${state.products.length} sản phẩm phù hợp'
                            : '${state.products.length} phù hợp trong ${state.loadedProducts.length}/${state.total} kết quả đã tải',
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                    ),
                    if (state.filtering || state.query.isNotEmpty)
                      TextButton(
                        onPressed: () {
                          _search.clear();
                          _debounce?.cancel();
                          context.read<CatalogCubit>().clearAll();
                        },
                        child: const Text('Xóa tất cả'),
                      ),
                  ],
                ),
              ),
            Expanded(
              child: switch (state.status) {
                CatalogStatus.initial ||
                CatalogStatus.loading => const AppLoadingView(),
                CatalogStatus.failure => AppErrorView(
                  message: state.message ?? 'Không thể tải thực đơn.',
                  onRetry: context.read<CatalogCubit>().load,
                ),
                _ => _CatalogResults(state),
              },
            ),
          ],
        ),
      ),
      bottomNavigationBar: AppBottomNavigation(
        activeIndex: 1,
        cartCount: cartState.cart.totalItems,
        onDestinationSelected: (index) {
          if (index == 0) context.go(AppRoutes.home);
          if (index == 2) context.go(AppRoutes.cart);
          if (index == 3) context.go(AppRoutes.account);
        },
      ),
    ),
  );
}

class _CatalogResults extends StatelessWidget {
  const _CatalogResults(this.state);
  final CatalogState state;
  @override
  Widget build(BuildContext context) {
    final products = state.products;
    return CustomScrollView(
      slivers: [
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
