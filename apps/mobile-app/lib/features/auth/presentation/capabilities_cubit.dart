import 'dart:async';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/domain/customer_capabilities.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';

class CapabilitiesState {
  const CapabilitiesState({this.loading = false, this.projection, this.error});
  final bool loading;
  final CustomerCapabilities? projection;
  final String? error;
  bool allows(String capability) =>
      !loading && (projection?.allows(capability) ?? false);
}

class CapabilitiesCubit extends Cubit<CapabilitiesState> {
  CapabilitiesCubit(this._repository, this._auth)
    : super(const CapabilitiesState(loading: true)) {
    _subscription = _auth.stream.listen((state) {
      if (state.status == AuthStatus.restoring ||
          state.status == AuthStatus.submitting) {
        ++_revision;
        emit(const CapabilitiesState(loading: true));
      } else {
        load();
      }
    });
    if (_auth.state.status != AuthStatus.restoring &&
        _auth.state.status != AuthStatus.submitting) {
      load();
    }
  }
  final CustomerCapabilitiesRepository _repository;
  final AuthCubit _auth;
  late final StreamSubscription<AuthState> _subscription;
  int _revision = 0;
  Future<void> load() async {
    final revision = ++_revision;
    emit(const CapabilitiesState(loading: true));
    try {
      final projection = await _repository.load();
      if (!isClosed && revision == _revision) {
        emit(CapabilitiesState(projection: projection));
      }
    } catch (e) {
      if (!isClosed && revision == _revision) {
        emit(
          CapabilitiesState(
            error: e is ApiException
                ? e.failure.message
                : 'Chưa thể tải quyền truy cập. Vui lòng thử lại.',
          ),
        );
      }
    }
  }

  @override
  Future<void> close() async {
    await _subscription.cancel();
    await super.close();
  }
}
