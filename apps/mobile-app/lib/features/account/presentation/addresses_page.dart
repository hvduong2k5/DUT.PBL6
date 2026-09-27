import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/account/presentation/addresses_cubit.dart';

class AddressesPage extends StatelessWidget {
  const AddressesPage({super.key});

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Địa chỉ giao hàng', showBack: true),
    body: BlocBuilder<AddressesCubit, AddressesState>(
      builder: (context, state) {
        if (state.status == AddressesStatus.initial ||
            state.status == AddressesStatus.loading) {
          return const AppLoadingView();
        }
        if (state.addresses.isEmpty) {
          return AppEmptyView(
            title: 'Chưa có địa chỉ',
            message: 'Thêm địa chỉ để việc thanh toán và giao hàng nhanh hơn.',
            icon: Icons.location_on_outlined,
            action: FilledButton(
              onPressed: () async {
                await context.push(AppRoutes.addressForm);
                if (context.mounted) {
                  await context.read<AddressesCubit>().load();
                }
              },
              child: const Text('Thêm địa chỉ'),
            ),
          );
        }
        return ListView(
          padding: const EdgeInsets.all(16),
          children: [
            for (final address in state.addresses)
              _AddressCard(address: address),
            const SizedBox(height: 8),
            AppPrimaryButton(
              onPressed: () async {
                await context.push(AppRoutes.addressForm);
                if (context.mounted) {
                  await context.read<AddressesCubit>().load();
                }
              },
              icon: Icons.add,
              label: 'Thêm địa chỉ mới',
            ),
          ],
        );
      },
    ),
  );
}

class _AddressCard extends StatelessWidget {
  const _AddressCard({required this.address});
  final Address address;
  @override
  Widget build(BuildContext context) => Card(
    child: Padding(
      padding: const EdgeInsets.all(AppSpacing.md),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  '${address.recipientName} • ${address.phoneNumber}',
                  style: const TextStyle(fontWeight: FontWeight.bold),
                ),
              ),
              if (address.isDefault)
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    border: Border.all(color: AppColors.border),
                    borderRadius: BorderRadius.circular(16),
                  ),
                  child: const Text('Mặc định', style: TextStyle(fontSize: 10)),
                ),
            ],
          ),
          Text(address.formatted),
          const Divider(),
          Row(
            children: [
              TextButton.icon(
                onPressed: () async {
                  await context.push(AppRoutes.addressForm, extra: address);
                  if (context.mounted) {
                    await context.read<AddressesCubit>().load();
                  }
                },
                icon: const Icon(Icons.edit_outlined),
                label: const Text('Chỉnh sửa'),
              ),
              TextButton.icon(
                onPressed: () =>
                    context.read<AddressesCubit>().delete(address.id),
                icon: const Icon(Icons.delete_outline),
                label: const Text('Xóa'),
              ),
              if (!address.isDefault)
                TextButton(
                  onPressed: () =>
                      context.read<AddressesCubit>().setDefault(address.id),
                  child: const Text('Đặt mặc định'),
                ),
            ],
          ),
        ],
      ),
    ),
  );
}
