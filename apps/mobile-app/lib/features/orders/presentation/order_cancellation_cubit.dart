import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';

class OrderCancellationState {
  const OrderCancellationState({this.pending = false, this.status, this.error});
  final bool pending;
  final String? status, error;
}

class OrderCancellationCubit extends Cubit<OrderCancellationState> {
  OrderCancellationCubit(this._repository, this.orderId)
    : super(const OrderCancellationState());
  final OrdersRepository _repository;
  final String orderId;
  String? _reason, _key;
  Future<void> cancel(String input) async {
    if (state.pending || state.status != null) return;
    final reason = input.trim();
    if (reason.isEmpty || reason.length > 300) {
      emit(
        const OrderCancellationState(
          error: 'Nhập lý do hủy, tối đa 300 ký tự.',
        ),
      );
      return;
    }
    if (_reason != reason) {
      _reason = reason;
      _key = const Uuid().v4();
    }
    emit(const OrderCancellationState(pending: true));
    try {
      final status = await _repository.cancel(
        orderId,
        reason,
        idempotencyKey: _key!,
      );
      if (!isClosed) emit(OrderCancellationState(status: status));
    } catch (e) {
      if (!isClosed) {
        emit(
          OrderCancellationState(
            error: e is ApiException
                ? e.failure.message
                : 'Chưa thể hủy đơn. Vui lòng thử lại.',
          ),
        );
      }
    }
  }
}
