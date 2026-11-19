import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_filters_sheet.dart';

/// Shared browse/search filter bar (Shopee/Tiki pattern, no side rail):
/// a row of category chips, then "Bộ lọc" + sort, then active filters.
/// Reads and drives the nearest [CatalogCubit].
class ProductFilterBar extends StatelessWidget {
  const ProductFilterBar({super.key, this.showCategories = true});
  final bool showCategories;

  static String sortLabel(CatalogSort sort, {required bool searching}) =>
      switch (sort) {
        CatalogSort.curated => searching ? 'Liên quan nhất' : 'Tuyển chọn',
        CatalogSort.priceAscending => 'Giá thấp đến cao',
        CatalogSort.priceDescending => 'Giá cao đến thấp',
      };

  @override
  Widget build(BuildContext context) => BlocBuilder<CatalogCubit, CatalogState>(
    builder: (context, state) {
      final cubit = context.read<CatalogCubit>();
      final f = state.filters;
      final activeCount =
          (f.minimum != null || f.maximum != null ? 1 : 0) +
          (f.stars != null ? 1 : 0) +
          (f.availableOnly ? 1 : 0);
      return Material(
        key: const Key('product-filter-bar'),
        color: HeritageColors.kem,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (showCategories && state.categories.isNotEmpty)
              SizedBox(
                height: 52,
                child: ListView.separated(
                  key: const Key('category-chips'),
                  padding: const EdgeInsets.fromLTRB(16, 8, 16, 4),
                  scrollDirection: Axis.horizontal,
                  itemCount: state.categories.length,
                  separatorBuilder: (_, _) => const SizedBox(width: 8),
                  itemBuilder: (context, i) {
                    final c = state.categories[i];
                    return ChoiceChip(
                      label: Text(c.name),
                      selected: state.categoryId == c.id,
                      onSelected: (_) => cubit.selectCategory(c.id),
                    );
                  },
                ),
              ),
            Padding(
              padding: const EdgeInsets.fromLTRB(8, 0, 8, 0),
              child: Row(
                children: [
                  TextButton.icon(
                    icon: const Icon(Icons.tune, size: 20),
                    label: Text(
                      activeCount == 0 ? 'Bộ lọc' : 'Bộ lọc ($activeCount)',
                    ),
                    onPressed: () async {
                      final result = await showCatalogFilters(
                        context,
                        state.filters,
                        categories: state.categories,
                        categoryId: state.categoryId,
                      );
                      if (result == null || !context.mounted) return;
                      cubit.setFilters(result.filters);
                      if (result.categoryId != state.categoryId) {
                        cubit.selectCategory(result.categoryId);
                      }
                    },
                  ),
                  const Spacer(),
                  PopupMenuButton<CatalogSort>(
                    tooltip: 'Sắp xếp',
                    initialValue: f.sort,
                    onSelected: (sort) => cubit.setFilters(
                      CatalogFilters(
                        minimum: f.minimum,
                        maximum: f.maximum,
                        stars: f.stars,
                        availableOnly: f.availableOnly,
                        sort: sort,
                      ),
                    ),
                    itemBuilder: (_) => [
                      for (final sort in CatalogSort.values)
                        PopupMenuItem(
                          value: sort,
                          child: Text(
                            sortLabel(sort, searching: state.query.isNotEmpty),
                          ),
                        ),
                    ],
                    child: Padding(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 12,
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          const Icon(
                            Icons.swap_vert,
                            size: 20,
                            color: HeritageColors.nauDam,
                          ),
                          const SizedBox(width: 4),
                          Text(
                            sortLabel(
                              f.sort,
                              searching: state.query.isNotEmpty,
                            ),
                            style: const TextStyle(
                              fontWeight: FontWeight.w600,
                              color: HeritageColors.nauDam,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
            if (activeCount > 0)
              SizedBox(
                height: 44,
                child: ListView(
                  padding: const EdgeInsets.fromLTRB(16, 0, 16, 6),
                  scrollDirection: Axis.horizontal,
                  children: [
                    if (f.minimum != null || f.maximum != null)
                      _ActiveChip(
                        label: _priceLabel(f.minimum, f.maximum),
                        onRemove: () => cubit.setFilters(
                          CatalogFilters(
                            stars: f.stars,
                            availableOnly: f.availableOnly,
                            sort: f.sort,
                          ),
                        ),
                      ),
                    if (f.stars != null)
                      _ActiveChip(
                        label: 'OCOP ${f.stars} sao',
                        onRemove: () => cubit.setFilters(
                          CatalogFilters(
                            minimum: f.minimum,
                            maximum: f.maximum,
                            availableOnly: f.availableOnly,
                            sort: f.sort,
                          ),
                        ),
                      ),
                    if (f.availableOnly)
                      _ActiveChip(
                        label: 'Còn hàng',
                        onRemove: () => cubit.setFilters(
                          CatalogFilters(
                            minimum: f.minimum,
                            maximum: f.maximum,
                            stars: f.stars,
                            sort: f.sort,
                          ),
                        ),
                      ),
                  ],
                ),
              ),
            const Divider(height: 1),
          ],
        ),
      );
    },
  );

  static String _priceLabel(int? minimum, int? maximum) {
    String m(int v) => Money.vnd(v).format();
    if (minimum == null) return 'Dưới ${m(maximum!)}';
    if (maximum == null) return 'Trên ${m(minimum)}';
    return '${m(minimum)} – ${m(maximum)}';
  }
}

class _ActiveChip extends StatelessWidget {
  const _ActiveChip({required this.label, required this.onRemove});
  final String label;
  final VoidCallback onRemove;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(right: 8),
    child: InputChip(
      label: Text(label),
      onDeleted: onRemove,
      deleteButtonTooltipMessage: 'Bỏ lọc $label',
    ),
  );
}
