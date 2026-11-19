import 'package:flutter/material.dart';
import 'package:mobile_app/core/theme/app_colors.dart';

class AppLoadingView extends StatelessWidget {
  const AppLoadingView({super.key, this.label = 'Đang tải...'});
  final String label;
  @override
  Widget build(BuildContext context) => Center(
    child: Semantics(
      label: label,
      liveRegion: true,
      child: const CircularProgressIndicator(color: AppColors.brandBrown),
    ),
  );
}
