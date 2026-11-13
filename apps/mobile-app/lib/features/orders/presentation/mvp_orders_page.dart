import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/features/orders/presentation/mvp_orders_cubit.dart';

class MvpOrdersPage extends StatelessWidget {
  const MvpOrdersPage({super.key, this.local = false});
  final bool local;
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Đơn hàng của tôi', showBack: true),
    body: SafeArea(
      child: BlocBuilder<MvpOrdersCubit, MvpOrdersState>(
        builder: (context, state) {
          final cubit = context.read<MvpOrdersCubit>();
          if (state.loading) return const AppLoadingView();
          if (state.message != null) {
            return AppErrorView(message: state.message!, onRetry: cubit.load);
          }
          final page = state.page!;
          if (page.items.isEmpty) {
            return AppEmptyView(
              title: 'Chưa có đơn hàng',
              message: local
                  ? 'Các đơn thử nghiệm trong phiên này sẽ xuất hiện ở đây.'
                  : 'Đơn đã tạo sẽ được hiển thị ở đây.',
              action: TextButton(
                onPressed: () => context.go('/cart'),
                child: const Text('Xem giỏ hàng'),
              ),
            );
          }
          return RefreshIndicator(
            onRefresh: cubit.load,
            child: ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.all(16),
              children: [
                if (local)
                  const Text('Đơn thử nghiệm local trong phiên chạy app.'),
                for (final order in page.items)
                  Card(
                    child: ListTile(
                      title: Text(order.code),
                      subtitle: Text(
                        '${order.statusLabel}\n${DateFormat('dd/MM/yyyy HH:mm').format(order.createdAt.toLocal())}',
                      ),
                      isThreeLine: true,
                      trailing: Text(order.total.format()),
                      onTap: () async {
                        await context.push('/orders/${order.id}');
                        if (context.mounted) await cubit.load();
                      },
                    ),
                  ),
                if (page.nextCursor.isNotEmpty)
                  const Padding(
                    padding: EdgeInsets.all(12),
                    child: Text(
                      'Danh sách hiện chưa hiển thị hết lịch sử đơn hàng.',
                    ),
                  ),
              ],
            ),
          );
        },
      ),
    ),
  );
}
