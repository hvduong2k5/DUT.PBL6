import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_radius.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/checkout/presentation/checkout_cubit.dart';

class CheckoutPage extends StatefulWidget {
  const CheckoutPage({super.key});

  @override
  State<CheckoutPage> createState() => _CheckoutPageState();
}

class _CheckoutPageState extends State<CheckoutPage> {
  final _name = TextEditingController();
  final _phone = TextEditingController();
  final _street = TextEditingController();
  final _ward = TextEditingController();
  final _district = TextEditingController();
  final _province = TextEditingController();

  void _syncAddressFields(CheckoutState state) {
    _name.text = state.fullName;
    _phone.text = state.phone;
    _street.text = state.streetAddress;
    _ward.text = state.ward;
    _district.text = state.district;
    _province.text = state.province;
  }

  @override
  void dispose() {
    for (final controller in [
      _name,
      _phone,
      _street,
      _ward,
      _district,
      _province,
    ]) {
      controller.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(
    BuildContext context,
  ) => BlocConsumer<CheckoutCubit, CheckoutState>(
    listener: (context, state) {
      if (state.useAccountAddress) _syncAddressFields(state);
      if (state.status == CheckoutStatus.success) {
        context.read<CartCubit>().clear();
        context.go(
          '/order-confirmation/${state.confirmation!.orderId}',
          extra: state.confirmation,
        );
      }
    },
    builder: (context, state) {
      final cart = context.watch<CartCubit>().state.cart;
      final authenticated =
          context.watch<AuthCubit>().state.status == AuthStatus.authenticated;
      final submitting = state.status == CheckoutStatus.submitting;
      return Scaffold(
        appBar: const AppMobileHeader(title: 'Thanh Toán', showBack: true),
        body: ListView(
          padding: const EdgeInsets.fromLTRB(
            AppSpacing.md,
            AppSpacing.xs,
            AppSpacing.md,
            110,
          ),
          children: [
            if (submitting)
              Container(
                margin: const EdgeInsets.only(bottom: AppSpacing.sm),
                padding: const EdgeInsets.all(AppSpacing.sm),
                decoration: BoxDecoration(
                  color: AppColors.muted,
                  borderRadius: BorderRadius.circular(AppRadius.large),
                ),
                child: const Text(
                  'Đang gửi đơn hàng. Vui lòng không thao tác lặp lại.',
                  style: TextStyle(fontSize: 12),
                ),
              ),
            Text(
              authenticated
                  ? 'Thanh toán bằng tài khoản'
                  : 'Thanh toán khách • Không cần tạo tài khoản',
              style: const TextStyle(
                color: AppColors.brandBrown,
                fontSize: 12,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: AppSpacing.md),
            Text(
              'Thông tin giao hàng',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: AppSpacing.xs),
            if (state.loadingSavedAddress)
              const LinearProgressIndicator(minHeight: 2),
            if (state.savedAddress != null)
              CheckboxListTile(
                key: const Key('use-account-address'),
                contentPadding: EdgeInsets.zero,
                controlAffinity: ListTileControlAffinity.leading,
                title: const Text('Dùng địa chỉ tài khoản'),
                subtitle: Text(state.savedAddress!.formatted),
                value: state.useAccountAddress,
                onChanged: submitting
                    ? null
                    : (value) => context.read<CheckoutCubit>().useSavedAddress(
                        value ?? false,
                      ),
              )
            else if (state.checkedSavedAddresses)
              const Padding(
                padding: EdgeInsets.only(bottom: AppSpacing.sm),
                child: Text(
                  'Tài khoản chưa có địa chỉ đã lưu. Vui lòng nhập thủ công.',
                  style: TextStyle(fontSize: 12, color: AppColors.textMuted),
                ),
              ),
            AppTextField(
              controller: _name,
              label: 'Họ và tên *',
              hint: 'Nguyễn Văn An',
              enabled: !state.useAccountAddress,
              errorText: state.fieldErrors['fullName'],
              onChanged: context.read<CheckoutCubit>().setName,
            ),
            const SizedBox(height: AppSpacing.sm),
            AppTextField(
              controller: _phone,
              label: 'Số điện thoại *',
              hint: '09xx xxx xxx',
              enabled: !state.useAccountAddress,
              keyboardType: TextInputType.phone,
              errorText: state.fieldErrors['phone'],
              onChanged: context.read<CheckoutCubit>().setPhone,
            ),
            const SizedBox(height: AppSpacing.sm),
            AppTextField(
              controller: _street,
              label: 'Số nhà, tên đường *',
              hint: '123 Lê Duẩn',
              enabled: !state.useAccountAddress,
              errorText: state.fieldErrors['streetAddress'],
              onChanged: context.read<CheckoutCubit>().setStreetAddress,
            ),
            const SizedBox(height: AppSpacing.sm),
            AppTextField(
              controller: _ward,
              label: 'Phường/Xã *',
              enabled: !state.useAccountAddress,
              errorText: state.fieldErrors['ward'],
              onChanged: context.read<CheckoutCubit>().setWard,
            ),
            const SizedBox(height: AppSpacing.sm),
            AppTextField(
              controller: _district,
              label: 'Quận/Huyện *',
              enabled: !state.useAccountAddress,
              errorText: state.fieldErrors['district'],
              onChanged: context.read<CheckoutCubit>().setDistrict,
            ),
            const SizedBox(height: AppSpacing.sm),
            AppTextField(
              controller: _province,
              label: 'Tỉnh/Thành phố *',
              enabled: !state.useAccountAddress,
              errorText: state.fieldErrors['province'],
              onChanged: context.read<CheckoutCubit>().setProvince,
            ),
            const SizedBox(height: AppSpacing.lg),
            Text(
              'Phương thức giao hàng',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            Text(
              'Lựa chọn và phí giao hàng được lấy từ API.',
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: AppSpacing.xs),
            const _ApiDrivenCard(
              title: 'Theo phương thức API đề xuất',
              subtitle: 'Thời gian và phí được xác nhận khi backend phản hồi',
            ),
            const SizedBox(height: AppSpacing.lg),
            Text(
              'Phương thức thanh toán',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: AppSpacing.xs),
            ...PaymentMethod.values.map(
              (method) => Card(
                color: state.paymentMethod == method
                    ? AppColors.softYellow
                    : Colors.white,
                child: InkWell(
                  onTap: submitting
                      ? null
                      : () => context.read<CheckoutCubit>().setPayment(method),
                  borderRadius: BorderRadius.circular(AppRadius.large),
                  child: SizedBox(
                    height: 56,
                    child: Row(
                      children: [
                        const SizedBox(width: AppSpacing.sm),
                        Icon(
                          state.paymentMethod == method
                              ? Icons.radio_button_checked
                              : Icons.radio_button_off,
                          color: state.paymentMethod == method
                              ? AppColors.honeyGold
                              : AppColors.textMuted,
                        ),
                        const SizedBox(width: AppSpacing.sm),
                        Text(method.label),
                      ],
                    ),
                  ),
                ),
              ),
            ),
            const SizedBox(height: AppSpacing.lg),
            Text(
              'Sản phẩm trong đơn',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: AppSpacing.xs),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Column(
                  children: [
                    for (final line in cart.items)
                      ListTile(
                        contentPadding: EdgeInsets.zero,
                        leading: const CircleAvatar(
                          backgroundColor: AppColors.muted,
                          child: Icon(Icons.redeem_outlined),
                        ),
                        title: Text(line.product.name),
                        subtitle: Text(
                          '${line.variant.name} • SL ${line.quantity}',
                        ),
                        trailing: Text(line.subtotal.format()),
                      ),
                  ],
                ),
              ),
            ),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const Text(
                      'Tổng thanh toán',
                      style: TextStyle(fontWeight: FontWeight.bold),
                    ),
                    Text(
                      '${cart.subtotal.format()} + phí',
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                  ],
                ),
              ),
            ),
            if (state.message != null &&
                state.status != CheckoutStatus.submitting)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(
                  state.message!,
                  style: const TextStyle(color: AppColors.brickRed),
                ),
              ),
          ],
        ),
        bottomSheet: SafeArea(
          top: false,
          child: Container(
            padding: const EdgeInsets.all(12),
            decoration: const BoxDecoration(
              color: AppColors.ivory,
              border: Border(top: BorderSide(color: AppColors.border)),
            ),
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    '${cart.subtotal.format()} + phí',
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                ),
                SizedBox(
                  width: 150,
                  child: AppPrimaryButton(
                    label: 'Đặt Hàng',
                    loading: submitting,
                    onPressed: cart.items.isEmpty
                        ? null
                        : context.read<CheckoutCubit>().submit,
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    },
  );
}

class _ApiDrivenCard extends StatelessWidget {
  const _ApiDrivenCard({required this.title, required this.subtitle});
  final String title;
  final String subtitle;
  @override
  Widget build(BuildContext context) => Card(
    color: const Color(0xFFFAF7EF),
    child: ListTile(
      leading: const Icon(
        Icons.radio_button_checked,
        color: AppColors.honeyGold,
      ),
      title: Text(title),
      subtitle: Text(subtitle),
      trailing: const Text('Theo API', style: TextStyle(fontSize: 11)),
    ),
  );
}
