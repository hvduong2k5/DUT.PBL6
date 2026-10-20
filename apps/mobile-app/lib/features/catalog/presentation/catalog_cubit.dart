import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/models/page_result.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/features/catalog/domain/catalog_repository.dart';

enum CatalogStatus { initial, loading, success, failure }

enum CatalogSort { curated, priceAscending, priceDescending }

class CatalogFilters extends Equatable {
  const CatalogFilters({
    this.minimum,
    this.maximum,
    this.stars,
    this.availableOnly = false,
    this.sort = CatalogSort.curated,
  });
  final int? minimum, maximum, stars;
  final bool availableOnly;
  final CatalogSort sort;
  bool get active =>
      minimum != null ||
      maximum != null ||
      stars != null ||
      availableOnly ||
      sort != CatalogSort.curated;
  @override
  List<Object?> get props => [minimum, maximum, stars, availableOnly, sort];
}

class CatalogState extends Equatable {
  const CatalogState({
    this.status = CatalogStatus.initial,
    this.categories = const [],
    List<Product> products = const [],
    this.query = '',
    this.categoryId = 'all',
    this.message,
    this.pageMessage,
    this.searchError,
    this.page = 0,
    this.pageSize = 20,
    this.total = 0,
    this.loadingMore = false,
    this.filters = const CatalogFilters(),
  }) : loadedProducts = products;
  final CatalogStatus status;
  final List<Category> categories;
  final List<Product> loadedProducts;
  final String query;
  final String categoryId;
  final String? message;
  final String? pageMessage, searchError;
  final int page, pageSize, total;
  final bool loadingMore;
  final CatalogFilters filters;
  bool get hasMore => page * pageSize < total;
  bool get complete => loadedProducts.length >= total;
  bool get filtering => filters.active || categoryId != 'all';
  List<Product> get products {
    final category = categories.where((c) => c.id == categoryId).firstOrNull;
    final result = loadedProducts
        .where(
          (p) =>
              (categoryId == 'all' ||
                  p.category == categoryId ||
                  p.category == category?.name) &&
              (filters.stars == null || p.ocopStars == filters.stars) &&
              (!filters.availableOnly || p.stockKnown && p.inStock) &&
              (filters.minimum == null ||
                  p.price.currencyCode == 'VND' &&
                      p.price.compareTo(Money.vnd(filters.minimum!)) >= 0) &&
              (filters.maximum == null ||
                  p.price.currencyCode == 'VND' &&
                      p.price.compareTo(Money.vnd(filters.maximum!)) <= 0),
        )
        .toList();
    if (filters.sort != CatalogSort.curated) {
      result.sort((a, b) {
        final comparison = a.price.currencyCode == b.price.currencyCode
            ? a.price.compareTo(b.price)
            : a.price.currencyCode.compareTo(b.price.currencyCode);
        final value = filters.sort == CatalogSort.priceDescending
            ? -comparison
            : comparison;
        return value == 0 ? a.id.compareTo(b.id) : value;
      });
    }
    return result;
  }

  CatalogState copyWith({
    CatalogStatus? status,
    List<Category>? categories,
    List<Product>? products,
    String? query,
    String? categoryId,
    String? message,
    String? pageMessage,
    String? searchError,
    int? page,
    int? pageSize,
    int? total,
    bool? loadingMore,
    CatalogFilters? filters,
  }) => CatalogState(
    status: status ?? this.status,
    categories: categories ?? this.categories,
    products: products ?? loadedProducts,
    query: query ?? this.query,
    categoryId: categoryId ?? this.categoryId,
    message: message,
    pageMessage: pageMessage,
    searchError: searchError,
    page: page ?? this.page,
    pageSize: pageSize ?? this.pageSize,
    total: total ?? this.total,
    loadingMore: loadingMore ?? this.loadingMore,
    filters: filters ?? this.filters,
  );
  @override
  List<Object?> get props => [
    status,
    categories,
    loadedProducts,
    query,
    categoryId,
    message,
    pageMessage,
    searchError,
    page,
    pageSize,
    total,
    loadingMore,
    filters,
  ];
}

