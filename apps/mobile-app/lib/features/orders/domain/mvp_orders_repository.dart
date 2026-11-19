import 'package:mobile_app/features/orders/domain/mvp_order_models.dart';

abstract interface class MvpOrdersRepository {
  Future<MvpOrderPageData> firstPage();
  Future<MvpOrder> order(String id);
  Future<MvpPaymentStatus> payment(String id);
  Future<MvpPaymentInstructions> instructions(String id);
  Future<MvpCancellation> cancel(MvpCancellationCommand command);
}

abstract interface class MvpCancellationStore {
  Future<MvpCancellationCommand?> read(
    String owner,
    String scope,
    String orderId,
  );
  Future<void> write(MvpCancellationCommand command);
}

class MvpOrdersRuntime {
  const MvpOrdersRuntime({
    required this.repository,
    required this.cancellations,
    required this.scope,
    this.local = false,
  });
  final MvpOrdersRepository repository;
  final MvpCancellationStore cancellations;
  final String scope;
  final bool local;
}
