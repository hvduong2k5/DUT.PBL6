import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/orders/domain/mvp_order_models.dart';
import 'package:mobile_app/features/orders/domain/mvp_orders_repository.dart';

class MvpPaymentState {
  const MvpPaymentState({
    this.loading = true,
    this.order,
    this.status,
    this.instructions,
    this.message,
    this.serverAtRead,
    this.elapsedAtRead = Duration.zero,
  });
  final bool loading;
  final MvpOrder? order;
  final MvpPaymentStatus? status;
  final MvpPaymentInstructions? instructions;
  final String? message;
  final DateTime? serverAtRead;
  final Duration elapsedAtRead;
}

class MvpPaymentCubit extends Cubit<MvpPaymentState> {
  MvpPaymentCubit(
    this.repository,
    this.orderId,
    this.principal, {
    Duration Function()? elapsed,
  }) : _clock = Stopwatch()..start(),
       _elapsedOverride = elapsed,
       super(const MvpPaymentState());
  final MvpOrdersRepository repository;
  final String orderId;
  final String? Function() principal;
  final Stopwatch _clock;
  final Duration Function()? _elapsedOverride;
  Duration get _elapsed => _elapsedOverride?.call() ?? _clock.elapsed;
  int _generation = 0;
  bool _valid(int generation, String? owner) =>
      !isClosed &&
      generation == _generation &&
      owner != null &&
      owner == principal();
  DateTime? get serverNow =>
      state.serverAtRead?.add(_elapsed - state.elapsedAtRead);
  bool get instructionsUsable =>
      !state.loading &&
      state.message == null &&
      state.order?.canPay == true &&
      state.instructions != null &&
      serverNow != null &&
      (state.status?.payableAt(serverNow!) ?? false) &&
      state.instructions!.expiresAt.difference(serverNow!) >
          const Duration(seconds: 30);
  bool get shouldPoll =>
      !state.loading &&
      state.message == null &&
      state.status != null &&
      state.status!.nextPollSeconds > 0 &&
      _elapsed - state.elapsedAtRead >=
          Duration(seconds: state.status!.nextPollSeconds);
  bool _matches(MvpPaymentInstructions value, MvpPaymentStatus status) =>
      value.orderId == orderId &&
      value.intentId == status.intentId &&
      value.amount == status.expected &&
      value.expiresAt == status.expiresAt;
  Future<void> load() async {
    if (isClosed) return;
    if (state.loading && _generation > 0) return;
    final generation = ++_generation, owner = principal(), previous = state;
    emit(MvpPaymentState(order: previous.order, status: previous.status));
    MvpOrder? order;
    MvpPaymentStatus? status;
    try {
      if (owner == null) throw StateError('Unauthenticated');
      final orderCandidate = await repository.order(orderId);
      if (!_valid(generation, owner)) return;
      if (orderCandidate.id != orderId ||
          previous.order != null &&
              orderCandidate.version < previous.order!.version) {
        throw const FormatException('Stale payment order');
      }
      order = orderCandidate;
      if (order.paymentMethod != 'VIETQR') {
        emit(
          MvpPaymentState(
            loading: false,
            order: order,
            message:
                'Phương thức của đơn này không dùng hướng dẫn chuyển khoản VietQR.',
          ),
        );
        return;
      }
      var before = _elapsed;
      final statusCandidate = await repository.payment(orderId);
      if (!_valid(generation, owner)) return;
      if (statusCandidate.orderId != orderId ||
          previous.status != null &&
              (statusCandidate.intentId != previous.status!.intentId ||
                  statusCandidate.version < previous.status!.version)) {
        throw const FormatException('Stale payment projection');
      }
      status = statusCandidate;
      var effective = status.serverTime.add(_elapsed - before);
      var receivedAt = _elapsed;
      MvpPaymentInstructions? instructions;
      if (order.canPay && status.payableAt(effective)) {
        instructions = previous.instructions;
        if (instructions == null || !_matches(instructions, status)) {
          before = _elapsed;
          instructions = await repository.instructions(orderId);
          if (!_valid(generation, owner)) return;
          final instructionsNow = instructions.serverTime.add(
            _elapsed - before,
          );
          effective = effective.add(_elapsed - receivedAt);
          if (instructionsNow.isAfter(effective)) effective = instructionsNow;
          receivedAt = _elapsed;
        }
        if (!_matches(instructions, status)) {
          throw const FormatException('Payment instructions mismatch');
        }
      }
      emit(
        MvpPaymentState(
          loading: false,
          order: order,
          status: status,
          instructions: instructions,
          serverAtRead: effective,
          elapsedAtRead: receivedAt,
        ),
      );
    } catch (_) {
      if (_valid(generation, owner)) {
        emit(
          MvpPaymentState(
            loading: false,
            order: order ?? previous.order,
            status: status ?? previous.status,
            message:
                'Chưa thể xác nhận thông tin thanh toán. Hướng dẫn chuyển khoản tạm ẩn; vui lòng kiểm tra lại.',
          ),
        );
      }
    }
  }

  @override
  Future<void> close() {
    ++_generation;
    _clock.stop();
    return super.close();
  }
}
