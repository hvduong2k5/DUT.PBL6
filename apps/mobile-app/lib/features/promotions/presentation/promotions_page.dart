import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:intl/intl.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/features/promotions/presentation/promotions_cubit.dart';

class PromotionsPage extends StatefulWidget {
  const PromotionsPage({super.key, this.checkoutSubtotal});
  final int? checkoutSubtotal;
  @override
  State<PromotionsPage> createState() => _PromotionsPageState();
}

class _PromotionsPageState extends State<PromotionsPage> {
  final _code = TextEditingController();
  late final _amount = TextEditingController(
    text: widget.checkoutSubtotal?.toString() ?? '',
  );
  final _checker = GlobalKey();
  @override
  void dispose() {
    _code.dispose();
    _amount.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Ưu đãi Tri Kỷ', showBack: true),
    body: BlocBuilder<PromotionsCubit, PromotionsState>(
      builder: (context, state) {
        final cubit = context.read<PromotionsCubit>();
        return ListView(
          padding: const EdgeInsets.all(20),
          children: [
            Text(
              'Ưu đãi dành cho những cuộc sum vầy',
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: 8),
            const Text(
              'Khám phá mã đang phát hành và kiểm tra điều kiện trước khi đặt hàng.',
            ),
            const SizedBox(height: 24),
            Text('Kho voucher', style: Theme.of(context).textTheme.titleLarge),
            const SizedBox(height: 12),
            if (state.loading) const LinearProgressIndicator(),
            if (state.error != null) ...[
              Text(state.error!),
              TextButton(
                onPressed: cubit.load,
                child: const Text('Thử tải lại'),
              ),
            ],
            if (!state.loading && state.error == null && state.vouchers.isEmpty)
              const Padding(
                padding: EdgeInsets.all(16),
                child: Text(
                  'Chưa có ưu đãi đang phát hành. Hãy quay lại vào dịp khác.',
                ),
              ),
            for (final voucher in state.vouchers)
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        voucher.label,
                        style: Theme.of(context).textTheme.titleLarge,
                      ),
                      const SizedBox(height: 8),
                      Text(voucher.title),
                      const SizedBox(height: 8),
                      SelectableText(
                        voucher.code,
                        style: const TextStyle(
                          color: AppColors.darkRed,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        'Đơn tối thiểu: ${Money.vnd(voucher.minimum).format()}',
                      ),
                      Text(
                        'Giảm tối đa: ${Money.vnd(voucher.maximum).format()}',
                      ),
                      Text(
                        'Hết hạn: ${DateFormat('dd/MM/yyyy').format(voucher.expiresAt.toLocal())}',
                      ),
                      TextButton(
                        onPressed: voucher.expiresAt.isAfter(DateTime.now())
                            ? () {
                                _code.text = voucher.code;
                                cubit.setCode(voucher.code);
                                final checkerContext = _checker.currentContext;
                                if (checkerContext != null) {
                                  Scrollable.ensureVisible(
                                    checkerContext,
                                    duration: const Duration(milliseconds: 200),
                                  );
                                }
                              }
                            : null,
                        child: Text(
                          voucher.expiresAt.isAfter(DateTime.now())
                              ? 'Kiểm tra mã này'
                              : 'Đã hết hạn',
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            const SizedBox(height: 24),
            Text(
              'Kiểm tra điều kiện mã',
              key: _checker,
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 12),
            AppTextField(
              controller: _code,
              label: 'Mã ưu đãi',
              onChanged: cubit.setCode,
            ),
            const SizedBox(height: 12),
            AppTextField(
              controller: _amount,
              label: widget.checkoutSubtotal == null
                  ? 'Giá trị hàng (đ)'
                  : 'Giá trị hàng đã chọn (đ)',
              keyboardType: TextInputType.number,
              enabled: widget.checkoutSubtotal == null,
              onChanged: cubit.setSubtotal,
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => cubit.validate(),
            ),
            const SizedBox(height: 16),
            AppPrimaryButton(
              label: 'Kiểm tra mã',
              loading: state.validating,
              onPressed: cubit.validate,
            ),
            if (state.validationError != null)
              Padding(
                padding: const EdgeInsets.only(top: 12),
                child: Semantics(
                  liveRegion: true,
                  child: Text(
                    state.validationError!,
                    style: const TextStyle(color: AppColors.darkRed),
                  ),
                ),
              ),
            if (state.result?.valid == true)
              Padding(
                padding: const EdgeInsets.only(top: 12),
                child: Semantics(
                  liveRegion: true,
                  child: Text(
                    '${state.result!.message}\nGiá trị ưu đãi dự kiến: ${state.result!.discount.format()}',
                    style: const TextStyle(color: AppColors.success),
                  ),
                ),
              ),
            const SizedBox(height: 16),
            const Text(
              'Kiểm tra mã chưa áp dụng ưu đãi vào đơn. Tính năng áp mã khi đặt hàng hiện chưa khả dụng.',
              style: TextStyle(fontSize: 12, color: AppColors.textMuted),
            ),
          ],
        );
      },
    ),
  );
}
