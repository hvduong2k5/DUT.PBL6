import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/config/app_config.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/features/returns/domain/returns_repository.dart';
import 'package:mobile_app/features/returns/presentation/returns_cubit.dart';

class ReturnRequestPage extends StatefulWidget {
  const ReturnRequestPage({super.key, required this.orderId});
  final String orderId;
  @override
  State<ReturnRequestPage> createState() => _ReturnRequestPageState();
}

class _ReturnRequestPageState extends State<ReturnRequestPage> {
  final _details = TextEditingController(),
      _urls = TextEditingController(),
      _bank = TextEditingController(),
      _account = TextEditingController(),
      _holder = TextEditingController();
  String? _line, _reason;
  int _quantity = 1;
  @override
  void dispose() {
    for (final controller in [_details, _urls, _bank, _account, _holder]) {
      controller.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Đổi trả & hậu mãi', showBack: true),
    body: SafeArea(
      child: BlocConsumer<ReturnsCubit, ReturnsState>(
        listener: (context, state) {
          if (state.created != null) {
            context.go(
              '/after-sales/${Uri.encodeComponent(state.created!.id)}',
            );
          }
        },
        builder: (context, state) {
          final cubit = context.read<ReturnsCubit>(), data = state.context;
          if (state.loading) return const AppLoadingView();
          if (data == null) {
            return AppErrorView(
              message: state.error ?? 'Chưa thể mở yêu cầu.',
              onRetry: () => cubit.load(widget.orderId),
            );
          }
          final selected = data.lines.where((l) => l.id == _line).firstOrNull;
          return ListView(
            padding: const EdgeInsets.all(20),
            children: [
              Text(
                'Yêu cầu đổi trả & hoàn tiền',
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: 12),
              SelectableText('Đơn #${data.orderId}'),
              const SizedBox(height: 12),
              const Text(
                'Mỗi yêu cầu dành cho một sản phẩm. Điều kiện và khoản hoàn được xác nhận sau khi gửi.',
              ),
              if (AppConfig.dataSourceMode == DataSourceMode.mock)
                const Padding(
                  padding: EdgeInsets.symmetric(vertical: 12),
                  child: Text(
                    'Thử nghiệm local: không gửi hồ sơ hoặc chuyển tiền thật.',
                    style: TextStyle(color: AppColors.textMuted),
                  ),
                ),
              if (data.blockingMessage != null || data.lines.isEmpty) ...[
                _ReturnSection(
                  title: 'Cần hỗ trợ kiểm tra đơn',
                  children: [
                    Text(
                      data.blockingMessage ??
                          'Chưa có sản phẩm khả dụng để gửi yêu cầu.',
                    ),
                  ],
                ),
                AppPrimaryButton(
                  label: 'Gửi yêu cầu hỗ trợ',
                  onPressed: () => context.push(AppRoutes.supportRequest),
                ),
              ] else ...[
                _ReturnSection(
                  title: '1 · Chọn một sản phẩm',
                  children: [
                    DropdownButtonFormField<String>(
                      initialValue: selected?.id,
                      isExpanded: true,
                      decoration: const InputDecoration(
                        labelText: 'Sản phẩm cần hỗ trợ',
                      ),
                      items: [
                        for (final line in data.lines)
                          DropdownMenuItem(
                            value: line.id,
                            child: Text(
                              '${line.name} · ${line.variant}',
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                      ],
                      onChanged: state.pending
                          ? null
                          : (value) => setState(() {
                              _line = value;
                              _quantity = 1;
                            }),
                    ),
                    if (selected != null) ...[
                      const SizedBox(height: 12),
                      Text('Số lượng có thể yêu cầu: ${selected.maximum}'),
                      Row(
                        children: [
                          IconButton(
                            tooltip: 'Giảm số lượng',
                            onPressed: state.pending || _quantity <= 1
                                ? null
                                : () => setState(() => _quantity--),
                            icon: const Icon(Icons.remove),
                          ),
                          Text('$_quantity'),
                          IconButton(
                            tooltip: 'Tăng số lượng',
                            onPressed:
                                state.pending || _quantity >= selected.maximum
                                ? null
                                : () => setState(() => _quantity++),
                            icon: const Icon(Icons.add),
                          ),
                        ],
                      ),
                      Text(
                        'Giá trị tham chiếu: ${selected.price.times(_quantity).format()}',
                      ),
                      const Text(
                        'Giá trị tham chiếu không phải khoản hoàn đã duyệt.',
                      ),
                    ],
                  ],
                ),
                _ReturnSection(
                  title: '2 · Lý do & tình trạng',
                  children: [
                    DropdownButtonFormField<String>(
                      initialValue: _reason,
                      isExpanded: true,
                      decoration: const InputDecoration(
                        labelText: 'Lý do yêu cầu',
                      ),
                      items: [
                        for (final entry in returnReasons.entries)
                          DropdownMenuItem(
                            value: entry.key,
                            child: Text(entry.value),
                          ),
                      ],
                      onChanged: state.pending
                          ? null
                          : (value) => setState(() => _reason = value),
                    ),
                    const SizedBox(height: 16),
                    AppTextField(
                      controller: _details,
                      label: 'Mô tả tình trạng',
                      maxLines: 5,
                      maxLength: 1000,
                      enabled: !state.pending,
                    ),
                  ],
                ),
                _ReturnSection(
                  title: '3 · Bằng chứng',
                  children: [
                    const Text(
                      'Nhập 1–5 liên kết HTTPS của ảnh hoặc video đã được tải lên, mỗi dòng một liên kết.',
                    ),
                    const SizedBox(height: 12),
                    AppTextField(
                      controller: _urls,
                      label: 'URL bằng chứng',
                      maxLines: 4,
                      enabled: !state.pending,
                    ),
                    const Text('Ứng dụng hiện chưa có thao tác tải tệp.'),
                  ],
                ),
                _ReturnSection(
                  title: '4 · Tài khoản nhận hoàn tiền',
                  children: [
                    const Text(
                      'Nhập thông tin không có nghĩa yêu cầu đã được duyệt hoặc tiền đã được chuyển.',
                    ),
                    const SizedBox(height: 16),
                    AppTextField(
                      controller: _bank,
                      label: 'Mã ngân hàng',
                      maxLength: 30,
                      enabled: !state.pending,
                    ),
                    const SizedBox(height: 16),
                    AppTextField(
                      controller: _account,
                      label: 'Số tài khoản',
                      maxLength: 30,
                      keyboardType: TextInputType.number,
                      inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                      enabled: !state.pending,
                    ),
                    const SizedBox(height: 16),
                    AppTextField(
                      controller: _holder,
                      label: 'Tên chủ tài khoản',
                      maxLength: 100,
                      enabled: !state.pending,
                    ),
                  ],
                ),
                if (data.address.isNotEmpty)
                  _ReturnSection(
                    title: 'Thông tin giao hàng trong đơn',
                    children: [Text(data.address)],
                  ),
                if (state.error != null)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 12),
                    child: Semantics(
                      liveRegion: true,
                      child: Text(
                        state.error!,
                        style: const TextStyle(color: AppColors.darkRed),
                      ),
                    ),
                  ),
                AppPrimaryButton(
                  label: 'Gửi yêu cầu đổi trả',
                  loading: state.pending,
                  onPressed: () {
                    FocusScope.of(context).unfocus();
                    cubit.submit(
                      ReturnDraft(
                        orderId: data.orderId,
                        lineId: _line ?? '',
                        quantity: _quantity,
                        reason: _reason ?? '',
                        details: _details.text.trim(),
                        evidenceUrls: _urls.text
                            .split(RegExp(r'\r?\n'))
                            .map((s) => s.trim())
                            .where((s) => s.isNotEmpty)
                            .toList(),
                        bank: _bank.text.trim().toUpperCase(),
                        account: _account.text.trim(),
                        holder: _holder.text.trim().toUpperCase(),
                      ),
                    );
                  },
                ),
              ],
              TextButton(
                onPressed: state.pending
                    ? null
                    : () => context.go(
                        '/orders/${Uri.encodeComponent(data.orderId)}',
                      ),
                child: const Text('Về chi tiết đơn hàng'),
              ),
            ],
          );
        },
      ),
    ),
  );
}

class ReturnLookupPage extends StatefulWidget {
  const ReturnLookupPage({super.key});
  @override
  State<ReturnLookupPage> createState() => _ReturnLookupPageState();
}

class _ReturnLookupPageState extends State<ReturnLookupPage> {
  final _id = TextEditingController();
  String? _error;
  @override
  void dispose() {
    _id.dispose();
    super.dispose();
  }