class CatalogCubit extends Cubit<CatalogState> {
  CatalogCubit(
    this._repository, {
    String initialQuery = '',
    String initialCategoryId = 'all',
  }) : super(CatalogState(query: initialQuery, categoryId: initialCategoryId));
  final CatalogRepository _repository;
  int _revision = 0;
  Future<void> load() async {
    final revision = ++_revision;
    final query = state.query.trim().replaceAll(RegExp(r'\s+'), ' ');
    if (query.length == 1 || query.length > 200) {
      emit(
        state.copyWith(
          status: CatalogStatus.success,
          query: query,
          products: [],
          total: 0,
          page: 0,
          loadingMore: false,
          searchError: 'Nhập từ khóa từ 2 đến 200 ký tự.',
        ),
      );
      return;
    }
    emit(
      state.copyWith(
        status: CatalogStatus.loading,
        query: query,
        products: [],
        total: 0,
        page: 0,
        loadingMore: false,
      ),
    );
    try {
      final values = await Future.wait([
        _repository.categories(),
        _repository.productPage(query: query),
      ]);
      if (isClosed || revision != _revision) return;
      final page = values[1] as PageResult<Product>;
      emit(
        state.copyWith(
          status: CatalogStatus.success,
          categories: [
            const Category(id: 'all', name: 'Tất cả', slug: 'all'),
            ...(values[0] as List<Category>).where((c) => c.id != 'all'),
          ],
          products: page.items,
          total: page.total,
          page: page.page,
          pageSize: page.pageSize,
        ),
      );
    } on ApiException catch (error) {
      if (isClosed || revision != _revision) return;
      emit(
        state.copyWith(
          status: CatalogStatus.failure,
          message: error.failure.message,
        ),
      );
    } catch (_) {
      if (isClosed || revision != _revision) return;
      emit(
        state.copyWith(
          status: CatalogStatus.failure,
          message: 'Không thể tải thực đơn. Vui lòng thử lại.',
        ),
      );
    }
  }

  Future<void> search(String value) async {
    emit(state.copyWith(query: value));
    await load();
  }

  Future<void> selectCategory(String id) async {
    emit(
      state.copyWith(
        categoryId: id,
        pageMessage: state.pageMessage,
        searchError: state.searchError,
      ),
    );
  }

  void setFilters(CatalogFilters filters) => emit(
    state.copyWith(
      filters: filters,
      pageMessage: state.pageMessage,
      searchError: state.searchError,
    ),
  );
  Future<void> clearAll() async {
    emit(
      state.copyWith(
        query: '',
        categoryId: 'all',
        filters: const CatalogFilters(),
      ),
    );
    await load();
  }

  Future<void> loadMore() async {
    if (state.status != CatalogStatus.success ||
        state.loadingMore ||
        !state.hasMore) {
      return;
    }
    final revision = _revision, next = state.page + 1;
    emit(state.copyWith(loadingMore: true));
    try {
      final result = await _repository.productPage(
        query: state.query,
        page: next,
        pageSize: state.pageSize,
      );
      if (isClosed || revision != _revision) return;
      final merged = {for (final p in state.loadedProducts) p.id: p};
      final previousCount = merged.length;
      for (final p in result.items) {
        merged[p.id] = p;
      }
      if (result.page != next ||
          merged.length == previousCount &&
              state.loadedProducts.length < result.total) {
        throw const FormatException('Pagination did not progress');
      }
      emit(
        state.copyWith(
          products: merged.values.toList(),
          total: result.total,
          page: result.page,
          pageSize: result.pageSize,
          loadingMore: false,
        ),
      );
    } catch (error) {
      if (isClosed || revision != _revision) return;
      emit(
        state.copyWith(
          loadingMore: false,
          pageMessage: error is ApiException
              ? error.failure.message
              : 'Chưa thể tải thêm. Thử lại hoặc tải mới danh mục.',
        ),
      );
    }
  }
}
