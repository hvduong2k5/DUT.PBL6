import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/auth/domain/auth_validation.dart';
import 'package:mobile_app/core/network/api_exception.dart';

enum AuthStatus { guest, submitting, authenticated, failure }

class AuthState extends Equatable {
  const AuthState({this.status = AuthStatus.guest, this.user, this.message});
  final AuthStatus status;
  final AuthUser? user;
  final String? message;
  @override
  List<Object?> get props => [status, user, message];
}

class AuthCubit extends Cubit<AuthState> {
  AuthCubit(this._repository) : super(const AuthState());
  final AuthRepository _repository;

  Future<bool> login(String phone, String password) async {
    if (state.status == AuthStatus.submitting) return false;
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
      emit(AuthState(status: AuthStatus.authenticated, user: session.user));
      return true;
    } catch (error) {
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
    if (state.status == AuthStatus.submitting) return false;
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
      emit(AuthState(status: AuthStatus.authenticated, user: session.user));
      return true;
    } catch (_) {
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
    await _repository.logout();
    emit(const AuthState());
  }

  void resetToGuest() => emit(const AuthState());
}
