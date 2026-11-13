import 'package:flutter/material.dart';
import 'package:mobile_app/core/theme/app_colors.dart';

abstract final class AppSnackBar {
  static void success(BuildContext context, String message) =>
      _show(context, message, AppColors.success, Icons.check_circle_outline);
  static void error(BuildContext context, String message) =>
      _show(context, message, AppColors.brickRed, Icons.error_outline);
  static void info(BuildContext context, String message) =>
      _show(context, message, AppColors.brandBrown, Icons.info_outline);
  static void _show(
    BuildContext context,
    String message,
    Color color,
    IconData icon,
  ) => ScaffoldMessenger.of(context).showSnackBar(
    SnackBar(
      behavior: SnackBarBehavior.floating,
      backgroundColor: color,
      content: Row(
        children: [
          Icon(icon, color: Colors.white),
          const SizedBox(width: 8),
          Expanded(child: Text(message)),
        ],
      ),
    ),
  );
}
