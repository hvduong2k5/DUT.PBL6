import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/features/account/domain/customer_account_repository.dart';

class LoyaltyPage extends StatefulWidget {
  const LoyaltyPage({super.key});
  @override
  State<LoyaltyPage> createState() => _LoyaltyPageState();
}

class _LoyaltyPageState extends State<LoyaltyPage> {
  late Future<LoyaltyOverview> _future;
  @override
  void initState() {
    super.initState();
    _load();
  }

  void _load() {
    _future = context.read<CustomerAccountRepository>().loyalty();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Ví điểm & đặc quyền', showBack: true),
    body: FutureBuilder<LoyaltyOverview>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done) {
          return const AppLoadingView();
        }
        if (snapshot.hasError || snapshot.data == null) {
          return AppErrorView(
            message: snapshot.error is ApiException
                ? (snapshot.error as ApiException).failure.message
                : 'Chưa thể mở ví điểm.',
            onRetry: () => setState(_load),
          );
        }
        final overview = snapshot.data!;
        final value = Money(
          currencyCode: 'VND',
          units: BigInt.from(overview.points) * BigInt.from(overview.vndRate),
        );
        return ListView(
          padding: const EdgeInsets.all(20),
          children: [
            Text(
              'Ví điểm & đặc quyền',
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: 20),
            Container(
              padding: const EdgeInsets.all(24),
              decoration: BoxDecoration(
                color: AppColors.headingBrown,
                borderRadius: BorderRadius.circular(16),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Ô MẠ · TRI KỶ    Hạng ${overview.tierLabel}',
                    style: const TextStyle(color: Colors.white),
                  ),
                  const SizedBox(height: 28),
                  const Text(
                    'Điểm khả dụng',
                    style: TextStyle(color: Colors.white70),
                  ),
                  Text(
                    '${overview.points}',
                    style: Theme.of(context).textTheme.headlineLarge?.copyWith(
                      color: AppColors.honeyGold,
                    ),
                  ),
                  const Text(
                    'Điểm Di Sản',
                    style: TextStyle(color: Colors.white70),
                  ),
                  const SizedBox(height: 24),
                  SelectableText(
                    overview.customerId,
                    style: const TextStyle(color: Colors.white),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 24),
            Text(
              'Giá trị quy đổi hiện tại',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            Text(
              value.format(),
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            Text(
              'Tạm tính theo tỷ lệ 1 điểm = ${Money.vnd(overview.vndRate).format()}.',
            ),
            const SizedBox(height: 12),
            const Text(
              'Khả năng sử dụng điểm và số tiền cuối cùng được xác nhận theo chính sách khi đặt hàng.',
            ),
            const SizedBox(height: 24),
            AppPrimaryButton(
              label: 'Khám phá ưu đãi',
              onPressed: () => context.push(AppRoutes.promotions),
            ),
          ],
        );
      },
    ),
  );
}
