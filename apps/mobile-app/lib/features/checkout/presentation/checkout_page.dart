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
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/features/auth/presentation/auth_return_location.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';

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
        context.read<CartCubit>().consume(
          context
              .read<CheckoutCubit>()
              .cart
              .items
              .map((line) => line.itemId)
              .toSet(),
        );
        context.go(
          '/order-confirmation/${state.confirmation!.orderId}',
          extra: state.confirmation,
        );
      }
    },
    builder: (context, state) {
      final cart = context.read<CheckoutCubit>().cart;
      final authenticated =
          context.watch<AuthCubit>().state.status == AuthStatus.authenticated;
      final submitting = state.status == CheckoutStatus.submitting;
      final province = state.locations
          .where((item) => item.name == state.province)
          .firstOrNull;
      if (cart.items.isEmpty) {
        return Scaffold(
          appBar: const AppMobileHeader(
            title: 'Thông tin đặt hàng',
            showBack: true,
          ),
          body: AppEmptyView(
            title: 'Chưa chọn sản phẩm',
            message:
                'Hãy chọn các dòng khả dụng trong giỏ trước khi thanh toán.',
            action: AppPrimaryButton(
              label: 'Về giỏ hàng',
              onPressed: () => context.go(AppRoutes.cart),
            ),
          ),
        );
      }
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
            if (!authenticated)
              TextButton(
                onPressed: submitting
                    ? null
                    : () => context.push(
                        authLocation(
                          AppRoutes.login,
                          GoRouterState.of(context).uri.toString(),
                        ),
                      ),
                child: const Text('Đăng nhập để dùng địa chỉ đã lưu'),
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
              enabled: !state.useAccountAddress && !submitting,
              errorText: state.fieldErrors['fullName'],
              onChanged: context.read<CheckoutCubit>().setName,
            ),
            const SizedBox(height: AppSpacing.sm),
            AppTextField(
              controller: _phone,
              label: 'Số điện thoại *',
              hint: '09xx xxx xxx',
              enabled: !state.useAccountAddress && !submitting,
              keyboardType: TextInputType.phone,
              errorText: state.fieldErrors['phone'],
              onChanged: context.read<CheckoutCubit>().setPhone,
            ),
            const SizedBox(height: AppSpacing.sm),
            AppTextField(
              controller: _street,
              label: 'Số nhà, tên đường *',
              hint: '123 Lê Duẩn',
              enabled: !state.useAccountAddress && !submitting,
              errorText: state.fieldErrors['streetAddress'],
              onChanged: context.read<CheckoutCubit>().setStreetAddress,
            ),
            const SizedBox(height: AppSpacing.sm),
            if (state.loadingLocations) const LinearProgressIndicator(),
            if (state.locationsError != null) ...[
              Text(
                state.locationsError!,
                style: const TextStyle(color: AppColors.darkRed),
              ),
              TextButton(
                onPressed: context.read<CheckoutCubit>().loadLocations,
                child: const Text('Tải lại khu vực giao hàng'),
              ),
            ],
            DropdownButtonFormField<String>(
              key: ValueKey(
                'province-${state.province}-${state.loadingLocations}',
              ),
              initialValue: province?.name,
              isExpanded: true,
              decoration: InputDecoration(
                labelText: 'Tỉnh/Thành phố *',
                errorText: state.fieldErrors['province'],
              ),
              items: state.locations
                  .map(
                    (item) => DropdownMenuItem(
                      value: item.name,
                      child: Text(item.name),
                    ),
                  )
                  .toList(),
              onChanged: submitting
                  ? null
                  : (value) =>
                        context.read<CheckoutCubit>().setProvince(value ?? ''),
            ),
            const SizedBox(height: AppSpacing.sm),
            DropdownButtonFormField<String>(
              key: ValueKey('ward-${state.province}-${state.ward}'),
              initialValue:
                  province?.wards.any((item) => item.name == state.ward) == true
                  ? state.ward
                  : null,
              isExpanded: true,
              decoration: InputDecoration(
                labelText: 'Phường/Xã *',
                errorText: state.fieldErrors['ward'],
              ),
              items: (province?.wards ?? [])
                  .map(
                    (item) => DropdownMenuItem(
                      value: item.name,
                      child: Text(item.name),
                    ),
                  )
                  .toList(),
              onChanged: submitting || province == null
                  ? null
                  : (value) =>
                        context.read<CheckoutCubit>().setWard(value ?? ''),
            ),
            if (state.fieldErrors['locations'] != null)
              Text(
                state.fieldErrors['locations']!,
                style: const TextStyle(color: AppColors.darkRed),
              ),
            const SizedBox(height: AppSpacing.lg),
            Text(
              'Phương thức giao hàng',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            Text(
              'Phí giao hàng sẽ được xác nhận khi đơn hàng được tiếp nhận.',
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: AppSpacing.xs),
            const _ApiDrivenCard(
              title: 'Giao hàng theo địa chỉ nhận',
              subtitle: 'Phí và thời gian giao được xác nhận khi đặt hàng.',
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
            Card(
              child: ListTile(
                leading: const Icon(Icons.local_offer_outlined),
                title: const Text('Kiểm tra mã ưu đãi'),
                subtitle: const Text('Việc áp mã vào đơn hiện chưa khả dụng.'),
                trailing: const Icon(Icons.chevron_right),
                onTap: submitting
                    ? null
                    : () => context.push(
                        Uri(
                          path: AppRoutes.promotions,
                          queryParameters: {
                            'subtotal': cart.subtotal.units.toString(),
                          },
                        ).toString(),
                      ),
              ),
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
                      'Tạm tính hàng',
                      style: TextStyle(fontWeight: FontWeight.bold),
                    ),
                    Text(
                      cart.subtotal.format(),
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
                    cart.subtotal.format(),
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
    ),
  );
}
