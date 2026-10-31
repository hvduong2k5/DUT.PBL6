import 'dart:async';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_repository.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';

class MvpCheckoutState {
  const MvpCheckoutState({
    this.ready = false,
    this.busy = false,
    this.storageBlocked = false,
    this.attempt,
    this.message,
  });
  final bool ready, busy, storageBlocked;
  final MvpCheckoutAttempt? attempt;
  final String? message;
  MvpCheckoutOperation? get operation => attempt?.operation;
  bool get unresolved => attempt != null && !(operation?.terminal ?? false);
  bool get canStart => ready && !busy && !storageBlocked && !unresolved;
}

class MvpCheckoutCubit extends Cubit<MvpCheckoutState> {
  MvpCheckoutCubit(
    this.runtime, {
    required this._principal,
    Stream<String?>? identityChanges,
    DateTime Function()? now,
  }) : _now = now ?? DateTime.now,
       super(const MvpCheckoutState()) {
    _identity = identityChanges?.distinct().listen((_) => unawaited(restore()));
  }
  final MvpCheckoutRuntime runtime;
  final String? Function() _principal;
  final DateTime Function() _now;
  StreamSubscription<String?>? _identity;
  String? _owner;
  int _generation = 0;

  bool _valid(int generation, String owner) =>
      !isClosed &&
      generation == _generation &&
      _owner == owner &&
      _principal() == owner;

  Future<void> restore() async {
    final generation = ++_generation;
    final owner = _owner = _principal();
    emit(const MvpCheckoutState());
    if (owner == null || owner.isEmpty) return;
    try {
      final attempt = await runtime.store.read(owner, runtime.scope);
      if (!_valid(generation, owner)) return;
      if (attempt != null &&
          (attempt.owner != owner || attempt.scope != runtime.scope)) {
        throw const FormatException('Stored checkout ownership mismatch');
      }
      emit(MvpCheckoutState(ready: true, attempt: attempt));
    } catch (_) {
      if (_valid(generation, owner)) {
        emit(
          const MvpCheckoutState(
            ready: true,
            storageBlocked: true,
            message:
                'Chưa thể đọc yêu cầu trước đó. Thử khôi phục lại trước khi đặt đơn mới.',
          ),
        );
      }
    }
  }

  Future<void> start(MvpQuote quote) async {
    if (!state.canStart || _owner == null || _principal() != _owner) return;
    if (state.attempt?.input.quoteId == quote.id) return;
    if (!quote.isUsableAt(_now())) {
      emit(
        MvpCheckoutState(
          ready: true,
          attempt: state.attempt,
          message: 'Báo giá đã hết hạn. Vui lòng lấy báo giá mới.',
        ),
      );
      return;
    }
    final attempt = MvpCheckoutAttempt(
      owner: _owner!,
      scope: runtime.scope,
      key: const Uuid().v4(),
      input: MvpCheckoutInput.fromQuote(quote),
    );
    await _run(attempt, persistBeforeSend: true);
  }

  Future<void> recover({bool automatic = false}) async {
    if (!state.ready ||
        state.busy ||
        _owner == null ||
        _principal() != _owner) {
      return;
    }
    final attempt = state.attempt;
    if (attempt == null) {
      if (!automatic) await restore();
      return;
    }
    if (automatic &&
        (state.storageBlocked ||
            state.message != null ||
            !(attempt.operation?.automaticPoll ?? false) ||
            (attempt.operation?.retryAt?.isAfter(_now()) ?? false))) {
      return;
    }
    if (attempt.operation?.terminal ?? false) {
      if (state.storageBlocked && !automatic) await _persistKnown(attempt);
      return;
    }
    await _run(
      attempt,
      persistBeforeSend: attempt.operation == null || state.storageBlocked,
    );
  }

  Future<void> _persistKnown(MvpCheckoutAttempt attempt) async {
    final generation = _generation;
    emit(MvpCheckoutState(ready: true, busy: true, attempt: attempt));
    try {
      await runtime.store.write(attempt);
      if (_valid(generation, attempt.owner)) {
        emit(MvpCheckoutState(ready: true, attempt: attempt));
      }
    } catch (_) {
      if (_valid(generation, attempt.owner)) {
        emit(
          MvpCheckoutState(
            ready: true,
            attempt: attempt,
            storageBlocked: true,
            message:
                'Chưa thể lưu kết quả. Không đặt lại đơn; hãy thử lưu lại.',
          ),
        );
      }
    }
  }

  Future<void> _run(
    MvpCheckoutAttempt attempt, {
    required bool persistBeforeSend,
  }) async {
    final generation = _generation;
    emit(MvpCheckoutState(ready: true, busy: true, attempt: attempt));
    if (persistBeforeSend) {
      try {
        await runtime.store.write(attempt);
      } catch (_) {
        if (_valid(generation, attempt.owner)) {
          emit(
            MvpCheckoutState(
              ready: true,
              attempt: attempt,
              storageBlocked: true,
              message:
                  'Chưa thể lưu yêu cầu an toàn. Chưa gửi thêm thao tác; hãy thử lại.',
            ),
          );
        }
        return;
      }
    }
    if (!_valid(generation, attempt.owner)) return;
    try {
      final previous = attempt.operation;
      final operation = attempt.pollId == null
          ? await runtime.repository.accept(attempt.input, key: attempt.key)
          : await runtime.repository.operation(attempt.pollId!);
      if (!_valid(generation, attempt.owner)) return;
      if (attempt.pollId != null && operation.id != attempt.pollId) {
        throw const FormatException('Unexpected recovery operation identity');
      }
      if (previous != null &&
          (operation.id != previous.id ||
              operation.acceptedAt != previous.acceptedAt ||
              operation.updatedAt.isBefore(previous.updatedAt))) {
        throw const FormatException('Unexpected operation snapshot');
      }
      final updated = attempt.withOperation(operation);
      try {
        await runtime.store.write(updated);
        if (_valid(generation, attempt.owner)) {
          emit(MvpCheckoutState(ready: true, attempt: updated));
        }
      } catch (_) {
        if (_valid(generation, attempt.owner)) {
          emit(
            MvpCheckoutState(
              ready: true,
              attempt: updated,
              storageBlocked: true,
              message:
                  'Đã nhận kết quả nhưng chưa lưu được. Kiểm tra yêu cầu này trước khi đặt thêm.',
            ),
          );
        }
      }
    } catch (error) {
      if (error is ApiException &&
          error.failure.operationId != null &&
          attempt.pollId == null &&
          _valid(generation, attempt.owner)) {
        try {
          final recovered = attempt.withRecoveryId(error.failure.operationId!);
          await runtime.store.write(recovered);
          if (_valid(generation, attempt.owner)) {
            emit(
              MvpCheckoutState(
                ready: true,
                attempt: recovered,
                message:
                    'Đã có mã yêu cầu. Kiểm tra lại kết quả trước khi đặt thêm.',
              ),
            );
          }
          return;
        } catch (_) {
          // Invalid/error metadata must never replace the original retry identity.
        }
      }
      if (_valid(generation, attempt.owner)) {
        emit(
          MvpCheckoutState(
            ready: true,
            attempt: attempt,
            message:
                'Chưa xác nhận được kết quả. Kiểm tra lại yêu cầu này để tránh tạo đơn trùng.',
          ),
        );
      }
    }
  }

  @override
  Future<void> close() async {
    ++_generation;
    await _identity?.cancel();
    return super.close();
  }
}
