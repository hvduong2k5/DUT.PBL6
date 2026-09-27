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
import 'package:mobile_app/core/widgets/app_snackbar.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/widgets/product_card.dart';

class CatalogPage extends StatefulWidget {
  const CatalogPage({super.key});
  @override
  State<CatalogPage> createState() => _CatalogPageState();
}

class _CatalogPageState extends State<CatalogPage> {
  Timer? _debounce;
  @override
  void dispose() {
    _debounce?.cancel();
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
            if (state.categories.isNotEmpty)
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
            Expanded(
              child: switch (state.status) {
                CatalogStatus.initial ||
                CatalogStatus.loading => const AppLoadingView(),
                CatalogStatus.failure => AppErrorView(
                  message: state.message ?? 'Không thể tải thực đơn.',
                  onRetry: context.read<CatalogCubit>().load,
                ),
                CatalogStatus.success when state.products.isEmpty =>
                  AppEmptyView(
                    title: 'Không tìm thấy sản phẩm',
                    message: 'Thử từ khóa khác hoặc chọn một danh mục khác.',
                    icon: Icons.search_off,
                    action: OutlinedButton(
                      onPressed: () => context.read<CatalogCubit>().search(''),
                      child: const Text('Xóa tìm kiếm'),
                    ),
                  ),
                _ => GridView.builder(
                  padding: const EdgeInsets.fromLTRB(
                    AppSpacing.md,
                    0,
                    AppSpacing.md,
                    AppSpacing.lg,
                  ),
                  itemCount: state.products.length,
                  gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                    crossAxisCount: 2,
                    childAspectRatio: .60,
                    crossAxisSpacing: AppSpacing.sm,
                    mainAxisSpacing: AppSpacing.sm,
                  ),
                  itemBuilder: (context, index) {
                    final product = state.products[index];
                    return ProductCard(
                      product: product,
                      onOpen: () => context.push('/product/${product.id}'),
                      onAdd: () {
                        context.read<CartCubit>().add(
                          product,
                          product.variants.first,
                        );
                        AppSnackBar.success(context, 'Đã thêm vào giỏ hàng');
                      },
                    );
                  },
                ),
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
