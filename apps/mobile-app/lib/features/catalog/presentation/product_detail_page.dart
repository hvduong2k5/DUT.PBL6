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
import 'package:mobile_app/core/widgets/app_product_image.dart';
import 'package:mobile_app/features/reviews/domain/reviews_repository.dart';
import 'package:mobile_app/features/reviews/presentation/reviews_cubits.dart';
import 'package:mobile_app/features/reviews/presentation/product_reviews_section.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';

class ProductDetailPage extends StatefulWidget {
  const ProductDetailPage({super.key, required this.productId});
  final String productId;
  @override
  State<ProductDetailPage> createState() => _ProductDetailPageState();
}

class _ProductDetailPageState extends State<ProductDetailPage> {
  late Future<Product> _future;
  int _quantity = 1;
  ProductVariant? _selectedVariant;
  int _imageIndex = 0;
  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void didUpdateWidget(covariant ProductDetailPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.productId != widget.productId) {
      _quantity = 1;
      _load();
    }
  }

  void _load() {
    _selectedVariant = null;
    _imageIndex = 0;
    _future = context.read<CatalogRepository>().product(widget.productId);
  }

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
          final images = product.images.isEmpty
              ? [product.imageUrl]
              : product.images;
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
                clipBehavior: Clip.antiAlias,
                child: AppProductImage(
                  url: images[_imageIndex],
                  label: product.name,
                ),
              ),
              if (images.length > 1) ...[
                const SizedBox(height: 12),
                SizedBox(
                  height: 76,
                  child: ListView.separated(
                    scrollDirection: Axis.horizontal,
                    itemCount: images.length,
                    separatorBuilder: (_, _) => const SizedBox(width: 8),
                    itemBuilder: (context, index) => Semantics(
                      selected: index == _imageIndex,
                      button: true,
                      label: 'Ảnh sản phẩm ${index + 1} trên ${images.length}',
                      child: InkWell(
                        onTap: () => setState(() => _imageIndex = index),
                        child: Container(
                          width: 76,
                          padding: const EdgeInsets.all(3),
                          decoration: BoxDecoration(
                            border: Border.all(
                              color: index == _imageIndex
                                  ? AppColors.brandBrown
                                  : AppColors.border,
                            ),
                            borderRadius: BorderRadius.circular(
                              AppRadius.medium,
                            ),
                          ),
                          child: AppProductImage(
                            url: images[index],
                            label: product.name,
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ],
              const SizedBox(height: AppSpacing.md),
              Wrap(
                spacing: 20,
                runSpacing: 8,
                children: [
                  if (product.ocopStars > 0)
                    Text(
                      '${product.ocopStars} SAO OCOP',
                      style: const TextStyle(
                        fontSize: 11,
                        fontWeight: FontWeight.bold,
                        color: AppColors.brandBrown,
                      ),
                    ),
                  Text(
                    product.availabilityLabel,
                    style: const TextStyle(
                      fontSize: 11,
                      color: AppColors.success,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ],
              ),
              if (product.certificateNumber.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: SelectableText(
                    'Mã chứng nhận OCOP: ${product.certificateNumber}',
                  ),
                ),
              const SizedBox(height: AppSpacing.xs),
              Text(
                product.name,
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              if (_selectedVariant?.originalPrice != null &&
                  _selectedVariant!.originalPrice!.compareTo(
                        _selectedVariant!.price,
                      ) >
                      0)
                Text(
                  _selectedVariant!.originalPrice!.format(),
                  style: const TextStyle(
                    decoration: TextDecoration.lineThrough,
                    color: AppColors.textMuted,
                  ),
                ),
              const SizedBox(height: AppSpacing.xs),
              Text(
                product.variants.isEmpty
                    ? 'Chưa công bố giá'
                    : (_selectedVariant?.price ?? product.price).format(),
                style: TextStyle(
                  fontFamily: GoogleFonts.notoSerif().fontFamily,
                  fontSize: 24,
                  color: AppColors.brickRed,
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(height: AppSpacing.md),
              Text(
                'Chọn quy cách',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: product.variants
                    .map(
                      (variant) => ChoiceChip(
                        label: Text(
                          '${variant.name} · ${variant.stockKnown ? (variant.stock > 0 ? variant.price.format() : 'Hết hàng') : 'Đang cập nhật tồn kho'}',
                        ),
                        selected: _selectedVariant?.id == variant.id,
                        onSelected: variant.stock < 1 || cartState.pending
                            ? null
                            : (_) => setState(() {
                                _selectedVariant = variant;
                                _quantity = 1;
                              }),
                      ),
                    )
                    .toList(),
              ),
              if (product.variants.isEmpty)
                const Text('Chưa có quy cách khả dụng.'),
              const SizedBox(height: 12),
              if (_selectedVariant == null)
                const Text(
                  'Chọn quy cách đã công bố để tiếp tục. Giỏ hàng không giữ tồn hoặc cố định giá.',
                )
              else ...[
                SelectableText('SKU: ${_selectedVariant!.sku}'),
                if (_selectedVariant!.weightGrams != null)
                  Text('Khối lượng: ${_selectedVariant!.weightGrams}g'),
              ],
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
                    onPressed:
                        _quantity > 1 &&
                            !cartState.pending &&
                            _selectedVariant != null
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
                    onPressed:
                        cartState.pending ||
                            _quantity >= maxCartItemQuantity ||
                            _selectedVariant == null ||
                            _quantity >= _selectedVariant!.stock
                        ? null
                        : () => setState(() => _quantity++),
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
              const Divider(height: 36),
              Text(
                'Thông tin thực phẩm',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 12),
              const Text(
                'Thông tin đang được cập nhật. Chưa có dữ liệu thành phần, cảnh báo dị ứng và hướng dẫn bảo quản. Vui lòng kiểm tra nhãn trên bao bì hoặc tra cứu lô hàng trước khi sử dụng.',
              ),
              TextButton(
                onPressed: () => context.push(AppRoutes.trace),
                child: const Text('Tra cứu lô hàng'),
              ),
              BlocProvider(
                key: ValueKey('reviews:${product.id}'),
                create: (_) =>
                    ReviewsCubit(context.read<ReviewsRepository>(), product.id)
                      ..load(),
                child: ProductReviewsSection(product: product),
              ),
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
                  Expanded(
                    child: AppPrimaryButton(
                      label: 'Thêm vào giỏ',
                      loading: cartState.pending,
                      icon: Icons.shopping_bag_outlined,
                      onPressed:
                          product == null ||
                              _selectedVariant == null ||
                              !product.inStock
                          ? null
                          : () async {
                              final added = await context.read<CartCubit>().add(
                                product,
                                _selectedVariant!,
                                quantity: _quantity,
                              );
                              if (!context.mounted) return;
                              if (added) {
                                AppSnackBar.success(
                                  context,
                                  'Đã thêm vào giỏ hàng',
                                );
                              } else {
                                AppSnackBar.error(
                                  context,
                                  context.read<CartCubit>().state.message ??
                                      'Chưa thể thêm sản phẩm.',
                                );
                              }
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
