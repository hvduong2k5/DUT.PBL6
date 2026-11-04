import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/auth/domain/auth_validation.dart';
import 'package:mobile_app/features/auth/domain/recovery_verification.dart';
import 'package:uuid/uuid.dart';

enum RecoveryPhase { request, sent, verifying, reset, success }

class RecoveryState extends Equatable {
  const RecoveryState({
    this.phase = RecoveryPhase.request,
    this.pending = false,
    this.email = '',
    this.message,
    this.fieldErrors = const {},
    this.retryAt,
  });
  final RecoveryPhase phase;
  final bool pending;
  final String email;
  final String? message;
  final Map<String, String> fieldErrors;
  final DateTime? retryAt;

  @override
  List<Object?> get props => [
    phase,
    pending,
    email,
    message,
    fieldErrors,
    retryAt,
  ];
}

class RecoveryCubit extends Cubit<RecoveryState> {
  RecoveryCubit(this._repository) : super(const RecoveryState());
  final AuthRepository _repository;
  RecoveryVerification? _verification;
  String? _lastPayload;
  String? _key;

  bool get canSubmit =>
      !state.pending && !(state.retryAt?.isAfter(DateTime.now()) ?? false);

  String _actionKey(String payload) {
    if (_lastPayload != payload) {
      _key = const Uuid().v4();
      _lastPayload = payload;
    }
    return _key!;
  }

  void newRequest() {
    if (state.pending) return;
    _verification = null;
    _lastPayload = null;
    _key = null;
    emit(RecoveryState(email: state.email, retryAt: state.retryAt));
  }

  Future<void> request(String email) async {
    if (!canSubmit) return;
    final normalized = email.trim();
    if (!AuthValidation.email(normalized)) {
      emit(
        const RecoveryState(
          fieldErrors: {'email': 'Vui lòng nhập email hợp lệ.'},
        ),
      );
      return;
    }
    emit(RecoveryState(pending: true, email: normalized));
    try {
      await _repository.requestRecovery(
        normalized,
        idempotencyKey: _actionKey('request:$normalized'),
      );
      if (!isClosed) {
        emit(RecoveryState(phase: RecoveryPhase.sent, email: normalized));
      }
    } catch (error) {
      _failure(error, RecoveryPhase.request);
    }
  }

  Future<void> verify(String token) async {
    if (!canSubmit) return;
    _verification = null;
    emit(
      RecoveryState(
        phase: RecoveryPhase.verifying,
        pending: true,
        email: state.email,
      ),
    );
    try {
      final result = await _repository.verifyRecovery(
        token,
        idempotencyKey: _actionKey('verify:$token'),
      );
      if (isClosed) return;
      if (!result.expiresAt.isAfter(DateTime.now())) {
        throw const ApiException(
          Failure(
            kind: FailureKind.expired,
            message: 'Liên kết đã hết hạn. Vui lòng gửi yêu cầu mới.',
          ),
        );
      }
      _verification = result;
      emit(RecoveryState(phase: RecoveryPhase.reset, email: state.email));
    } catch (error) {
      _failure(error, RecoveryPhase.request);
    }
  }

  Future<void> reset(String password, String confirmation) async {
    if (!canSubmit || state.phase != RecoveryPhase.reset) return;
    final proof = _verification;
    if (proof == null || !proof.expiresAt.isAfter(DateTime.now())) {
      _verification = null;
      emit(
        const RecoveryState(
          message: 'Liên kết đã hết hạn. Vui lòng gửi yêu cầu mới.',
        ),
      );
      return;
    }
    final errors = <String, String>{
      if (!AuthValidation.password(password))
        'newPassword': 'Mật khẩu chưa đáp ứng đầy đủ điều kiện.',
      if (password != confirmation)
        'newPasswordConfirmation': 'Mật khẩu xác nhận chưa trùng khớp.',
    };
    if (errors.isNotEmpty) {
      emit(RecoveryState(phase: RecoveryPhase.reset, fieldErrors: errors));
      return;
    }
    emit(const RecoveryState(phase: RecoveryPhase.reset, pending: true));
    try {
      await _repository.resetPassword(
        proof.resetProof,
        password,
        idempotencyKey: _actionKey('reset:${proof.resetProof}:$password'),
      );
      _verification = null;
      _lastPayload = null;
      _key = null;
      if (!isClosed) emit(const RecoveryState(phase: RecoveryPhase.success));
    } catch (error) {
      if (error is ApiException && error.failure.kind == FailureKind.expired) {
        _verification = null;
        _failure(error, RecoveryPhase.request);
      } else {
        _failure(error, RecoveryPhase.reset);
      }
    }
  }

  void _failure(Object error, RecoveryPhase phase) {
    if (isClosed) return;
    final failure = error is ApiException ? error.failure : null;
    emit(
      RecoveryState(
        phase: phase,
        email: state.email,
        message:
            failure?.message ?? 'Chưa thể xử lý yêu cầu. Vui lòng thử lại.',
        fieldErrors: failure?.fieldErrors ?? const {},
        retryAt: failure?.kind == FailureKind.rateLimited
            ? DateTime.now().add(
                Duration(seconds: failure?.retryAfterSeconds ?? 60),
              )
            : null,
      ),
    );
  }

  @override
  Future<void> close() {
    _verification = null;
    _lastPayload = null;
    _key = null;
    return super.close();
  }
}
