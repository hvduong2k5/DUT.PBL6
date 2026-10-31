import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/features/checkout/presentation/mvp_quote_cubit.dart';
import 'package:mobile_app/features/checkout/presentation/mvp_checkout_cubit.dart';
import 'package:mobile_app/app/router/routes.dart';

class MvpQuotePage extends StatefulWidget {
  const MvpQuotePage({
    super.key,
    this.localPreview = false,
    this.now,
    this.checkoutEnabled = false,
  });
  final bool localPreview;
  final bool checkoutEnabled;
  final DateTime Function()? now;
  @override
  State<MvpQuotePage> createState() => _MvpQuotePageState();
}

class _MvpQuotePageState extends State<MvpQuotePage> {
  Timer? _timer;
  @override
  void initState() {
    super.initState();
    _timer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (mounted && context.read<MvpQuoteCubit>().state.quote != null) {
        setState(() {});
      }
    });
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Báo giá cả giỏ', showBack: true),
    body: SafeArea(
      child: BlocBuilder<MvpQuoteCubit, MvpQuoteState>(
        builder: (context, state) {
          final cubit = context.read<MvpQuoteCubit>();
          final checkout = widget.checkoutEnabled
              ? context.watch<MvpCheckoutCubit>().state
              : null;
          if (state.status == MvpQuoteStatus.initial ||
              state.status == MvpQuoteStatus.loading) {
            return const AppLoadingView();
          }
          if (state.status == MvpQuoteStatus.failure) {
            return AppErrorView(
              message: state.message ?? 'Chưa thể tải báo giá.',
              onRetry: cubit.load,
            );
          }
          if (state.status == MvpQuoteStatus.emptyCart) {
            return AppEmptyView(
              title: 'Giỏ hàng trống',
              message: 'Thêm sản phẩm để xem báo giá.',
              action: TextButton(
                onPressed: () => context.go('/catalog'),
                child: const Text('Khám phá sản phẩm'),
              ),
            );
          }
          if (state.status == MvpQuoteStatus.noAddress) {
            return AppEmptyView(
              title: 'Chưa có địa chỉ phù hợp',
              message:
                  'Chưa thể tiếp tục khi chưa có địa chỉ giao hàng hợp lệ. Giỏ hàng của bạn vẫn được giữ lại.',
              action: OutlinedButton(
                onPressed: cubit.load,
                child: const Text('Tải lại địa chỉ'),
              ),
            );
          }
          final quote = state.quote;
          final usable = state.usableAt((widget.now ?? DateTime.now)());
          final pending = state.status == MvpQuoteStatus.quoting;
          final items = quote?.items ?? state.cart!.items;
          final selected = state.addresses
              .where((a) => a.id == state.addressId)
              .first;
          final provinceCode =
              quote?.address.provinceCode ?? selected.provinceCode;
          final wardCode = quote?.address.wardCode ?? selected.wardCode;
          // Names describe a snapshot only when its canonical codes still match.
          final province = provinceCode == selected.provinceCode
              ? selected.provinceName ?? provinceCode
              : provinceCode;
          final ward =
              wardCode == selected.wardCode &&
                  provinceCode == selected.provinceCode
              ? selected.wardName ?? wardCode
              : wardCode;
          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              if (widget.localPreview)
                const Padding(
                  padding: EdgeInsets.only(bottom: 12),
                  child: Text(
                    'Dữ liệu thử nghiệm trên thiết bị. Không tạo đơn hoặc thanh toán thật.',
                  ),
                ),
              if (checkout?.unresolved == true ||
                  checkout?.storageBlocked == true)
                OutlinedButton(
                  onPressed: () => context.push(AppRoutes.checkoutOperation),
                  child: const Text('Tiếp tục yêu cầu trước đó'),
                ),
              if (checkout?.message != null) Text(checkout!.message!),
              Text(
                'Kiểm tra đơn trước khi mua',
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: 8),
              const Text(
                'Báo giá áp dụng cho toàn bộ giỏ. Giá và phí giao hàng được kiểm tra trước khi xác nhận.',
              ),
              const SizedBox(height: 20),
              Text(
                'Địa chỉ nhận hàng',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                key: ValueKey(state.addressId),
                initialValue: state.addressId,
                isExpanded: true,
                decoration: const InputDecoration(labelText: 'Địa chỉ đã lưu'),
                items: state.addresses
                    .map(
                      (a) => DropdownMenuItem(
                        value: a.id,
                        child: Text(
                          a.recipient,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                    )
                    .toList(),
                onChanged: pending
                    ? null
                    : (id) {
                        if (id != null) cubit.selectAddress(id);
                      },
              ),
              const SizedBox(height: 12),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(quote?.address.recipient ?? selected.recipient),
                      Text(quote?.address.phone ?? selected.phone),
                      Text(quote?.address.street ?? selected.street),
                      Text('$ward, $province'),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 16),
              Text(
                'Toàn bộ sản phẩm (${items.length} dòng)',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              for (final item in items)
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(item.name ?? item.sku),
                        Text('Số lượng: ${item.quantity}'),
                        Text(
                          item.unitPrice == null
                              ? 'Giá sẽ được xác nhận trong báo giá'
                              : 'Đơn giá: ${item.unitPrice!.format()}',
                        ),
                      ],
                    ),
                  ),
                ),
              if (quote != null) ...[
                const SizedBox(height: 16),
                _Amount('Tiền hàng', quote.subtotal.format()),
                _Amount('Phí giao hàng', quote.shippingFee.format()),
                _Amount('Tổng báo giá', quote.total.format()),
                const SizedBox(height: 12),
                Semantics(
                  liveRegion: true,
                  child: Text(
                    usable
                        ? 'Có hiệu lực đến ${DateFormat('HH:mm:ss dd/MM/yyyy').format(quote.validUntil.toLocal())}'
                        : 'Báo giá đã hết hạn. Vui lòng lấy báo giá mới và kiểm tra lại.',
                  ),
                ),
                const SizedBox(height: 12),
                if (!widget.checkoutEnabled)
                  const Text(
                    'Chưa thể đặt hàng trong phiên này. Báo giá chưa tạo đơn và chưa giữ hàng.',
                  ),
                if (widget.checkoutEnabled) ...[
                  const Text(
                    'Xác nhận để gửi yêu cầu tạo đơn. Chỉ thanh toán sau khi đơn được tạo thành công.',
                  ),
                  const SizedBox(height: 12),
                  AppPrimaryButton(
                    label: checkout?.attempt?.input.quoteId == quote.id
                        ? 'Xem kết quả yêu cầu'
                        : 'Xác nhận đặt hàng',
                    loading: checkout?.busy ?? false,
                    onPressed: checkout?.attempt?.input.quoteId == quote.id
                        ? () => context.push(AppRoutes.checkoutOperation)
                        : usable && checkout?.canStart == true
                        ? () async {
                            final controller = context.read<MvpCheckoutCubit>();
                            await controller.start(quote);
                            if (context.mounted &&
                                controller.state.attempt != null) {
                              context.push(AppRoutes.checkoutOperation);
                            }
                          }
                        : null,
                  ),
                ],
              ],
              const SizedBox(height: 20),
              AppPrimaryButton(
                label: quote == null ? 'Lấy báo giá cả giỏ' : 'Lấy báo giá mới',
                loading: pending,
                onPressed: quote == null
                    ? cubit.requestQuote
                    : cubit.refreshQuote,
              ),
              const SizedBox(height: 8),
              OutlinedButton(
                onPressed: pending ? null : () => context.go('/cart'),
                child: const Text('Về giỏ hàng'),
              ),
            ],
          );
        },
      ),
    ),
  );
}

class _Amount extends StatelessWidget {
  const _Amount(this.label, this.value);
  final String label, value;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 6),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(child: Text(label)),
        const SizedBox(width: 12),
        Flexible(
          child: Text(
            value,
            textAlign: TextAlign.end,
            style: Theme.of(context).textTheme.titleMedium,
          ),
        ),
      ],
    ),
  );
}
