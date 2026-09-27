import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_radius.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_snackbar.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/features/catalog/domain/catalog_repository.dart';

class ProductDetailPage extends StatefulWidget {
  const ProductDetailPage({super.key, required this.productId});
  final String productId;
  @override
  State<ProductDetailPage> createState() => _ProductDetailPageState();
}

class _ProductDetailPageState extends State<ProductDetailPage> {
  late Future<Product> _future;
  int _quantity = 1;
  @override
  void initState() {
    super.initState();
    _load();
  }

  void _load() =>
      _future = context.read<CatalogRepository>().product(widget.productId);
  @override
  Widget build(BuildContext context) => BlocBuilder<CartCubit, CartState>(
    builder: (context, cartState) => Scaffold(
      appBar: AppMobileHeader(
        title: 'Chi Tiết Sản Phẩm',
        showBack: true,
        cartCount: cartState.cart.totalItems,
        onCart: () => context.go(AppRoutes.cart),
      ),
      body: FutureBuilder<Product>(
        future: _future,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const AppLoadingView();
          }
          if (snapshot.hasError || snapshot.data == null) {
            return AppErrorView(
              message: 'Không thể tải thông tin sản phẩm.',
              onRetry: () => setState(_load),
            );
          }
          final product = snapshot.data!;
          return ListView(
            padding: const EdgeInsets.fromLTRB(
              AppSpacing.md,
              AppSpacing.xs,
              AppSpacing.md,
              110,
            ),
            children: [
              Container(
                height: 310,
                decoration: BoxDecoration(
                  color: AppColors.muted,
                  borderRadius: BorderRadius.circular(AppRadius.card),
                ),
                child: const Icon(
                  Icons.redeem_outlined,
                  size: 100,
                  color: AppColors.brandBrown,
                ),
              ),
              const SizedBox(height: AppSpacing.md),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    '${product.ocopStars} SAO OCOP',
                    style: const TextStyle(
                      fontSize: 11,
                      fontWeight: FontWeight.bold,
                      color: AppColors.brandBrown,
                    ),
                  ),
                  if (product.inStock)
                    const Text(
                      'CÒN HÀNG',
                      style: TextStyle(
                        fontSize: 11,
                        color: AppColors.success,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                ],
              ),
              const SizedBox(height: AppSpacing.xs),
              Text(
                product.name,
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: AppSpacing.xs),
              Text(
                product.price.format(),
                style: TextStyle(
                  fontFamily: GoogleFonts.notoSerif().fontFamily,
                  fontSize: 24,
                  color: AppColors.brickRed,
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(height: AppSpacing.md),
              Row(
                children: [
                  const Expanded(
                    child: Text(
                      'Số lượng',
                      style: TextStyle(fontWeight: FontWeight.w600),
                    ),
                  ),
                  IconButton(
                    onPressed: _quantity > 1
                        ? () => setState(() => _quantity--)
                        : null,
                    icon: const Icon(Icons.remove),
                    constraints: const BoxConstraints.tightFor(
                      width: 44,
                      height: 44,
                    ),
                  ),
                  SizedBox(
                    width: 40,
                    child: Text('$_quantity', textAlign: TextAlign.center),
                  ),
                  IconButton(
                    onPressed: () => setState(() => _quantity++),
                    icon: const Icon(Icons.add),
                    constraints: const BoxConstraints.tightFor(
                      width: 44,
                      height: 44,
                    ),
                  ),
                ],
              ),
              const Divider(height: AppSpacing.xl),
              Text(
                'Câu chuyện sản phẩm',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: AppSpacing.xs),
              Text(product.description),
              if (product.story.isNotEmpty) ...[
                const SizedBox(height: AppSpacing.sm),
                Text(product.story),
              ],
            ],
          );
        },
      ),
      bottomSheet: FutureBuilder<Product>(
        future: _future,
        builder: (context, snapshot) {
          final product = snapshot.data;
          return SafeArea(
            top: false,
            child: Container(
              padding: const EdgeInsets.symmetric(
                horizontal: AppSpacing.md,
                vertical: AppSpacing.sm,
              ),
              decoration: const BoxDecoration(
                color: AppColors.ivory,
                border: Border(top: BorderSide(color: AppColors.border)),
              ),
              child: Row(
                children: [
                  IconButton(
                    onPressed: () {},
                    icon: const Icon(Icons.bookmark_border),
                    color: AppColors.brandBrown,
                  ),
                  const SizedBox(width: AppSpacing.sm),
                  Expanded(
                    child: AppPrimaryButton(
                      label: 'Thêm vào giỏ',
                      icon: Icons.shopping_bag_outlined,
                      onPressed: product == null
                          ? null
                          : () {
                              context.read<CartCubit>().add(
                                product,
                                product.variants.first,
                                quantity: _quantity,
                              );
                              AppSnackBar.success(
                                context,
                                'Đã thêm vào giỏ hàng',
                              );
                            },
                    ),
                  ),
                ],
              ),
            ),
          );
        },
      ),
    ),
  );
}
