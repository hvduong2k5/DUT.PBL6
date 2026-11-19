import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/account/domain/customer_account_repository.dart';
import 'package:mobile_app/features/auth/domain/auth_validation.dart';
import 'package:mobile_app/core/error/failures.dart';

class ProfileState extends Equatable {
  const ProfileState({
    this.loading = true,
    this.saving = false,
    this.profile,
    this.message,
    this.saved = false,
    this.errors = const {},
    this.conflict = false,
  });
  final bool loading;
  final bool saving;
  final CustomerProfile? profile;
  final String? message;
  final bool saved;
  final Map<String, String> errors;
  final bool conflict;
  @override
  List<Object?> get props => [
    loading,
    saving,
    profile,
    message,
    saved,
    errors,
    conflict,
  ];
}

class ProfileCubit extends Cubit<ProfileState> {
  ProfileCubit(this._repository) : super(const ProfileState());
  final CustomerAccountRepository _repository;
  Future<void> load() async {
    emit(const ProfileState());
    try {
      final profile = await _repository.profile();
      if (!isClosed) emit(ProfileState(loading: false, profile: profile));
    } catch (error) {
      if (!isClosed) {
        emit(ProfileState(loading: false, message: _message(error)));
      }
    }
  }

  Future<void> save(String name, String email) async {
    if (state.saving || state.loading || state.profile == null) return;
    final errors = <String, String>{
      if (name.trim().isEmpty) 'full_name': 'Vui lòng nhập họ và tên.',
      if (!AuthValidation.email(email)) 'email': 'Vui lòng nhập email hợp lệ.',
    };
    final profile = state.profile;
    if (errors.isNotEmpty) {
      emit(ProfileState(loading: false, profile: profile, errors: errors));
      return;
    }
    emit(ProfileState(loading: false, saving: true, profile: profile));
    try {
      final result = await _repository.updateProfile(name.trim(), email.trim());
      if (!isClosed) {
        emit(
          ProfileState(
            loading: false,
            profile: result,
            saved: true,
            message: 'Hồ sơ Tri Kỷ đã được cập nhật.',
          ),
        );
      }
    } catch (error) {
      if (!isClosed) {
        emit(
          ProfileState(
            loading: false,
            profile: profile,
            message: _message(error),
            conflict:
                error is ApiException &&
                error.failure.kind == FailureKind.conflict,
            errors: error is ApiException
                ? error.failure.fieldErrors
                : const {},
          ),
        );
      }
    }
  }

  String _message(Object error) => error is ApiException
      ? error.failure.message
      : 'Chưa thể xử lý hồ sơ. Vui lòng thử lại.';
}
