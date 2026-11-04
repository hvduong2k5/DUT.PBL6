import 'dart:convert';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/returns/domain/returns_repository.dart';

class ReturnsState {
  const ReturnsState({
    this.loading = false,
    this.pending = false,
    this.context,
    this.created,
    this.error,
  });
  final bool loading, pending;
  final ReturnFormContext? context;
  final ReturnCase? created;
  final String? error;
}

class ReturnsCubit extends Cubit<ReturnsState> {
  ReturnsCubit(this._repository) : super(const ReturnsState());
  final ReturnsRepository _repository;
  String? _key, _fingerprint;
  Future<void> load(String orderId) async {
    if (state.loading || state.pending) return;
    emit(const ReturnsState(loading: true));
    try {
      final ctx = await _repository.context(orderId);
      if (!isClosed) emit(ReturnsState(context: ctx));
    } catch (e) {
      _fail(e);
    }
  }

  void _fail(Object error) {
    if (!isClosed) {
      emit(
        ReturnsState(
          context: state.context,
          error: error is ApiException
              ? error.failure.message
              : 'Chưa thể xử lý hồ sơ. Vui lòng thử lại.',
        ),
      );
    }
  }

  Future<void> submit(ReturnDraft draft) async {
    final ctx = state.context;
    if (ctx == null ||
        ctx.blockingMessage != null ||
        state.pending ||
        state.created != null) {
      return;
    }
    final line = ctx.lines.where((l) => l.id == draft.lineId).firstOrNull;
    final mediaValid = draft.evidenceUrls.every((value) {
      final uri = Uri.tryParse(value);
      return uri != null &&
          uri.scheme == 'https' &&
          uri.host.isNotEmpty &&
          uri.userInfo.isEmpty;
    });
    if (draft.orderId != ctx.orderId ||
        line == null ||
        draft.quantity < 1 ||
        draft.quantity > line.maximum ||
        !returnReasons.containsKey(draft.reason) ||
        draft.details.length < 10 ||
        draft.details.length > 1000 ||
        draft.evidenceUrls.isEmpty ||
        draft.evidenceUrls.length > 5 ||
        !mediaValid ||
        draft.bank.isEmpty ||
        !RegExp(r'^\d{6,30}$').hasMatch(draft.account) ||
        draft.holder.length < 2) {
      emit(
        ReturnsState(
          context: ctx,
          error:
              'Kiểm tra sản phẩm/số lượng, lý do, mô tả 10–1000 ký tự, 1–5 URL HTTPS và thông tin ngân hàng hợp lệ.',
        ),
      );
      return;
    }
    final fingerprint = jsonEncode([
      draft.orderId,
      draft.lineId,
      draft.quantity,
      draft.reason,
      draft.details,
      draft.evidenceUrls,
      draft.bank,
      draft.account,
      draft.holder,
    ]);
    if (_fingerprint != fingerprint) {
      _fingerprint = fingerprint;
      _key = const Uuid().v4();
    }
    emit(ReturnsState(context: ctx, pending: true));
    try {
      final value = await _repository.create(draft, idempotencyKey: _key!);
      if (!isClosed) emit(ReturnsState(context: ctx, created: value));
    } catch (e) {
      _fail(e);
    }
  }

  @override
  Future<void> close() {
    _fingerprint = null;
    _key = null;
    return super.close();
  }
}
