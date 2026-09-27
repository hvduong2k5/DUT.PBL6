import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';

enum OrdersStatus { initial, loading, success, failure }

class OrdersState extends Equatable {
  const OrdersState({
    this.status = OrdersStatus.initial,
    this.orders = const [],
    this.filter,
    this.message,
  });
  final OrdersStatus status;
  final List<CustomerOrder> orders;
  final OrderStatus? filter;
  final String? message;
  List<CustomerOrder> get visible => filter == null
      ? orders
      : orders.where((order) => order.status == filter).toList();
  OrdersState copyWith({
    OrdersStatus? status,
    List<CustomerOrder>? orders,
    OrderStatus? filter,
    bool clearFilter = false,
    String? message,
  }) => OrdersState(
    status: status ?? this.status,
    orders: orders ?? this.orders,
    filter: clearFilter ? null : filter ?? this.filter,
    message: message,
  );
  @override
  List<Object?> get props => [status, orders, filter, message];
}

class OrdersCubit extends Cubit<OrdersState> {
  OrdersCubit(this._repository) : super(const OrdersState());
  final OrdersRepository _repository;
  Future<void> load() async {
    emit(state.copyWith(status: OrdersStatus.loading));
    try {
      emit(
        state.copyWith(
          status: OrdersStatus.success,
          orders: await _repository.orders(),
        ),
      );
    } catch (_) {
      emit(
        state.copyWith(
          status: OrdersStatus.failure,
          message: 'Không thể tải lịch sử đơn hàng.',
        ),
      );
    }
  }

  void filter(OrderStatus? value) =>
      emit(state.copyWith(filter: value, clearFilter: value == null));
}
