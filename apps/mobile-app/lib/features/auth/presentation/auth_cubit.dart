import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';

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
    if (phone.trim().isEmpty || password.isEmpty) {
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
        phoneNumber: phone.trim(),
        password: password,
      );
      emit(AuthState(status: AuthStatus.authenticated, user: session.user));
      return true;
    } catch (_) {
      emit(
        const AuthState(
          status: AuthStatus.failure,
          message: 'Thông tin đăng nhập chưa đúng. Vui lòng thử lại.',
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
    if (name.trim().isEmpty ||
        !RegExp(r'^0\d{9}$').hasMatch(phone.trim()) ||
        password.length < 8) {
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
        phoneNumber: phone.trim(),
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
}
