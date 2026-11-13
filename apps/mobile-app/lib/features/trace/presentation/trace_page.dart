import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/config/app_config.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/core/widgets/public_https_button.dart';
import 'package:mobile_app/features/trace/presentation/trace_cubit.dart';

class TracePage extends StatefulWidget {
  const TracePage({super.key, this.initialCode = ''});
  final String initialCode;
  @override
  State<TracePage> createState() => _TracePageState();
}

class _TracePageState extends State<TracePage> {
  late final _code = TextEditingController(text: widget.initialCode);
  @override
  void dispose() {
    _code.dispose();
    super.dispose();
  }

  String _date(String value) {
    final date = DateTime.tryParse(value);
    return date == null ? value : DateFormat('dd/MM/yyyy').format(date);
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Nguồn gốc thức quà', showBack: true),
    body: SafeArea(
      child: BlocBuilder<TraceCubit, TraceState>(
        builder: (context, state) {
          final cubit = context.read<TraceCubit>(), record = state.record;
          return ListView(
            padding: const EdgeInsets.all(20),
            children: [
              Text(
                'Kiểm chứng hành trình của từng thức quà',
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: 12),
              const Text(
                'Nhập mã QR in trên bao bì để xem lô sản xuất, nghệ nhân, nguyên liệu và chứng nhận.',
              ),
              const SizedBox(height: 24),
              AppTextField(
                controller: _code,
                label: 'Mã QR sản phẩm',
                maxLength: 160,
                textInputAction: TextInputAction.search,
                enabled: !state.loading,
                onSubmitted: cubit.lookup,
              ),
              const SizedBox(height: 12),
              AppPrimaryButton(
                label: 'Truy xuất ngay',
                loading: state.loading,
                onPressed: () {
                  FocusScope.of(context).unfocus();
                  cubit.lookup(_code.text);
                },
              ),
              if (AppConfig.dataSourceMode == DataSourceMode.mock)
                TextButton(
                  onPressed: state.loading
                      ? null
                      : () {
                          _code.text = 'QR-OMA-20261015-LOT08';
                          cubit.lookup(_code.text);
                        },
                  child: const Text('Xem hồ sơ mẫu local'),
                ),
              if (state.error != null)
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 20),
                  child: Semantics(
                    liveRegion: true,
                    child: Column(
                      children: [
                        Text(
                          state.error!,
                          style: const TextStyle(color: AppColors.darkRed),
                        ),
                        TextButton(
                          onPressed: state.code.isEmpty
                              ? null
                              : () => cubit.lookup(state.code),
                          child: const Text('Thử lại'),
                        ),
                        TextButton(
                          onPressed: () => context.go(AppRoutes.catalog),
                          child: const Text('Khám phá sản phẩm'),
                        ),
                      ],
                    ),
                  ),
                ),
              if (record != null) ...[
                _RecordCard(
                  title: record.product,
                  values: {
                    'Mã truy xuất': record.code,
                    'OCOP': '${record.stars} sao',
                    'Số chứng nhận': record.certificate,
                  },
                ),
                _RecordCard(
                  title: 'Lô sản xuất',
                  values: {
                    'Mã lô': record.batch,
                    'Ngày sản xuất': _date(record.productionDate),
                    'Hạn sử dụng': _date(record.expiryDate),
                    'Nghệ nhân': record.artisan,
                    'Xưởng sản xuất': record.workshop,
                  },
                ),
                _RecordCard(
                  title: 'Nguồn nguyên liệu',
                  values: {'Xuất xứ': record.origin},
                ),
                _RecordCard(
                  title: 'An toàn thực phẩm',
                  values: {'Chứng nhận kiểm định': record.safety},
                ),
                if (record.video.isNotEmpty)
                  PublicHttpsButton(
                    url: record.video,
                    label: 'Xem video chế tác',
                  ),
                OutlinedButton(
                  onPressed: () => context.go(
                    Uri(
                      path: AppRoutes.catalog,
                      queryParameters: {'q': record.product},
                    ).toString(),
                  ),
                  child: const Text('Tìm sản phẩm'),
                ),
              ],
            ],
          );
        },
      ),
    ),
  );
}

class _RecordCard extends StatelessWidget {
  const _RecordCard({required this.title, required this.values});
  final String title;
  final Map<String, String> values;
  @override
  Widget build(BuildContext context) => Card(
    margin: const EdgeInsets.symmetric(vertical: 12),
    child: Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 16),
          for (final entry in values.entries)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    entry.key,
                    style: const TextStyle(
                      fontSize: 12,
                      color: AppColors.textMuted,
                    ),
                  ),
                  const SizedBox(height: 4),
                  SelectableText(entry.value),
                ],
              ),
            ),
        ],
      ),
    ),
  );
}
