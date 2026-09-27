import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/features/catalog/domain/catalog_repository.dart';

enum CatalogStatus { initial, loading, success, failure }

class CatalogState extends Equatable {
  const CatalogState({
    this.status = CatalogStatus.initial,
    this.categories = const [],
    this.products = const [],
    this.query = '',
    this.categoryId = 'all',
    this.message,
  });
  final CatalogStatus status;
  final List<Category> categories;
  final List<Product> products;
  final String query;
  final String categoryId;
  final String? message;
  CatalogState copyWith({
    CatalogStatus? status,
    List<Category>? categories,
    List<Product>? products,
    String? query,
    String? categoryId,
    String? message,
  }) => CatalogState(
    status: status ?? this.status,
    categories: categories ?? this.categories,
    products: products ?? this.products,
    query: query ?? this.query,
    categoryId: categoryId ?? this.categoryId,
    message: message,
  );
  @override
  List<Object?> get props => [
    status,
    categories,
    products,
    query,
    categoryId,
    message,
  ];
}

class CatalogCubit extends Cubit<CatalogState> {
  CatalogCubit(this._repository) : super(const CatalogState());
  final CatalogRepository _repository;
  Future<void> load() async {
    emit(state.copyWith(status: CatalogStatus.loading));
    try {
      final values = await Future.wait([
        _repository.categories(),
        _repository.products(categoryId: state.categoryId, query: state.query),
      ]);
      emit(
        state.copyWith(
          status: CatalogStatus.success,
          categories: values[0] as List<Category>,
          products: values[1] as List<Product>,
        ),
      );
    } on ApiException catch (error) {
      emit(
        state.copyWith(
          status: CatalogStatus.failure,
          message: error.failure.message,
        ),
      );
    } catch (_) {
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
    emit(state.copyWith(categoryId: id));
    await load();
  }
}
