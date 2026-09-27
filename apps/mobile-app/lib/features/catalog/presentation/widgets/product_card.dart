import 'package:flutter/material.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_radius.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';

class ProductCard extends StatelessWidget {
  const ProductCard({
    super.key,
    required this.product,
    required this.onOpen,
    required this.onAdd,
  });
  final Product product;
  final VoidCallback onOpen;
  final VoidCallback onAdd;
  @override
  Widget build(BuildContext context) => Card(
    clipBehavior: Clip.antiAlias,
    margin: EdgeInsets.zero,
    shape: RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(AppRadius.card),
    ),
    child: InkWell(
      onTap: onOpen,
      child: Padding(
        padding: const EdgeInsets.all(10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(
              child: Container(
                width: double.infinity,
                decoration: BoxDecoration(
                  color: AppColors.muted,
                  borderRadius: BorderRadius.circular(AppRadius.large),
                ),
                child: const Icon(
                  Icons.redeem_outlined,
                  size: 52,
                  color: AppColors.brandBrown,
                ),
              ),
            ),
            const SizedBox(height: 8),
            Text(
              product.category.toUpperCase(),
              maxLines: 1,
              style: const TextStyle(
                color: AppColors.textMuted,
                fontSize: 10,
                fontWeight: FontWeight.w700,
              ),
            ),
            Text(
              product.name,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(
                context,
              ).textTheme.titleMedium?.copyWith(fontSize: 13),
            ),
            const SizedBox(height: 4),
            Text(
              product.price.format(),
              style: const TextStyle(
                color: AppColors.brickRed,
                fontFamily: 'Noto Serif',
                fontSize: 17,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 6),
            SizedBox(
              width: double.infinity,
              height: 44,
              child: FilledButton.icon(
                onPressed: product.inStock ? onAdd : null,
                style: FilledButton.styleFrom(
                  backgroundColor: AppColors.brandBrown,
                  foregroundColor: Colors.white,
                ),
                icon: const Icon(Icons.add_shopping_cart, size: 16),
                label: const Text('Thêm Giỏ', style: TextStyle(fontSize: 13)),
              ),
            ),
          ],
        ),
      ),
    ),
  );
}
