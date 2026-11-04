import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/promotions/domain/promotions_repository.dart';

class PromotionsState extends Equatable {
  const PromotionsState({
    this.loading = true,
    this.vouchers = const [],
    this.error,
    this.code = '',
    this.subtotal = 0,
    this.validating = false,
    this.result,
    this.validationError,
  });
  final bool loading;
  final List<Voucher> vouchers;
  final String? error;
  final String code;
  final int subtotal;
  final bool validating;
  final VoucherValidation? result;
  final String? validationError;
  PromotionsState copyWith({
    bool? loading,
    List<Voucher>? vouchers,
    String? error,
    String? code,
    int? subtotal,
    bool? validating,
    VoucherValidation? result,
    String? validationError,
    bool keepResult = false,
    bool clearError = false,
  }) => PromotionsState(
    loading: loading ?? this.loading,
    vouchers: vouchers ?? this.vouchers,
    error: clearError ? null : error ?? this.error,
    code: code ?? this.code,
    subtotal: subtotal ?? this.subtotal,
    validating: validating ?? this.validating,
    result: keepResult ? this.result : result,
    validationError: validationError,
  );
  @override
  List<Object?> get props => [
    loading,
    vouchers,
    error,
    code,
    subtotal,
    validating,
    result,
    validationError,
  ];
}

class PromotionsCubit extends Cubit<PromotionsState> {
  PromotionsCubit(this._repository, {int subtotal = 0})
    : super(PromotionsState(subtotal: subtotal));
  final PromotionsRepository _repository;
  int _revision = 0;
  Future<void> load() async {
    emit(state.copyWith(loading: true, keepResult: true, clearError: true));
    try {
      final vouchers = await _repository.vouchers();
      if (!isClosed) {
        emit(
          state.copyWith(loading: false, vouchers: vouchers, keepResult: true),
        );
      }
    } catch (error) {
      if (!isClosed) {
        emit(
          state.copyWith(
            loading: false,
            error: _message(error),
            keepResult: true,
          ),
        );
      }
    }
  }

  void setCode(String code) {
    _revision++;
    emit(state.copyWith(code: code, validating: false));
  }

  void setSubtotal(String amount) {
    _revision++;
    emit(
      state.copyWith(subtotal: int.tryParse(amount) ?? 0, validating: false),
    );
  }

  Future<void> validate() async {
    if (state.validating) return;
    final code = state.code.trim().toUpperCase();
    if (code.isEmpty ||
        state.subtotal <= 0 ||
        state.subtotal > 9007199254740991) {
      emit(
        state.copyWith(
          validationError: 'Nhập mã ưu đãi và giá trị hàng hợp lệ.',
        ),
      );
      return;
    }
    final revision = ++_revision;
    emit(state.copyWith(code: code, validating: true));
    try {
      final result = await _repository.validate(code, state.subtotal);
      if (!isClosed && revision == _revision) {
        emit(
          state.copyWith(
            validating: false,
            result: result,
            validationError: result.valid ? null : result.message,
          ),
        );
      }
    } catch (error) {
      if (!isClosed && revision == _revision) {
        emit(
          state.copyWith(validating: false, validationError: _message(error)),
        );
      }
    }
  }

  String _message(Object error) => error is ApiException
      ? error.failure.message
      : 'Chưa thể kiểm tra ưu đãi. Vui lòng thử lại.';
}
