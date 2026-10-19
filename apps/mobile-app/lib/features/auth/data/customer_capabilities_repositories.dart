import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/customer_capabilities.dart';

class RestCustomerCapabilitiesRepository
    implements CustomerCapabilitiesRepository {
  const RestCustomerCapabilitiesRepository(this._client);
  final ApiClient _client;
  @override
  Future<CustomerCapabilities> load() async => CustomerCapabilities.fromJson(
    await _client.getJson('/api/v1/customer/capabilities'),
  );
}

class MockCustomerCapabilitiesRepository
    implements CustomerCapabilitiesRepository {
  const MockCustomerCapabilitiesRepository(this.currentUser);
  final AuthUser? Function() currentUser;
  @override
  Future<CustomerCapabilities> load() async {
    final user = currentUser();
    return CustomerCapabilities(
      actor: user == null ? 'GUEST' : 'REGISTERED',
      version: 1,
      values: {
        'CATALOG_VIEW',
        'CART_MANAGE',
        'CHECKOUT_CREATE',
        'GUEST_ORDER_TRACK',
        'SUPPORT_CREATE',
        if (user != null) ...{
          'PROFILE_MANAGE',
          'ADDRESS_MANAGE',
          'ORDER_HISTORY_VIEW',
          'LOYALTY_VIEW',
          'REVIEW_CREATE',
        },
      },
    );
  }
}
