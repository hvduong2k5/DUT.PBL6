import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/auth/domain/auth_validation.dart';
import 'package:mobile_app/core/network/api_exception.dart';

enum AuthStatus { guest, restoring, submitting, authenticated, failure }

class AuthState extends Equatable {
  const AuthState({
    this.status = AuthStatus.guest,
    this.user,
    this.message,
    this.sessionRestoreFailed = false,
  });
  final AuthStatus status;
  final AuthUser? user;
  final String? message;
  final bool sessionRestoreFailed;
  bool get pending =>
      status == AuthStatus.restoring || status == AuthStatus.submitting;
  @override
  List<Object?> get props => [status, user, message, sessionRestoreFailed];
}

class AuthCubit extends Cubit<AuthState> {
  AuthCubit(this._repository) : super(const AuthState()) {
    final Object source = _repository;
    if (source is SessionRecoverySource) {
      source.observeSessionRecovery((user) {
        if (isClosed || state.pending || state.user == null) return;
        ++_sessionRevision;
        emit(
          user == null
              ? const AuthState(
                  message:
                      'Phiên đăng nhập đã hết hạn. Vui lòng đăng nhập lại.',
                )
              : AuthState(status: AuthStatus.authenticated, user: user),
        );
      });
    }
  }
  final AuthRepository _repository;
  int _sessionRevision = 0;

  Future<void> restore() async {
    if (state.status == AuthStatus.submitting ||
        state.status == AuthStatus.restoring) {
      return;
    }
    final revision = ++_sessionRevision;
    emit(const AuthState(status: AuthStatus.restoring));
    try {
      final user = await _repository.restoreSession();
      if (isClosed || revision != _sessionRevision) return;
      emit(
        user == null
            ? const AuthState()
            : AuthState(status: AuthStatus.authenticated, user: user),
      );
    } catch (_) {
      if (!isClosed && revision == _sessionRevision) {
        emit(
          const AuthState(
            status: AuthStatus.failure,
            sessionRestoreFailed: true,
            message: 'Chưa thể khôi phục phiên. Hãy thử lại khi có kết nối.',
          ),
        );
      }
    }
  }

  Future<bool> login(String phone, String password) async {
    if (state.status == AuthStatus.submitting ||
        state.status == AuthStatus.restoring) {
      return false;
    }
    ++_sessionRevision;
    if (!AuthValidation.phone(phone) || password.isEmpty) {
      emit(
        const AuthState(
          status: AuthStatus.failure,
          message: 'Vui lòng nhập đủ thông tin.',
        ),
      );
      return false;
    }
    emit(const AuthState(status: AuthStatus.submitting));
    try {
      final session = await _repository.login(
        phoneNumber: AuthValidation.normalizePhone(phone),
        password: password,
      );
      if (isClosed) return false;
      emit(AuthState(status: AuthStatus.authenticated, user: session.user));
      return true;
    } catch (error) {
      if (isClosed) return false;
      emit(
        AuthState(
          status: AuthStatus.failure,
          message: error is ApiException
              ? error.failure.message
              : 'Thông tin đăng nhập chưa đúng. Vui lòng thử lại.',
        ),
      );
      return false;
    }
  }

  Future<bool> register({
    required String name,
    required String phone,
    required String password,
    String? email,
  }) async {
    if (state.status == AuthStatus.submitting ||
        state.status == AuthStatus.restoring) {
      return false;
    }
    ++_sessionRevision;
    if (name.trim().isEmpty ||
        !AuthValidation.phone(phone) ||
        !AuthValidation.password(password) ||
        !AuthValidation.email(email ?? '')) {
      emit(
        const AuthState(
          status: AuthStatus.failure,
          message: 'Vui lòng kiểm tra thông tin đăng ký.',
        ),
      );
      return false;
    }
    emit(const AuthState(status: AuthStatus.submitting));
    try {
      final session = await _repository.register(
        phoneNumber: AuthValidation.normalizePhone(phone),
        password: password,
        fullName: name.trim(),
        email: email?.trim(),
      );
      if (isClosed) return false;
      emit(AuthState(status: AuthStatus.authenticated, user: session.user));
      return true;
    } catch (_) {
      if (isClosed) return false;
      emit(
        const AuthState(
          status: AuthStatus.failure,
          message: 'Không thể đăng ký lúc này.',
        ),
      );
      return false;
    }
  }

  Future<void> logout() async {
    ++_sessionRevision;
    try {
      await _repository.logout();
      if (!isClosed) emit(const AuthState());
    } catch (_) {
      if (!isClosed) {
        emit(
          const AuthState(
            message:
                'Đã đăng xuất trên thiết bị. Chưa xác nhận kết thúc phiên phía máy chủ.',
          ),
        );
      }
    }
  }

  void resetToGuest() {
    ++_sessionRevision;
    emit(const AuthState());
  }

  void updateUser(AuthUser user) {
    if (state.status == AuthStatus.authenticated && state.user?.id == user.id) {
      emit(AuthState(status: AuthStatus.authenticated, user: user));
    }
  }

  @override
  Future<void> close() {
    final Object source = _repository;
    if (source is SessionRecoverySource) {
      source.observeSessionRecovery(null);
    }
    return super.close();
  }
}
