import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/orders/domain/mvp_order_models.dart';
import 'package:mobile_app/features/orders/domain/mvp_orders_repository.dart';

class MvpOrdersState {
  const MvpOrdersState({this.loading = true, this.page, this.message});
  final bool loading;
  final MvpOrderPageData? page;
  final String? message;
}

class MvpOrdersCubit extends Cubit<MvpOrdersState> {
  MvpOrdersCubit(this.repository, this.principal)
    : super(const MvpOrdersState());
  final MvpOrdersRepository repository;
  final String? Function() principal;
  int _generation = 0;
  Future<void> load() async {
    if (isClosed) return;
    final generation = ++_generation, owner = principal();
    emit(const MvpOrdersState());
    try {
      if (owner == null) throw StateError('Unauthenticated');
      final page = await repository.firstPage();
      if (!isClosed && generation == _generation && principal() == owner) {
        emit(MvpOrdersState(loading: false, page: page));
      }
    } catch (_) {
      if (!isClosed && generation == _generation && principal() == owner) {
        emit(
          const MvpOrdersState(
            loading: false,
            message: 'Chưa thể tải danh sách đơn. Vui lòng thử lại.',
          ),
        );
      }
    }
  }
}
