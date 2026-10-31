import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';

abstract interface class MvpCheckoutRepository {
  Future<MvpCheckoutOperation> accept(
    MvpCheckoutInput input, {
    required String key,
  });
  Future<MvpCheckoutOperation> operation(String id);
}

abstract interface class MvpCheckoutStore {
  Future<MvpCheckoutAttempt?> read(String owner, String scope);
  Future<void> write(MvpCheckoutAttempt attempt);
}

class MvpCheckoutRuntime {
  const MvpCheckoutRuntime({
    required this.repository,
    required this.store,
    required this.scope,
    this.local = false,
  });
  final MvpCheckoutRepository repository;
  final MvpCheckoutStore store;
  final String scope;
  final bool local;
}
