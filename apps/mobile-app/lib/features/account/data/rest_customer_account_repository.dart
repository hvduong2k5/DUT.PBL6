import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/account/domain/customer_account_repository.dart';

class RestCustomerAccountRepository implements CustomerAccountRepository {
  const RestCustomerAccountRepository(this._client);
  final ApiClient _client;
  @override
  Future<CustomerProfile> profile() async =>
      CustomerProfile.fromJson(await _client.getJson('/api/v1/profile'));
  @override
  Future<CustomerProfile> updateProfile(String fullName, String email) async =>
      CustomerProfile.fromJson(
        await _client.putJson(
          '/api/v1/profile',
          data: {'full_name': fullName, 'email': email},
        ),
      );
  @override
  Future<LoyaltyOverview> loyalty() async =>
      LoyaltyOverview.fromJson(await _client.getJson('/api/v1/loyalty/points'));
}
