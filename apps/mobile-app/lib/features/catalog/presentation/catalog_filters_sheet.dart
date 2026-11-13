import 'package:flutter/material.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_cubit.dart';

Future<CatalogFilters?> showCatalogFilters(
  BuildContext context,
  CatalogFilters initial,
) => showModalBottomSheet<CatalogFilters>(
  context: context,
  isScrollControlled: true,
  useSafeArea: true,
  showDragHandle: true,
  builder: (_) => _FiltersSheet(initial),
);

class _FiltersSheet extends StatefulWidget {
  const _FiltersSheet(this.initial);
  final CatalogFilters initial;
  @override
  State<_FiltersSheet> createState() => _FiltersSheetState();
}

class _FiltersSheetState extends State<_FiltersSheet> {
  late int? _minimum = widget.initial.minimum,
      _maximum = widget.initial.maximum,
      _stars = widget.initial.stars;
  late bool _available = widget.initial.availableOnly;
  @override
  Widget build(BuildContext context) => ConstrainedBox(
    constraints: BoxConstraints(
      maxHeight: MediaQuery.sizeOf(context).height * .85,
    ),
    child: SingleChildScrollView(
      padding: EdgeInsets.fromLTRB(
        24,
        8,
        24,
        24 + MediaQuery.viewInsetsOf(context).bottom,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            'Tìm thức quà vừa ý',
            style: Theme.of(context).textTheme.headlineSmall,
          ),
          const SizedBox(height: 16),
          Text('Khoảng giá', style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final preset
                  in const <({String label, int? minimum, int? maximum})>[
                    (label: 'Tất cả mức giá', minimum: null, maximum: null),
                    (label: 'Dưới 200.000đ', minimum: null, maximum: 200000),
                    (
                      label: '200.000đ – 500.000đ',
                      minimum: 200000,
                      maximum: 500000,
                    ),
                    (label: 'Trên 500.000đ', minimum: 500000, maximum: null),
                  ])
                ChoiceChip(
                  label: Text(preset.label),
                  selected:
                      _minimum == preset.minimum && _maximum == preset.maximum,
                  onSelected: (_) => setState(() {
                    _minimum = preset.minimum;
                    _maximum = preset.maximum;
                  }),
                ),
            ],
          ),
          const SizedBox(height: 24),
          Text(
            'Chứng nhận OCOP',
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final stars in const <int?>[null, 5, 4, 3])
                ChoiceChip(
                  label: Text(stars == null ? 'Tất cả' : '$stars sao'),
                  selected: _stars == stars,
                  onSelected: (_) => setState(() => _stars = stars),
                ),
            ],
          ),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('Chỉ sản phẩm đã xác nhận còn hàng'),
            value: _available,
            onChanged: (value) => setState(() => _available = value),
          ),
          const Text(
            'Lọc và sắp xếp trên các kết quả đã tải. Tiếp tục tải thêm để mở rộng phạm vi.',
          ),
          const SizedBox(height: 24),
          AppPrimaryButton(
            label: 'Áp dụng bộ lọc',
            onPressed: () => Navigator.of(context).pop(
              CatalogFilters(
                minimum: _minimum,
                maximum: _maximum,
                stars: _stars,
                availableOnly: _available,
                sort: widget.initial.sort,
              ),
            ),
          ),
          TextButton(
            onPressed: () => setState(() {
              _minimum = null;
              _maximum = null;
              _stars = null;
              _available = false;
            }),
            child: const Text('Đặt lại các lựa chọn'),
          ),
        ],
      ),
    ),
  );
}
