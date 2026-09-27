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
import 'package:mobile_app/core/widgets/app_snackbar.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/widgets/product_card.dart';

class HomePage extends StatelessWidget {
  const HomePage({super.key});

  @override
  Widget build(BuildContext context) => BlocBuilder<CartCubit, CartState>(
    builder: (context, cartState) => Scaffold(
      appBar: AppMobileHeader(
        showMenu: true,
        onSearch: () => context.go(AppRoutes.catalog),
        cartCount: cartState.cart.totalItems,
        onCart: () => context.go(AppRoutes.cart),
      ),
      drawer: const AppNavigationDrawer(activeRoute: AppRoutes.home),
      body: BlocBuilder<CatalogCubit, CatalogState>(
        builder: (context, state) => Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(
                AppSpacing.md,
                AppSpacing.sm,
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
                  key: const Key('home-search-field'),
                  textInputAction: TextInputAction.search,
                  onSubmitted: context.read<CatalogCubit>().search,
                  decoration: const InputDecoration(
                    isDense: true,
                    border: InputBorder.none,
                    hintText: 'Tìm mứt sen, mè xửng giòn...',
                    prefixIcon: Icon(Icons.search, size: 20),
                    suffixIcon: Icon(Icons.mic_none, size: 18),
                    contentPadding: EdgeInsets.symmetric(vertical: 10),
                  ),
                ),
              ),
            ),
            if (state.categories.isNotEmpty)
              SizedBox(
                height: 42,
                child: ListView.separated(
                  key: const Key('home-category-chips'),
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  scrollDirection: Axis.horizontal,
                  itemCount: state.categories.length,
                  separatorBuilder: (_, _) => const SizedBox(width: 8),
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
                ),
              ),
            const SizedBox(height: 8),
            Expanded(child: _HomeResults(state: state)),
          ],
        ),
      ),
      bottomNavigationBar: AppBottomNavigation(
        activeIndex: 0,
        cartCount: cartState.cart.totalItems,
        onDestinationSelected: (index) {
          if (index == 1) context.go(AppRoutes.catalog);
          if (index == 2) context.go(AppRoutes.cart);
          if (index == 3) context.go(AppRoutes.account);
        },
      ),
    ),
  );
}

class _HomeResults extends StatelessWidget {
  const _HomeResults({required this.state});

  final CatalogState state;

  @override
  Widget build(BuildContext context) {
    if (state.status == CatalogStatus.initial ||
        state.status == CatalogStatus.loading) {
      return const AppLoadingView();
    }
    if (state.status == CatalogStatus.failure) {
      return AppErrorView(
        message: state.message ?? 'Không thể tải trang chủ.',
        onRetry: context.read<CatalogCubit>().load,
      );
    }
    if (state.products.isEmpty) {
      return AppEmptyView(
        title: 'Chưa có sản phẩm phù hợp',
        message: 'Thử từ khóa khác hoặc chọn lại danh mục.',
        icon: Icons.search_off,
        action: OutlinedButton(
          onPressed: () => context.read<CatalogCubit>().search(''),
          child: const Text('Xóa tìm kiếm'),
        ),
      );
    }

    final featured = state.products.take(2).toList();
    return CustomScrollView(
      key: const Key('home-content-scroll'),
      slivers: [
        SliverPadding(
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
          sliver: SliverList.list(
            children: [
              Container(
                constraints: const BoxConstraints(minHeight: 114),
                padding: const EdgeInsets.all(14),
                decoration: BoxDecoration(
                  gradient: const LinearGradient(
                    colors: [AppColors.brandBrown, AppColors.gradientBrownEnd],
                  ),
                  borderRadius: BorderRadius.circular(AppRadius.card),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    const Text(
                      'MÙA VỤ BÁNH CUNG ĐÌNH 2026',
                      style: TextStyle(
                        color: AppColors.honeyGold,
                        fontSize: 10,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      'Vị Ngọt Thanh Tao Cố Đô — Đậm Tình Xứ Huế',
                      style: Theme.of(context).textTheme.titleLarge?.copyWith(
                        color: Colors.white,
                        fontSize: 17,
                      ),
                    ),
                    const SizedBox(height: 4),
                    const Text(
                      'Kẹo cau truyền thống, mè xửng dẻo và trà sen.',
                      style: TextStyle(color: Colors.white70, fontSize: 11),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 16),
              Row(
                children: [
                  const Icon(Icons.circle, size: 8, color: AppColors.brickRed),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text(
                      'Món Mới Ra Lò Cố Đô',
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                  ),
                  TextButton(
                    onPressed: () => context.go(AppRoutes.catalog),
                    child: const Text('Xem thêm →'),
                  ),
                ],
              ),
            ],
          ),
        ),
        SliverPadding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          sliver: SliverGrid.builder(
            itemCount: featured.length,
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: 2,
              childAspectRatio: .60,
              crossAxisSpacing: 12,
              mainAxisSpacing: 12,
            ),
            itemBuilder: (context, index) {
              final product = featured[index];
              return ProductCard(
                product: product,
                onOpen: () => context.push('/product/${product.id}'),
                onAdd: () {
                  context.read<CartCubit>().add(
                    product,
                    product.variants.first,
                  );
                  AppSnackBar.success(context, 'Đã thêm ${product.name}');
                },
              );
            },
          ),
        ),
        SliverPadding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
          sliver: SliverToBoxAdapter(
            child: Card(
              child: ListTile(
                minTileHeight: 62,
                leading: const CircleAvatar(
                  backgroundColor: AppColors.cream,
                  child: Icon(
                    Icons.local_shipping_outlined,
                    color: AppColors.honeyGold,
                  ),
                ),
                title: const Text('Theo dõi đơn hàng của bạn'),
                subtitle: const Text('Xem trạng thái và hành trình giao hàng'),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => context.push(AppRoutes.orders),
              ),
            ),
          ),
        ),
      ],
    );
  }
}
