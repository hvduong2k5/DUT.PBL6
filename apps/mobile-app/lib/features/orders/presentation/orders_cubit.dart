import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/core/network/api_exception.dart';

enum OrdersStatus { initial, loading, success, failure }

class OrdersState extends Equatable {
  const OrdersState({
    this.status = OrdersStatus.initial,
    this.orders = const [],
    this.filter,
    this.message,
    this.query = '',
    this.total = 0,
    this.page = 0,
    this.pageSize = 10,
    this.loadingMore = false,
    this.pageMessage,
  });
  final OrdersStatus status;
  final List<CustomerOrder> orders;
  final OrderStatus? filter;
  final String? message;
  final String query;
  final int total, page, pageSize;
  final bool loadingMore;
  final String? pageMessage;
  bool get hasMore => page * pageSize < total;
  List<CustomerOrder> get visible => filter == null
      ? orders
      : orders.where((order) => order.status == filter).toList();
  OrdersState copyWith({
    OrdersStatus? status,
    List<CustomerOrder>? orders,
    OrderStatus? filter,
    bool clearFilter = false,
    String? message,
    String? query,
    int? total,
    int? page,
    int? pageSize,
    bool? loadingMore,
    String? pageMessage,
  }) => OrdersState(
    status: status ?? this.status,
    orders: orders ?? this.orders,
    filter: clearFilter ? null : filter ?? this.filter,
    message: message,
    query: query ?? this.query,
    total: total ?? this.total,
    page: page ?? this.page,
    pageSize: pageSize ?? this.pageSize,
    loadingMore: loadingMore ?? this.loadingMore,
    pageMessage: pageMessage,
  );
  @override
  List<Object?> get props => [
    status,
    orders,
    filter,
    message,
    query,
    total,
    page,
    pageSize,
    loadingMore,
    pageMessage,
  ];
}

class OrdersCubit extends Cubit<OrdersState> {
  OrdersCubit(this._repository) : super(const OrdersState());
  final OrdersRepository _repository;
  int _revision = 0;
  String? get _filterValue => switch (state.filter) {
    OrderStatus.pending => 'PENDING_PAYMENT',
    OrderStatus.confirmed => 'CONFIRMED',
    OrderStatus.shipping => 'SHIPPING',
    OrderStatus.delivered => 'DELIVERED',
    OrderStatus.cancelled => 'CANCELLED',
    _ => null,
  };
  Future<void> load() async {
    final revision = ++_revision;
    if (state.query.length > 200) {
      emit(
        state.copyWith(
          status: OrdersStatus.failure,
          message: 'Từ khóa tối đa 200 ký tự.',
        ),
      );
      return;
    }
    emit(
      state.copyWith(
        status: OrdersStatus.loading,
        loadingMore: false,
        page: 0,
        total: 0,
      ),
    );
    try {
      final result = await _repository.orderPage(
        query: state.query,
        status: _filterValue,
      );
      if (isClosed || revision != _revision) return;
      emit(
        state.copyWith(
          status: OrdersStatus.success,
          orders: result.items,
          total: result.total,
          page: result.page,
          pageSize: result.pageSize,
        ),
      );
    } catch (error) {
      if (isClosed || revision != _revision) return;
      emit(
        state.copyWith(
          status: OrdersStatus.failure,
          message: error is ApiException
              ? error.failure.message
              : 'Không thể tải lịch sử đơn hàng.',
        ),
      );
    }
  }

  Future<void> filter(OrderStatus? value) async {
    emit(state.copyWith(filter: value, clearFilter: value == null));
    await load();
  }

  Future<void> search(String value) async {
    emit(state.copyWith(query: value.trim()));
    await load();
  }

  Future<void> clearAll() async {
    emit(state.copyWith(query: '', clearFilter: true));
    await load();
  }

  Future<void> loadMore() async {
    if (state.status != OrdersStatus.success ||
        state.loadingMore ||
        !state.hasMore) {
      return;
    }
    final revision = _revision, next = state.page + 1;
    emit(state.copyWith(loadingMore: true));
    try {
      final result = await _repository.orderPage(
        query: state.query,
        status: _filterValue,
        page: next,
        pageSize: state.pageSize,
      );
      if (isClosed || revision != _revision) return;
      final merged = {for (final order in state.orders) order.id: order};
      final count = merged.length;
      for (final order in result.items) {
        merged[order.id] = order;
      }
      if (result.page != next ||
          merged.length == count && state.orders.length < result.total) {
        throw const FormatException('No pagination progress');
      }
      emit(
        state.copyWith(
          orders: merged.values.toList(),
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
              : 'Chưa thể tải thêm đơn. Vui lòng thử lại.',
        ),
      );
    }
  }
}
