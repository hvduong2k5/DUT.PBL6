import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/features/notifications/presentation/notifications_cubit.dart';

class NotificationsPage extends StatelessWidget {
  const NotificationsPage({super.key});
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Thông báo', showBack: true),
    body: BlocBuilder<NotificationsCubit, NotificationsState>(
      builder: (context, state) {
        if (state.loading) return const AppLoadingView();
        if (state.items.isEmpty) {
          return const AppEmptyView(
            title: 'Không có thông báo mới',
            message:
                'Các cập nhật đơn hàng và thông báo mới sẽ xuất hiện tại đây.',
            icon: Icons.notifications_none,
          );
        }
        return ListView(
          padding: const EdgeInsets.all(AppSpacing.md),
          children: [
            const Text('Cập nhật đơn hàng và thông tin dành cho bạn'),
            const SizedBox(height: AppSpacing.md),
            for (final item in state.items)
              Card(
                color: item.unread ? AppColors.muted : AppColors.surface,
                child: ListTile(
                  minTileHeight: 94,
                  leading: CircleAvatar(
                    backgroundColor: AppColors.softYellow,
                    child: const Icon(
                      Icons.receipt_long_outlined,
                      color: AppColors.brandBrown,
                    ),
                  ),
                  title: Text(
                    item.title,
                    style: const TextStyle(fontWeight: FontWeight.w600),
                  ),
                  subtitle: Text(item.message),
                  trailing: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Text(
                        item.timeLabel,
                        style: const TextStyle(fontSize: 10),
                      ),
                      if (item.unread)
                        const Padding(
                          padding: EdgeInsets.only(top: AppSpacing.sm),
                          child: Badge(),
                        ),
                    ],
                  ),
                ),
              ),
          ],
        );
      },
    ),
  );
}
