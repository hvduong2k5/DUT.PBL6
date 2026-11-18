import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/widgets/app_product_image.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_button.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_frame.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';

/// Kit "card" frame (single line); the product opens on tap.
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
  Widget build(BuildContext context) => InkWell(
    onTap: onOpen,
    child: HeritageFrame(
      level: HeritageFrameLevel.card,
      padding: const EdgeInsets.all(10),
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
          const SizedBox(height: 8),
          Text(
            product.category.toUpperCase(),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(
              color: AppColors.textMuted,
              fontSize: 13,
              fontWeight: FontWeight.w700,
              letterSpacing: .6,
            ),
          ),
          Text(
            product.name,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: Theme.of(
              context,
            ).textTheme.titleMedium?.copyWith(fontSize: 14),
          ),
          const SizedBox(height: 4),
          Text(
            product.price.format(),
            style: GoogleFonts.playfairDisplay(
              color: AppColors.brickRed,
              fontSize: 17,
              fontWeight: FontWeight.w700,
              fontFeatures: const [FontFeature.liningFigures()],
            ),
          ),
          const SizedBox(height: 2),
          Text(
            product.availabilityLabel,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(fontSize: 13, color: AppColors.textMuted),
          ),
          const SizedBox(height: 8),
          HeritageButton(
            label: product.stockKnown && !product.inStock
                ? 'Xem sản phẩm'
                : 'Chọn quy cách',
            onPressed: onAdd,
          ),
        ],
      ),
    ),
  );
}