  void _open() {
    if (_id.text.trim().isEmpty) {
      setState(() => _error = 'Nhập mã hồ sơ đã được cấp.');
      return;
    }
    context.push('/after-sales/${Uri.encodeComponent(_id.text.trim())}');
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Hồ sơ hậu mãi', showBack: true),
    body: ListView(
      padding: const EdgeInsets.all(24),
      children: [
        Text(
          'Theo dõi yêu cầu đổi trả',
          style: Theme.of(context).textTheme.headlineMedium,
        ),
        const SizedBox(height: 16),
        const Text(
          'Nhập mã hồ sơ đã được tiếp nhận. Để tạo yêu cầu mới, mở một đơn đã giao trong lịch sử đơn hàng.',
        ),
        const SizedBox(height: 24),
        AppTextField(
          controller: _id,
          label: 'Mã hồ sơ hậu mãi',
          errorText: _error,
          textInputAction: TextInputAction.search,
          onSubmitted: (_) => _open(),
        ),
        const SizedBox(height: 20),
        AppPrimaryButton(label: 'Tra cứu hồ sơ', onPressed: _open),
        TextButton(
          onPressed: () => context.go(AppRoutes.orders),
          child: const Text('Đơn hàng của tôi'),
        ),
      ],
    ),
  );
}

class ReturnCasePage extends StatefulWidget {
  const ReturnCasePage({super.key, required this.caseId});
  final String caseId;
  @override
  State<ReturnCasePage> createState() => _ReturnCasePageState();
}

class _ReturnCasePageState extends State<ReturnCasePage> {
  late Future<ReturnCase> _future;
  @override
  void initState() {
    super.initState();
    _load();
  }

