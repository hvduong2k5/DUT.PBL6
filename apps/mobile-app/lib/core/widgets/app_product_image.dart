import 'package:flutter/material.dart';
import 'package:mobile_app/core/theme/app_colors.dart';

class AppProductImage extends StatelessWidget {
  const AppProductImage({super.key, required this.url, required this.label});
  final String url;
  final String label;
  @override
  Widget build(BuildContext context) {
    final fallback = Semantics(
      label: 'Chưa có ảnh $label',
      child: const ColoredBox(
        color: AppColors.muted,
        child: Center(
          child: Icon(
            Icons.image_outlined,
            color: AppColors.textMuted,
            size: 32,
          ),
        ),
      ),
    );
    if (url.isEmpty) return fallback;
    return Image.network(
      url,
      fit: BoxFit.cover,
      semanticLabel: label,
      errorBuilder: (_, _, _) => fallback,
      loadingBuilder: (_, child, progress) => progress == null
          ? child
          : const ColoredBox(
              color: AppColors.muted,
              child: Center(
                child: SizedBox.square(
                  dimension: 24,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              ),
            ),
    );
  }
}
