import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/trace/domain/trace_repository.dart';

class TraceState {
  const TraceState({
    this.loading = false,
    this.code = '',
    this.record,
    this.error,
  });
  final bool loading;
  final String code;
  final TraceRecord? record;
  final String? error;
}

class TraceCubit extends Cubit<TraceState> {
  TraceCubit(this._repository) : super(const TraceState());
  final TraceRepository _repository;
  int _revision = 0;
  Future<void> lookup(String input) async {
    final revision = ++_revision, code = input.trim();
    if (code.isEmpty || code.length > 160) {
      emit(
        TraceState(code: code, error: 'Nhập mã trên bao bì, tối đa 160 ký tự.'),
      );
      return;
    }
    emit(TraceState(loading: true, code: code));
    try {
      final record = await _repository.lookup(code);
      if (!isClosed && revision == _revision) {
        emit(TraceState(code: code, record: record));
      }
    } catch (e) {
      if (!isClosed && revision == _revision) {
        emit(
          TraceState(
            code: code,
            error: e is ApiException
                ? e.failure.message
                : 'Chưa thể tải hồ sơ nguồn gốc. Vui lòng thử lại.',
          ),
        );
      }
    }
  }
}
