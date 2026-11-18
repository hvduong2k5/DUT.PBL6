import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/safe_navigation.dart';
import 'package:mobile_app/core/storage/search_history_store.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_surface.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/widgets/product_card.dart';
import 'package:mobile_app/features/catalog/presentation/widgets/product_filter_bar.dart';

/// Dedicated search screen: search repeatedly, results come from the catalog
/// API, and the 5 most recent queries are kept on the device.
class SearchPage extends StatefulWidget {
  const SearchPage({super.key, this.initialQuery = '', this.history});
  final String initialQuery;
  final SearchHistoryStore? history;

  @override
  State<SearchPage> createState() => _SearchPageState();
}

class _SearchPageState extends State<SearchPage> {
  late final _controller = TextEditingController(text: widget.initialQuery);
  late final _store = widget.history ?? SearchHistoryStore();
  final _focus = FocusNode();
  List<String> _history = const [];
  late String _submitted = widget.initialQuery.trim();

  @override
  void initState() {
    super.initState();
    _store.read().then((value) {
      if (mounted) setState(() => _history = value);
    });
    if (_submitted.isNotEmpty) {
      _store.add(_submitted).then((value) {
        if (mounted) setState(() => _history = value);
      });
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    _focus.dispose();
    super.dispose();
  }

  Future<void> _submit(String raw) async {
    final query = raw.trim();
    if (query.isEmpty) return;
    _controller.text = query;
    _focus.unfocus();
    setState(() => _submitted = query);
    context.read<CatalogCubit>().search(query);
    final history = await _store.add(query);
    if (mounted) setState(() => _history = history);
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      leading: IconButton(
        tooltip: 'Quay lại',
        icon: const Icon(Icons.arrow_back_ios_new),
        onPressed: () => safeBackOrHome(context),
      ),
      titleSpacing: 0,
      title: Padding(
        padding: const EdgeInsets.only(right: 12),
        child: TextField(
          key: const Key('search-input'),
          controller: _controller,
          focusNode: _focus,
          autofocus: widget.initialQuery.isEmpty,
          textInputAction: TextInputAction.search,
          onSubmitted: _submit,
          style: const TextStyle(color: HeritageColors.muc, fontSize: 15),
          decoration: InputDecoration(
            isDense: true,
            hintText: 'Tìm mè xửng, trà sen…',
            floatingLabelBehavior: FloatingLabelBehavior.never,
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 12,
              vertical: 12,
            ),
            suffixIcon: IconButton(
              tooltip: 'Tìm',
              icon: const Icon(Icons.search, color: HeritageColors.nau),
              onPressed: () => _submit(_controller.text),
            ),
          ),
        ),
      ),
    ),
    body: HeritageSurface(
      child: _submitted.isEmpty
          ? _historyView()
          // Same filter bar as the Danh mục tab, applied to search results.
          : Column(
              children: [
                const ProductFilterBar(),
                Expanded(child: _results()),
              ],
            ),
    ),
  );

  Widget _historyView() => ListView(
    key: const Key('search-history'),
    padding: const EdgeInsets.all(16),
    children: [
      Row(
        children: [
          Expanded(
            child: Text(
              'Tìm kiếm gần đây',
              style: Theme.of(context).textTheme.titleMedium,
            ),
          ),
          if (_history.isNotEmpty)
            TextButton(
              onPressed: () async {
                await _store.clear();
                if (mounted) setState(() => _history = const []);
              },
              child: const Text('Xóa lịch sử'),
            ),
        ],
      ),
      if (_history.isEmpty)
        const Padding(
          padding: EdgeInsets.symmetric(vertical: 12),
          child: Text(
            'Chưa có lượt tìm nào. Các từ khóa bạn tìm sẽ được lưu tại đây.',
          ),
        ),
      for (final query in _history)
        ListTile(
          contentPadding: EdgeInsets.zero,
          leading: const Icon(Icons.history, color: HeritageColors.nau),
          title: Text(query),
          onTap: () => _submit(query),
          trailing: IconButton(
            tooltip: 'Xóa "$query"',
            icon: const Icon(Icons.close, size: 20),
            onPressed: () async {
              final next = await _store.remove(query);
              if (mounted) setState(() => _history = next);
            },
          ),
        ),
    ],
  );

  Widget _results() => BlocBuilder<CatalogCubit, CatalogState>(
    builder: (context, state) {
      if (state.status == CatalogStatus.initial ||
          state.status == CatalogStatus.loading) {
        return const AppLoadingView();
      }
      if (state.status == CatalogStatus.failure) {
        return AppErrorView(
          message: state.message ?? 'Không thể tìm kiếm lúc này.',
          onRetry: context.read<CatalogCubit>().load,
        );
      }
      if (state.searchError != null) {
        return AppEmptyView(
          title: 'Từ khóa chưa hợp lệ',
          message: state.searchError!,
        );
      }
      final products = state.products;
      if (products.isEmpty) {
        return AppEmptyView(
          title: 'Không tìm thấy sản phẩm',
          message: 'Không có kết quả cho "$_submitted". Thử từ khóa khác.',
        );
      }
      return CustomScrollView(
        key: const Key('search-results'),
        slivers: [
          SliverPadding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
            sliver: SliverToBoxAdapter(
              child: Text(
                '${state.total} kết quả cho "$_submitted"',
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ),
          ),
          SliverPadding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            sliver: SliverGrid.builder(
              itemCount: products.length,
              gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                crossAxisCount: 2,
                childAspectRatio:
                    .5 / MediaQuery.textScalerOf(context).scale(1).clamp(1, 2),
                crossAxisSpacing: 12,
                mainAxisSpacing: 12,
              ),
              itemBuilder: (context, i) {
                final product = products[i];
                final path = '/product/${Uri.encodeComponent(product.id)}';
                return ProductCard(
                  product: product,
                  onOpen: () => context.push(path),
                  onAdd: () => context.push(path),
                );
              },
            ),
          ),
          if (state.hasMore)
            SliverPadding(
              padding: const EdgeInsets.all(16),
              sliver: SliverToBoxAdapter(
                child: OutlinedButton(
                  onPressed: state.loadingMore
                      ? null
                      : context.read<CatalogCubit>().loadMore,
                  child: Text(
                    state.loadingMore ? 'Đang tải…' : 'Tải thêm kết quả',
                  ),
                ),
              ),
            ),
          const SliverToBoxAdapter(child: SizedBox(height: 24)),
        ],
      );
    },
  );
}
