import 'package:flutter/material.dart';
import 'package:mobile_app/core/theme/app_colors.dart';

class AppOfflineBanner extends StatelessWidget {
  const AppOfflineBanner({super.key});
  @override
  Widget build(BuildContext context) => Container(
    height: 36,
    width: double.infinity,
    color: AppColors.honeyGold.withValues(alpha: .18),
    alignment: Alignment.center,
    padding: const EdgeInsets.symmetric(horizontal: 12),
    child: const Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(Icons.wifi_off, size: 15),
        SizedBox(width: 6),
        Flexible(
          child: Text(
            'Đang ngoại tuyến • Dữ liệu có thể là bản đã lưu',
            style: TextStyle(fontSize: 11),
          ),
        ),
      ],
    ),
  );
}
