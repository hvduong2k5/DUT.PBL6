import 'dart:convert';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/support/domain/support_repository.dart';

class SupportState {
  const SupportState({
    this.loading = false,
    this.pending = false,
    this.context,
    this.ticket,
    this.createdId,
    this.error,
  });
  final bool loading, pending;
  final SupportContext? context;
  final SupportTicket? ticket;
  final String? createdId, error;
}

class SupportCubit extends Cubit<SupportState> {
  SupportCubit(this._repository) : super(const SupportState());
  final SupportRepository _repository;
  String? _payload, _key;
  String _actionKey(List<String> values) {
    final payload = jsonEncode(values);
    if (_payload != payload) {
      _payload = payload;
      _key = const Uuid().v4();
    }
    return _key!;
  }

  void _failure(Object error) {
    if (!isClosed) {
      emit(
        SupportState(
          context: state.context,
          ticket: state.ticket,
          error: error is ApiException
              ? error.failure.message
              : 'Chưa thể kết nối Ban Tri Kỷ. Vui lòng thử lại.',
        ),
      );
    }
  }

  Future<void> loadContext() async {
    if (state.loading || state.pending) return;
    emit(const SupportState(loading: true));
    try {
      final value = await _repository.context();
      if (!isClosed) emit(SupportState(context: value));
    } catch (e) {
      _failure(e);
    }
  }

  Future<void> loadTicket(String id) async {
    if (state.loading || state.pending) return;
    emit(const SupportState(loading: true));
    try {
      final value = await _repository.ticket(id);
      if (!isClosed) emit(SupportState(ticket: value));
    } catch (e) {
      _failure(e);
    }
  }

  Future<void> create({
    required String subject,
    required String title,
    required String message,
    required String priority,
    required String orderId,
  }) async {
    final ctx = state.context;
    if (state.pending || ctx == null) return;
    final t = title.trim(), m = message.trim();
    if (t.length < 5 ||
        t.length > 160 ||
        m.length < 10 ||
        m.length > 3000 ||
        !ctx.subjects.any((o) => o.code == subject) ||
        !ctx.priorities.any((o) => o.code == priority) ||
        (orderId.isNotEmpty && !ctx.orders.any((o) => o.id == orderId))) {
      emit(
        SupportState(
          context: ctx,
          error:
              'Chọn chủ đề, ưu tiên và đơn hợp lệ. Tiêu đề 5–160 ký tự, nội dung 10–3000 ký tự.',
        ),
      );
      return;
    }
    emit(SupportState(context: ctx, pending: true));
    try {
      final id = await _repository.create(
        subject: subject,
        title: t,
        message: m,
        priority: priority,
        orderId: orderId,
        idempotencyKey: _actionKey([
          'create',
          subject,
          t,
          m,
          priority,
          orderId,
        ]),
      );
      if (!isClosed) emit(SupportState(context: ctx, createdId: id));
    } catch (e) {
      _failure(e);
    }
  }

  Future<bool> reply(String message) async {
    final ticket = state.ticket, m = message.trim();
    if (ticket == null || !ticket.canReply || state.pending) return false;
    if (m.length < 10 || m.length > 3000) {
      emit(SupportState(ticket: ticket, error: 'Phản hồi cần 10–3000 ký tự.'));
      return false;
    }
    emit(SupportState(ticket: ticket, pending: true));
    try {
      final result = await _repository.reply(
        ticket.id,
        m,
        idempotencyKey: _actionKey(['reply', ticket.id, m]),
      );
      _payload = null;
      _key = null;
      if (!isClosed) emit(SupportState(ticket: ticket.withReply(result)));
      return true;
    } catch (e) {
      _failure(e);
      return false;
    }
  }
}
