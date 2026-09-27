import 'package:flutter/material.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_secondary_button.dart';

class AppErrorView extends StatelessWidget {
  const AppErrorView({
    super.key,
    this.title = 'Đã xảy ra lỗi',
    required this.message,
    required this.onRetry,
    this.offline = false,
  });
  final String title;
  final String message;
  final VoidCallback onRetry;
  final bool offline;
  @override
  Widget build(BuildContext context) => Center(
    child: Padding(
      padding: const EdgeInsets.all(AppSpacing.lg),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          CircleAvatar(
            radius: 32,
            backgroundColor: AppColors.muted,
            foregroundColor: AppColors.brickRed,
            child: Icon(
              offline ? Icons.wifi_off : Icons.warning_amber_rounded,
              size: 30,
            ),
          ),
          const SizedBox(height: AppSpacing.md),
          Text(
            offline ? 'Không có kết nối mạng' : title,
            style: Theme.of(context).textTheme.titleLarge,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: AppSpacing.xs),
          Text(message, textAlign: TextAlign.center),
          const SizedBox(height: AppSpacing.lg),
          AppSecondaryButton(label: 'Thử lại', onPressed: onRetry),
        ],
      ),
    ),
  );
}
