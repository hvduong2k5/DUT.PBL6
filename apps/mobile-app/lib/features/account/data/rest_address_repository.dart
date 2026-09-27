import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';

class RestAddressRepository implements AddressRepository {
  const RestAddressRepository(this._client);
  final ApiClient _client;

  @override
  Future<List<Address>> addresses() async {
    final json = await _client.getJson('/api/v1/profile/addresses');
    return (json['addresses'] as List? ?? const [])
        .map(
          (item) => Address.fromApiJson(Map<String, dynamic>.from(item as Map)),
        )
        .toList();
  }

  @override
  Future<void> save(Address address) async {
    final current = await addresses();
    final exists = current.any((item) => item.id == address.id);
    if (!exists) {
      await _client.postJson(
        '/api/v1/profile/addresses',
        data: address.toApiJson(),
      );
    } else {
      await _client.putJson(
        '/api/v1/profile/addresses/${address.id}',
        data: address.toApiJson(),
      );
    }
  }

  @override
  Future<void> delete(String id) async {
    await _client.deleteJson('/api/v1/profile/addresses/$id');
  }

  @override
  Future<void> setDefault(String id) async {
    final current = await addresses();
    final selected = current.where((item) => item.id == id).firstOrNull;
    if (selected == null) return;
    await _client.putJson(
      '/api/v1/profile/addresses/$id',
      data: selected.copyWith(isDefault: true).toApiJson(),
    );
  }
}
