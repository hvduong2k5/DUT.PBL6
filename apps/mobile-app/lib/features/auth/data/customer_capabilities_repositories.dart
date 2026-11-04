import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/customer_capabilities.dart';
import 'package:mobile_app/features/auth/data/mock_auth_repository.dart';

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
    final verifiedB2b = user?.id == MockAuthRepository.b2bFixtureId;
    return CustomerCapabilities(
      actor: user == null
          ? 'GUEST'
          : verifiedB2b
          ? 'B2B'
          : 'REGISTERED',
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
        if (verifiedB2b) ...{
          'B2B_COMPANY_VIEW',
          'B2B_QUOTE_CREATE',
          'B2B_QUOTE_VIEW',
          'B2B_QUOTE_ACCEPT',
        },
      },
    );
  }
}
