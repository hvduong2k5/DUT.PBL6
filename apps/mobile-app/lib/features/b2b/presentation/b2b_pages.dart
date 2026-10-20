import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/features/b2b/domain/b2b_repository.dart';
import 'package:mobile_app/features/b2b/presentation/b2b_cubits.dart';
import 'package:mobile_app/features/auth/presentation/capabilities_cubit.dart';

class B2bRequestPage extends StatefulWidget {
  const B2bRequestPage({super.key});
  @override
  State<B2bRequestPage> createState() => _B2bRequestPageState();
}

class _B2bRequestPageState extends State<B2bRequestPage> {
  final _date = TextEditingController();
  final Map<String, TextEditingController> _quantities = {};
  final Map<String, bool> _branding = {
    'engraveLogo': true,
    'customSleeve': true,
    'greetingCard': true,
    'ribbon': false,
  };
  String _purpose = 'VIP_CUSTOMER_GIFT';
  String? _province;
  bool _invoice = true, _sample = true;
  @override
  void dispose() {
    _date.dispose();
    for (final controller in _quantities.values) {
      controller.dispose();
    }
    super.dispose();
  }

  Future<void> _pickDate() async {
    final now = DateTime.now(),
        minimum = DateTime(
          DateTime.now().year,
          DateTime.now().month,
          DateTime.now().day,
        ).add(const Duration(days: 7));
    final selected = await showDatePicker(
      context: context,
      initialDate: minimum,
      firstDate: minimum,
      lastDate: DateTime(now.year + 10),
    );
    if (mounted && selected != null) {
      setState(() => _date.text = DateFormat('yyyy-MM-dd').format(selected));
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(
      title: 'Quà biếu doanh nghiệp',
      showBack: true,
    ),
    body: SafeArea(
      child: BlocBuilder<B2bRequestCubit, B2bRequestState>(
        builder: (context, state) {
          final cubit = context.read<B2bRequestCubit>(), data = state.context;
          if (state.loading) return const AppLoadingView();
          if (data == null) {
            return AppErrorView(
              message: state.error ?? 'Chưa thể mở không gian B2B.',
              onRetry: cubit.load,
            );
          }
          if (data.organization.verification != 'VERIFIED') {
            return AppErrorView(
              message:
                  'Tổ chức chưa được xác minh. Vui lòng liên hệ hỗ trợ doanh nghiệp.',
              onRetry: cubit.load,
            );
          }
          for (var index = 0; index < data.products.length; index++) {
            final product = data.products[index];
            _quantities.putIfAbsent(
              product.sku,
              () => TextEditingController(
                text: index == 0 ? '${product.minimum}' : '0',
              ),
            );
          }
          if (!data.provinces.any((p) => p.code == _province)) {
            _province = data.provinces.firstOrNull?.code;
          }
          final quantities = {
            for (final product in data.products)
              product.sku: int.tryParse(_quantities[product.sku]!.text) ?? 0,
          };
          final totalQuantity = quantities.values.fold(0, (a, b) => a + b);
          final merchandise = data.products.fold(
            0,
            (a, p) => a + p.price * (quantities[p.sku] ?? 0),
          );
          final tiers =
              data.tiers
                  .where(
                    (t) =>
                        totalQuantity >= t.minimum &&
                        (t.maximum == null || totalQuantity <= t.maximum!),
                  )
                  .toList()
                ..sort((a, b) => a.minimum.compareTo(b.minimum));
          final tier = tiers.lastOrNull;
          final estimate =
              (merchandise * (100 - (tier?.percent ?? 0)) + 50) ~/ 100;
          return ListView(
            padding: const EdgeInsets.all(20),
            children: [
              Text(
                'Đặc quyền quà biếu Cố Đô',
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: 12),
              const Text(
                'Chọn thức quà, số lượng và dấu ấn doanh nghiệp. Giá chính thức được chốt trên báo giá đã phát hành.',
              ),
              _B2bPanel(
                title: 'Doanh nghiệp đã xác minh',
                children: [
                  Text(
                    data.organization.name,
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  Text('Mã số thuế: ${data.organization.taxCode}'),
                  Text(
                    '${data.organization.representative} · ${data.organization.title}',
                  ),
                  if (data.organization.email.isNotEmpty)
                    Text(data.organization.email),
                  if (data.organization.phone.isNotEmpty)
                    Text(data.organization.phone),
                  Text(data.organization.address),
                ],
              ),
              if (state.receipt != null)
                _B2bPanel(
                  title: 'Đã tiếp nhận ${state.receipt!.number}',
                  children: [
                    Text(state.receipt!.message),
                    const Text(
                      'Yêu cầu chưa phải báo giá hay xác nhận thanh toán.',
                    ),
                    TextButton(
                      onPressed: cubit.newRequest,
                      child: const Text('Tạo yêu cầu khác'),
                    ),
                  ],
                )
              else ...[
                _B2bPanel(
                  title: '1 · Mục đích & thời hạn',
                  children: [
                    DropdownButtonFormField<String>(
                      initialValue: _purpose,
                      isExpanded: true,
                      decoration: const InputDecoration(
                        labelText: 'Mục đích tặng quà',
                      ),
                      items: [
                        for (final entry in b2bPurposes.entries)
                          DropdownMenuItem(
                            value: entry.key,
                            child: Text(
                              entry.value,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                      ],
                      onChanged: state.pending
                          ? null
                          : (value) => setState(() => _purpose = value!),
                    ),
                    const SizedBox(height: 16),
                    AppTextField(
                      controller: _date,
                      label: 'Ngày nhận dự kiến',
                      hint: 'YYYY-MM-DD · từ 7 ngày tới',
                      keyboardType: TextInputType.datetime,
                      enabled: !state.pending,
                      suffixIcon: IconButton(
                        tooltip: 'Chọn ngày',
                        onPressed: state.pending ? null : _pickDate,
                        icon: const Icon(Icons.calendar_month_outlined),
                      ),
                    ),
                    const SizedBox(height: 16),
                    DropdownButtonFormField<String>(
                      initialValue: _province,
                      isExpanded: true,
                      decoration: const InputDecoration(
                        labelText: 'Tỉnh/thành nhận hàng',
                      ),
                      items: [
                        for (final p in data.provinces)
                          DropdownMenuItem(value: p.code, child: Text(p.name)),
                      ],
                      onChanged: state.pending
                          ? null
                          : (value) => setState(() => _province = value),
                    ),
                  ],
                ),
                _B2bPanel(
                  title: '2 · Thức quà & số lượng dự kiến',
                  children: [
                    if (data.products.isEmpty)
                      const Text(
                        'Chưa có thức quà khả dụng. Vui lòng thử lại sau.',
                      ),
                    for (final product in data.products)
                      Padding(
                        padding: const EdgeInsets.only(bottom: 24),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            Text(
                              product.name,
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                            Text(product.variant),
                            Text('${Money.vnd(product.price).format()}/bộ'),
                            const SizedBox(height: 12),
                            AppTextField(
                              controller: _quantities[product.sku],
                              label: 'Số lượng · tối thiểu ${product.minimum}',
                              keyboardType: TextInputType.number,
                              inputFormatters: [
                                FilteringTextInputFormatter.digitsOnly,
                              ],
                              enabled: !state.pending,
                              onChanged: (_) => setState(() {}),
                            ),
                          ],
                        ),
                      ),
                    Text('Tổng dự kiến: $totalQuantity bộ'),
                    Text(
                      'Ưu đãi tham khảo: ${tier == null ? 'Chờ đạt số lượng' : '${tier.percent}% · ${tier.label}'}',
                    ),
                    Text(
                      'Tạm tính: ${Money.vnd(estimate).format()}',
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const Text(
                      'Thuế, vận chuyển và tùy biến sẽ được chốt trong báo giá chính thức.',
                    ),
                  ],
                ),
                _B2bPanel(
                  title: '3 · Dấu ấn doanh nghiệp',
                  children: [
                    for (final option in b2bBranding.entries)
                      CheckboxListTile(
                        contentPadding: EdgeInsets.zero,
                        controlAffinity: ListTileControlAffinity.leading,
                        title: Text(option.value),
                        value: _branding[option.key],
                        onChanged: state.pending
                            ? null
                            : (value) => setState(
                                () => _branding[option.key] = value!,
                              ),
                      ),
                    const Text(
                      'Chưa hỗ trợ tải logo/tài liệu. Trao đổi yêu cầu riêng qua Ban Tri Kỷ.',
                    ),
                  ],
                ),
                _B2bPanel(
                  title: '4 · Xác nhận nhu cầu',
                  children: [
                    CheckboxListTile(
                      contentPadding: EdgeInsets.zero,
                      controlAffinity: ListTileControlAffinity.leading,
                      title: const Text(
                        'Yêu cầu hóa đơn theo hồ sơ doanh nghiệp',
                      ),
                      value: _invoice,
                      onChanged: state.pending
                          ? null
                          : (v) => setState(() => _invoice = v!),
                    ),
                    CheckboxListTile(
                      contentPadding: EdgeInsets.zero,
                      controlAffinity: ListTileControlAffinity.leading,
                      title: const Text('Đề nghị tư vấn hộp mẫu'),
                      value: _sample,
                      onChanged: state.pending
                          ? null
                          : (v) => setState(() => _sample = v!),
                    ),
                  ],
                ),
                if (state.error != null) _B2bError(state.error!),
                AppPrimaryButton(
                  label: 'Gửi yêu cầu báo giá B2B',
                  loading: state.pending,
                  onPressed: data.products.isEmpty || data.provinces.isEmpty
                      ? null
                      : () {
                          FocusScope.of(context).unfocus();
                          final province = data.provinces.firstWhere(
                            (p) => p.code == _province,
                          );
                          cubit.submit(
                            B2bRequestDraft(
                              organizationId: data.organization.id,
                              purpose: _purpose,
                              date: _date.text.trim(),
                              provinceCode: province.code,
                              provinceName: province.name,
                              quantities: quantities,
                              branding: Map.of(_branding),
                              invoice: _invoice,
                              sample: _sample,
                            ),
                          );
                        },
                ),
              ],
              _B2bPanel(
                title: 'Yêu cầu gần đây',
                children: [
                  if (data.requests.isEmpty)
                    const Text('Chưa có yêu cầu nào thuộc doanh nghiệp.'),
                  for (final request in data.requests)
                    Padding(
                      padding: const EdgeInsets.symmetric(vertical: 10),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            request.number,
                            style: Theme.of(context).textTheme.titleMedium,
                          ),
                          Text('${request.purpose} · ${request.quantity} bộ'),
                          Text(request.status),
                          if (request.quoteId != null)
                            TextButton(
                              onPressed: state.pending
                                  ? null
                                  : () => context.push(
                                      '/b2b/quotes/${Uri.encodeComponent(request.quoteId!)}',
                                    ),
                              child: const Text('Xem báo giá'),
                            ),
                        ],
                      ),
                    ),
                  TextButton(
                    onPressed: state.pending ? null : cubit.load,
                    child: const Text('Cập nhật hồ sơ doanh nghiệp'),
                  ),
                ],
              ),
              TextButton(
                onPressed: () => context.push(AppRoutes.supportRequest),
                child: const Text('Trao đổi với Ban Tri Kỷ'),
              ),
            ],
          );
        },
      ),
    ),
  );
}

class B2bQuotePage extends StatelessWidget {
  const B2bQuotePage({super.key, required this.quoteId});
  final String quoteId;
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(
      title: 'Báo giá doanh nghiệp',
      showBack: true,
    ),
    body: SafeArea(
      child: BlocBuilder<B2bQuoteCubit, B2bQuoteState>(
        builder: (context, state) {
          final cubit = context.read<B2bQuoteCubit>(), quote = state.quote;
          if (state.loading) return const AppLoadingView();
          if (quote == null) {
            return AppErrorView(
              message: state.error ?? 'Không thể mở báo giá.',
              onRetry: () => cubit.load(quoteId),
            );
          }
          final capabilities = context.watch<CapabilitiesCubit>().state;
          final canAct =
              capabilities.projection?.actor == 'B2B' &&
              capabilities.allows('B2B_QUOTE_ACCEPT') &&
              (state.retryingAcceptance || quote.canAccept && !quote.expired) &&
              state.accepted == null;
          return ListView(
            padding: const EdgeInsets.all(20),
            children: [
              SelectableText(quote.number),
              Text(
                'Bảng báo giá quà biếu',
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              Text(
                '${quote.organization.name}\nPhiên bản ${quote.version} · ${quote.label}',
              ),
              Text(
                'Phát hành: ${DateFormat('dd/MM/yyyy').format(quote.issued.toLocal())}',
              ),
              Text(
                'Hiệu lực đến: ${DateFormat('dd/MM/yyyy HH:mm').format(quote.expires.toLocal())}',
              ),
              if (quote.expired && !state.retryingAcceptance)
                const _B2bError(
                  'Báo giá đã hết hạn. Vui lòng đề nghị báo giá mới.',
                ),
              if (state.accepted != null)
                _B2bPanel(
                  title: 'Đã ghi nhận chấp thuận',
                  children: [
                    Text(state.accepted!.message),
                    Text('Đơn tham chiếu: ${state.accepted!.orderNumber}'),
                    const Text(
                      'Chấp thuận không đồng nghĩa đã đặt cọc hoặc thanh toán.',
                    ),
                    TextButton(
                      onPressed: () => context.go(
                        '/orders/${Uri.encodeComponent(state.accepted!.orderId)}',
                      ),
                      child: const Text('Theo dõi đơn tham chiếu'),
                    ),
                  ],
                ),
              _B2bPanel(
                title: 'Thức quà & tùy biến',
                children: [
                  for (final item in quote.items)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 16),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            item.name,
                            style: Theme.of(context).textTheme.titleMedium,
                          ),
                          Text(item.variant),
                          Text(
                            '${item.quantity} bộ × ${Money.vnd(item.price).format()}',
                          ),
                          Text(Money.vnd(item.total).format()),
                        ],
                      ),
                    ),
                  for (final option in quote.customizations)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 8),
                      child: Text('• $option'),
                    ),
                ],
              ),
              _B2bPanel(
                title: 'Điều khoản phiên bản ${quote.version}',
                children: [
                  for (final term in quote.terms)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 16),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            term.title,
                            style: const TextStyle(fontWeight: FontWeight.w600),
                          ),
                          SelectableText(term.description),
                        ],
                      ),
                    ),
                ],
              ),
              _B2bPanel(
                title: 'Quyết toán tài chính',
                children: [
                  for (final entry in const {
                    'merchandiseVnd': 'Giá trị gốc',
                    'discountVnd': 'Chiết khấu',
                    'customizationVnd': 'Tùy biến',
                    'vatVnd': 'VAT',
                    'grandTotalVnd': 'Tổng giá trị',
                    'depositVnd': 'Khoản đặt cọc',
                    'remainingVnd': 'Còn lại',
                  }.entries)
                    Padding(
                      padding: const EdgeInsets.symmetric(vertical: 6),
                      child: Row(
                        children: [
                          Expanded(child: Text(entry.value)),
                          const SizedBox(width: 12),
                          Flexible(
                            child: Text(
                              Money.vnd(quote.totals[entry.key] ?? 0).format(),
                              textAlign: TextAlign.end,
                            ),
                          ),
                        ],
                      ),
                    ),
                  Text('Tỷ lệ đặt cọc: ${quote.totals['depositPercent']}%'),
                ],
              ),
              if (state.accepted == null)
                _B2bPanel(
                  title: 'Chấp thuận báo giá',
                  children: [
                    const Text(
                      'Chỉ đại diện có thẩm quyền được chấp thuận. Thao tác này ghi nhận báo giá; thanh toán là bước riêng.',
                    ),
                    CheckboxListTile(
                      contentPadding: EdgeInsets.zero,
                      controlAffinity: ListTileControlAffinity.leading,
                      title: Text(
                        'Tôi đã đọc và đồng ý toàn bộ báo giá, điều khoản phiên bản ${quote.version}.',
                      ),
                      value: state.confirmed,
                      onChanged: state.pending || !canAct
                          ? null
                          : (value) => cubit.agree(value!),
                    ),
                    if (state.error != null) _B2bError(state.error!),
                    AppPrimaryButton(
                      label: state.retryingAcceptance
                          ? 'Kiểm tra lại chấp thuận'
                          : 'Chấp thuận báo giá',
                      loading: state.pending,
                      onPressed: canAct && state.confirmed
                          ? cubit.accept
                          : null,
                    ),
                    if (!state.retryingAcceptance)
                      TextButton(
                        onPressed: state.pending
                            ? null
                            : () => cubit.load(quote.id),
                        child: const Text('Cập nhật báo giá'),
                      ),
                  ],
                ),
              _B2bPanel(
                title: 'Thông tin thanh toán công khai',
                children: [
                  for (final key in const [
                    'bankName',
                    'accountName',
                    'accountNumberMasked',
                    'transferContent',
                  ])
                    SelectableText(quote.payment[key] ?? ''),
                ],
              ),
              _B2bPanel(
                title: 'Chuyên viên phụ trách',
                children: [
                  for (final key in const [
                    'displayName',
                    'title',
                    'phone',
                    'email',
                  ])
                    if (quote.contact[key]?.isNotEmpty ?? false)
                      Text(quote.contact[key]!),
                ],
              ),
              TextButton(
                onPressed: state.pending
                    ? null
                    : () => context.go(AppRoutes.b2b),
                child: const Text('Về không gian doanh nghiệp'),
              ),
            ],
          );
        },
      ),
    ),
  );
}

class _B2bPanel extends StatelessWidget {
  const _B2bPanel({required this.title, required this.children});
  final String title;
  final List<Widget> children;
  @override
  Widget build(BuildContext context) => Card(
    margin: const EdgeInsets.symmetric(vertical: 12),
    child: Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 16),
          ...children,
        ],
      ),
    ),
  );
}

class _B2bError extends StatelessWidget {
  const _B2bError(this.message);
  final String message;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 12),
    child: Semantics(
      liveRegion: true,
      child: Text(
        message,
        style: TextStyle(color: Theme.of(context).colorScheme.error),
      ),
    ),
  );
}
