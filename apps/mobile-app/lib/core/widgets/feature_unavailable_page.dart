import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/config/app_features.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_secondary_button.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_surface.dart';

/// Shown instead of calling an endpoint the backend does not expose yet.
class FeatureUnavailablePage extends StatelessWidget {
  const FeatureUnavailablePage({super.key, required this.feature});
  final AppFeature? feature;

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppMobileHeader(
      title: feature?.label ?? 'Chưa khả dụng',
      showBack: true,
    ),
    body: HeritageSurface(
      child: AppEmptyView(
        key: const Key('feature-unavailable'),
        title: 'Tính năng chưa khả dụng',
        message:
            'Hệ thống Gói Huế chưa mở ${feature == null ? 'tính năng này' : '"${feature!.label}"'}. Tính năng sẽ bật khi máy chủ sẵn sàng.',
        action: SizedBox(
          width: 220,
          child: AppSecondaryButton(
            label: 'Về trang chủ',
            onPressed: () => context.go(AppRoutes.home),
          ),
        ),
      ),
    ),
  );
}
