import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/domain/auth_validation.dart';
import 'package:mobile_app/features/orders/domain/guest_orders_repository.dart';

class GuestLookupState extends Equatable {
  const GuestLookupState({
    this.pending = false,
    this.challenge,
    this.access,
    this.error,
    this.retryAt,
    this.verificationThrottled = false,
  });
  final bool pending;
  final GuestOrderChallenge? challenge;
  final GuestOrderAccess? access;
  final String? error;
  final DateTime? retryAt;
  final bool verificationThrottled;
  @override
  List<Object?> get props => [
    pending,
    challenge,
    access,
    error,
    retryAt,
    verificationThrottled,
  ];
}

class GuestLookupCubit extends Cubit<GuestLookupState> {
  GuestLookupCubit(this._repository) : super(const GuestLookupState());
  final GuestOrdersRepository _repository;
  String? _lastPayload;
  String? _key;
  bool get canRequest =>
      !state.pending && !(state.retryAt?.isAfter(DateTime.now()) ?? false);
  String _actionKey(String payload) {
    if (_lastPayload != payload) {
      _lastPayload = payload;
      _key = const Uuid().v4();
    }
    return _key!;
  }

  void useOtherInformation() {
    _lastPayload = null;
    _key = null;
    if (!state.pending) emit(GuestLookupState(retryAt: state.retryAt));
  }

  Future<void> resend(String order, String contact) async {
    if (!canRequest) return;
    _lastPayload = null;
    _key = null;
    await request(order, contact);
  }

  Future<void> request(String order, String contact) async {
    if (!canRequest) return;
    final normalized = contact.trim();
    if (order.trim().isEmpty ||
        (!AuthValidation.email(normalized) &&
            !AuthValidation.phone(normalized))) {
      emit(
        GuestLookupState(
          error: 'Nhập mã đơn và số điện thoại hoặc email đặt hàng hợp lệ.',
          retryAt: state.retryAt,
        ),
      );
      return;
    }
    emit(const GuestLookupState(pending: true));
    try {
      final challenge = await _repository.challenge(
        order.trim(),
        normalized,
        idempotencyKey: _actionKey('request:${order.trim()}:$normalized'),
      );
      if (!isClosed) {
        emit(
          GuestLookupState(
            challenge: challenge,
            retryAt: DateTime.now().add(
              Duration(seconds: challenge.resendAfterSeconds),
            ),
          ),
        );
      }
    } catch (error) {
      _failure(error);
    }
  }

  Future<void> verify(String otp) async {
    final challenge = state.challenge;
    if (state.pending ||
        challenge == null ||
        (state.verificationThrottled &&
            (state.retryAt?.isAfter(DateTime.now()) ?? false))) {
      return;
    }
    if (!challenge.expiresAt.isAfter(DateTime.now())) {
      emit(
        GuestLookupState(
          error: 'Mã đã hết hạn. Vui lòng gửi yêu cầu mới.',
          retryAt: state.retryAt,
        ),
      );
      return;
    }
    if (!RegExp(r'^\d{6}$').hasMatch(otp)) {
      emit(
        GuestLookupState(
          challenge: challenge,
          error: 'Mã xác minh gồm 6 chữ số.',
          retryAt: state.retryAt,
        ),
      );
      return;
    }
    emit(
      GuestLookupState(
        pending: true,
        challenge: challenge,
        retryAt: state.retryAt,
      ),
    );
    try {
      final access = await _repository.verify(
        challenge.id,
        otp,
        idempotencyKey: _actionKey('verify:${challenge.id}:$otp'),
      );
      if (!access.expiresAt.isAfter(DateTime.now())) {
        throw const ApiException(
          Failure(
            kind: FailureKind.expired,
            message: 'Quyền truy cập đã hết hạn. Vui lòng xác minh lại.',
          ),
        );
      }
      if (!isClosed) emit(GuestLookupState(access: access));
    } catch (error) {
      _failure(error);
    }
  }

  void _failure(Object error) {
    if (isClosed) return;
    final failure = error is ApiException ? error.failure : null;
    emit(
      GuestLookupState(
        challenge: failure?.kind == FailureKind.expired
            ? null
            : state.challenge,
        error: failure?.message ?? 'Chưa thể xác minh đơn. Vui lòng thử lại.',
        verificationThrottled:
            failure?.kind == FailureKind.rateLimited && state.challenge != null,
        retryAt: failure?.kind == FailureKind.rateLimited
            ? DateTime.now().add(
                Duration(seconds: failure?.retryAfterSeconds ?? 60),
              )
            : state.retryAt,
      ),
    );
  }

  @override
  Future<void> close() {
    _lastPayload = null;
    _key = null;
    return super.close();
  }
}
