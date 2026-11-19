import 'dart:convert';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/b2b/domain/b2b_repository.dart';

String _message(Object error) => error is ApiException
    ? error.failure.message
    : 'Chưa thể kết nối không gian B2B. Vui lòng thử lại.';

class B2bRequestState {
  const B2bRequestState({
    this.loading = false,
    this.pending = false,
    this.context,
    this.receipt,
    this.error,
  });
  final bool loading, pending;
  final B2bContext? context;
  final B2bRequestReceipt? receipt;
  final String? error;
}

class B2bRequestCubit extends Cubit<B2bRequestState> {
  B2bRequestCubit(this._repository) : super(const B2bRequestState());
  final B2bRepository _repository;
  String? _fingerprint, _key;
  Future<void> load() async {
    if (state.loading || state.pending) return;
    emit(const B2bRequestState(loading: true));
    try {
      final value = await _repository.context();
      if (!isClosed) emit(B2bRequestState(context: value));
    } catch (e) {
      if (!isClosed) emit(B2bRequestState(error: _message(e)));
    }
  }

  void newRequest() {
    if (state.pending) return;
    _fingerprint = null;
    _key = null;
    emit(B2bRequestState(context: state.context));
  }

  Future<void> submit(B2bRequestDraft draft) async {
    final ctx = state.context;
    if (ctx == null || state.pending || state.receipt != null) return;
    final now = DateTime.now(), date = DateTime.tryParse(draft.date);
    final minimum = DateTime(
      now.year,
      now.month,
      now.day,
    ).add(const Duration(days: 7));
    final province = ctx.provinces
        .where(
          (p) => p.code == draft.provinceCode && p.name == draft.provinceName,
        )
        .firstOrNull;
    final invalidQuantity = draft.quantities.entries.any(
      (e) =>
          e.value < 0 ||
          !ctx.products.any((p) => p.sku == e.key) ||
          (e.value > 0 &&
              !ctx.products.any((p) => p.sku == e.key && e.value >= p.minimum)),
    );
    if (ctx.organization.verification != 'VERIFIED' ||
        draft.organizationId != ctx.organization.id ||
        !b2bPurposes.containsKey(draft.purpose) ||
        province == null ||
        date == null ||
        date.isBefore(minimum) ||
        date.toIso8601String().substring(0, 10) != draft.date ||
        invalidQuantity ||
        !draft.quantities.values.any((q) => q > 0)) {
      emit(
        B2bRequestState(
          context: ctx,
          error:
              'Kiểm tra tổ chức đã xác minh, mục đích, tỉnh/thành, ngày nhận từ 7 ngày tới và số lượng tối thiểu từng sản phẩm.',
        ),
      );
      return;
    }
    final fingerprint = jsonEncode([
      draft.organizationId,
      draft.purpose,
      draft.date,
      draft.provinceCode,
      draft.provinceName,
      draft.quantities,
      draft.branding,
      draft.invoice,
      draft.sample,
    ]);
    if (_fingerprint != fingerprint) {
      _fingerprint = fingerprint;
      _key = const Uuid().v4();
    }
    emit(B2bRequestState(context: ctx, pending: true));
    try {
      final receipt = await _repository.request(draft, idempotencyKey: _key!);
      if (!isClosed) emit(B2bRequestState(context: ctx, receipt: receipt));
    } catch (e) {
      if (!isClosed) emit(B2bRequestState(context: ctx, error: _message(e)));
    }
  }
}

class B2bQuoteState {
  const B2bQuoteState({
    this.loading = false,
    this.pending = false,
    this.confirmed = false,
    this.retryingAcceptance = false,
    this.quote,
    this.accepted,
    this.error,
  });
  final bool loading, pending, confirmed, retryingAcceptance;
  final B2bQuote? quote;
  final B2bAcceptance? accepted;
  final String? error;
}

class B2bQuoteCubit extends Cubit<B2bQuoteState> {
  B2bQuoteCubit(this._repository, {required this.mayAccept})
    : super(const B2bQuoteState());
  final B2bRepository _repository;
  final bool Function() mayAccept;
  String? _key;
  bool _postAttempted = false;
  Future<void> load(String id) async {
    if (state.loading || state.pending) return;
    emit(const B2bQuoteState(loading: true));
    try {
      final quote = await _repository.quote(id);
      if (quote.id != id) {
        throw const FormatException('Quote identifier mismatch');
      }
      _key = null;
      _postAttempted = false;
      if (!isClosed) emit(B2bQuoteState(quote: quote));
    } catch (e) {
      if (!isClosed) emit(B2bQuoteState(error: _message(e)));
    }
  }

  void agree(bool value) {
    if (!state.pending && state.accepted == null) {
      emit(
        B2bQuoteState(
          quote: state.quote,
          confirmed: value,
          retryingAcceptance: _postAttempted,
        ),
      );
    }
  }

  Future<void> accept() async {
    final quote = state.quote;
    if (quote == null || state.pending || state.accepted != null) return;
    if (!mayAccept()) {
      emit(
        B2bQuoteState(
          quote: quote,
          error: 'Bạn chưa được cấp quyền chấp thuận báo giá.',
        ),
      );
      return;
    }
    if (!state.confirmed ||
        ((!quote.canAccept || quote.expired) && !_postAttempted)) {
      emit(
        B2bQuoteState(
          quote: quote,
          error: quote.expired
              ? 'Báo giá đã hết hạn. Vui lòng đề nghị báo giá mới.'
              : 'Hãy đọc và đồng ý báo giá còn hiệu lực trước khi chấp thuận.',
        ),
      );
      return;
    }
    emit(
      B2bQuoteState(
        quote: quote,
        pending: true,
        confirmed: true,
        retryingAcceptance: _postAttempted,
      ),
    );
    try {
      final latest = await _repository.quote(quote.id);
      if (isClosed) return;
      if (!mayAccept()) {
        emit(
          B2bQuoteState(
            quote: latest,
            error: 'Quyền chấp thuận đã thay đổi. Vui lòng kiểm tra tài khoản.',
          ),
        );
        return;
      }
      if (latest.agreementFingerprint != quote.agreementFingerprint) {
        _key = null;
        _postAttempted = false;
        if (!isClosed) {
          emit(
            B2bQuoteState(
              quote: latest,
              error:
                  'Báo giá đã cập nhật. Vui lòng đọc lại nội dung và xác nhận phiên bản mới.',
            ),
          );
        }
        return;
      }
      if ((!latest.canAccept || latest.expired) && !_postAttempted) {
        if (!isClosed) {
          emit(
            B2bQuoteState(
              quote: latest,
              error:
                  'Báo giá không còn có thể chấp thuận. Vui lòng cập nhật hoặc liên hệ hỗ trợ.',
            ),
          );
        }
        return;
      }
      // After an uncertain POST, replay the same key to recover its existing result.
      // Server must enforce atomic released-version eligibility; ZIP has no version field.
      _key ??= const Uuid().v4();
      _postAttempted = true;
      final result = await _repository.accept(quote.id, idempotencyKey: _key!);
      if (!isClosed) emit(B2bQuoteState(quote: latest, accepted: result));
    } catch (e) {
      if (e is ApiException &&
          !const {
            FailureKind.network,
            FailureKind.timeout,
            FailureKind.server,
            FailureKind.unknown,
          }.contains(e.failure.kind)) {
        _postAttempted = false;
      }
      if (!isClosed) {
        emit(
          B2bQuoteState(
            quote: quote,
            confirmed: _postAttempted,
            retryingAcceptance: _postAttempted,
            error: _message(e),
          ),
        );
      }
    }
  }
}