  void _load() =>
      _future = context.read<ReturnsRepository>().detail(widget.caseId);
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Chi tiết hậu mãi', showBack: true),
    body: FutureBuilder<ReturnCase>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done) {
          return const AppLoadingView();
        }
        if (!snapshot.hasData) {
          return AppErrorView(
            message: snapshot.error is ApiException
                ? (snapshot.error as ApiException).failure.message
                : 'Chưa thể tải hồ sơ hậu mãi.',
            onRetry: () => setState(_load),
          );
        }
        final data = snapshot.data!;
        return ListView(
          padding: const EdgeInsets.all(20),
          children: [
            Text(
              'Hồ sơ đổi trả',
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: 12),
            SelectableText(data.id),
            _ReturnSection(
              title: data.statusLabel,
              children: [
                Text(
                  'Tạo lúc ${DateFormat('dd/MM/yyyy HH:mm').format(data.createdAt.toLocal())}',
                ),
              ],
            ),
            _ReturnSection(
              title: 'Lý do ghi nhận',
              children: [SelectableText(data.reason)],
            ),
            _ReturnSection(
              title: 'Khoản hoàn trong hồ sơ',
              children: [
                Text(
                  data.refund.format(),
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                Text(
                  data.status == 'REFUNDED'
                      ? 'Hồ sơ đã ghi nhận hoàn tiền.'
                      : 'Chưa có xác nhận hoàn tiền thành công. Phê duyệt yêu cầu không đồng nghĩa đã chuyển tiền.',
                ),
              ],
            ),
            OutlinedButton(
              onPressed: () => setState(_load),
              child: const Text('Cập nhật trạng thái'),
            ),
            TextButton(
              onPressed: () => context.push(AppRoutes.supportRequest),
              child: const Text('Liên hệ hỗ trợ / bổ sung thông tin'),
            ),
            TextButton(
              onPressed: () =>
                  context.go('/orders/${Uri.encodeComponent(data.orderId)}'),
              child: const Text('Về đơn hàng'),
            ),
          ],
        );
      },
    ),
  );
}

class _ReturnSection extends StatelessWidget {
  const _ReturnSection({required this.title, required this.children});
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
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 16),
          ...children,
        ],
      ),
    ),
  );
}
