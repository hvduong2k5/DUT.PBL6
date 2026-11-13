import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/features/orders/domain/mvp_order_models.dart';
import 'package:mobile_app/features/orders/domain/mvp_orders_repository.dart';

class MvpOrderState {
  const MvpOrderState({
    this.loading = true,
    this.busy = false,
    this.order,
    this.command,
    this.storageBlocked = false,
    this.message,
  });
  final bool loading, busy, storageBlocked;
  final MvpOrder? order;
  final MvpCancellationCommand? command;
  final String? message;
  bool get canCancel =>
      !loading &&
      !busy &&
      !storageBlocked &&
      command == null &&
      (order?.actions.contains('CANCEL') ?? false);
}

class MvpOrderCubit extends Cubit<MvpOrderState> {
  MvpOrderCubit(this.runtime, this.orderId, this.principal)
    : super(const MvpOrderState());
  final MvpOrdersRuntime runtime;
  final String orderId;
  final String? Function() principal;
  int _generation = 0;
  bool _valid(int generation, String? owner) =>
      !isClosed &&
      generation == _generation &&
      owner != null &&
      principal() == owner;
  Future<void> load() async {
    if (isClosed || state.busy) return;
    final generation = ++_generation, owner = principal(), previous = state;
    emit(
      MvpOrderState(
        order: previous.order,
        command: previous.command,
        storageBlocked: previous.storageBlocked,
      ),
    );
    MvpOrder? order;
    try {
      final candidate = await runtime.repository.order(orderId);
      if (!_valid(generation, owner)) return;
      if (candidate.id != orderId ||
          previous.order != null &&
              candidate.version < previous.order!.version) {
        throw const FormatException('Regressive order snapshot');
      }
      order = candidate;
      var command = await runtime.cancellations.read(
        owner!,
        runtime.scope,
        orderId,
      );
      if (!_valid(generation, owner)) return;
      if (command != null &&
          (command.owner != owner ||
              command.scope != runtime.scope ||
              command.orderId != orderId)) {
        throw const FormatException('Cancellation ownership mismatch');
      }
      if (previous.command?.result != null && command?.result == null) {
        command = previous.command;
      }
      emit(MvpOrderState(loading: false, order: order, command: command));
    } catch (_) {
      if (_valid(generation, owner)) {
        emit(
          MvpOrderState(
            loading: false,
            order: order ?? previous.order,
            command: previous.command,
            storageBlocked: true,
            message:
                'Chưa thể đồng bộ đơn và yêu cầu hủy. Hãy tải lại trước khi thao tác.',
          ),
        );
      }
    }
  }

  Future<void> cancel(String reason) async {
    if (isClosed ||
        !state.canCancel ||
        reason.trim().isEmpty ||
        reason.trim().length > 255 ||
        principal() == null) {
      return;
    }
    final owner = principal()!,
        generation = ++_generation,
        shown = state.order!;
    emit(MvpOrderState(loading: false, busy: true, order: shown));
    try {
      final latest = await runtime.repository.order(orderId);
      if (!_valid(generation, owner)) return;
      if (latest.id != orderId || latest.version < shown.version) {
        throw const FormatException('Invalid order snapshot');
      }
      if (latest.version != shown.version ||
          !latest.actions.contains('CANCEL')) {
        emit(
          MvpOrderState(
            loading: false,
            order: latest,
            message:
                'Đơn đã thay đổi. Kiểm tra thông tin mới trước khi xác nhận lại.',
          ),
        );
        return;
      }
      final command = MvpCancellationCommand(
        owner: owner,
        scope: runtime.scope,
        orderId: orderId,
        key: const Uuid().v4(),
        reason: reason.trim(),
        version: shown.version,
      );
      await _send(command, generation);
    } catch (_) {
      if (_valid(generation, owner)) {
        emit(
          MvpOrderState(
            loading: false,
            order: shown,
            message: 'Chưa thể kiểm tra quyền hủy. Vui lòng tải lại.',
          ),
        );
      }
    }
  }

  Future<void> recoverCancellation() async {
    if (isClosed || state.busy || state.loading) return;
    final command = state.command;
    if (command == null) return load();
    if (principal() != command.owner) return;
    if (command.result == null) return _send(command, ++_generation);
    if (state.storageBlocked) {
      try {
        await runtime.cancellations.write(command);
      } catch (_) {
        return;
      }
    }
    await load();
  }

  Future<void> _send(MvpCancellationCommand command, int generation) async {
    final order = state.order;
    emit(
      MvpOrderState(loading: false, busy: true, order: order, command: command),
    );
    try {
      await runtime.cancellations.write(command);
    } catch (_) {
      if (_valid(generation, command.owner)) {
        emit(
          MvpOrderState(
            loading: false,
            order: order,
            command: command,
            storageBlocked: true,
            message:
                'Chưa lưu được yêu cầu hủy an toàn. Chưa gửi thêm thao tác.',
          ),
        );
      }
      return;
    }
    if (!_valid(generation, command.owner)) return;
    try {
      final result = await runtime.repository.cancel(command);
      if (!_valid(generation, command.owner)) return;
      if (result.orderId != orderId || result.version < command.version) {
        throw const FormatException('Invalid cancel result');
      }
      command = command.withResult(result);
      try {
        await runtime.cancellations.write(command);
      } catch (_) {
        if (_valid(generation, command.owner)) {
          emit(
            MvpOrderState(
              loading: false,
              order: order,
              command: command,
              storageBlocked: true,
              message:
                  'Đã nhận phản hồi hủy nhưng chưa lưu được. Không gửi yêu cầu mới.',
            ),
          );
        }
        return;
      }
      final latest = await runtime.repository.order(orderId);
      if (!_valid(generation, command.owner)) return;
      if (latest.id != orderId || latest.version < result.version) {
        throw const FormatException('Stale cancelled order');
      }
      emit(MvpOrderState(loading: false, order: latest, command: command));
    } catch (_) {
      if (_valid(generation, command.owner)) {
        emit(
          MvpOrderState(
            loading: false,
            order: order,
            command: command,
            message:
                'Chưa xác nhận được kết quả hủy. Kiểm tra lại yêu cầu này; không gửi lý do hoặc yêu cầu mới.',
          ),
        );
      }
    }
  }
}
