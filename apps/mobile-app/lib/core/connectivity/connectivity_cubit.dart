import 'dart:async';
import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

class ConnectivityCubit extends Cubit<bool> {
  ConnectivityCubit({Connectivity? connectivity})
    : _connectivity = connectivity ?? Connectivity(),
      super(true) {
    _subscription = _connectivity.onConnectivityChanged.listen(_onChanged);
    _connectivity.checkConnectivity().then(_onChanged);
  }
  final Connectivity _connectivity;
  StreamSubscription<List<ConnectivityResult>>? _subscription;
  void _onChanged(List<ConnectivityResult> results) =>
      emit(!results.contains(ConnectivityResult.none));
  @override
  Future<void> close() async {
    await _subscription?.cancel();
    return super.close();
  }
}
