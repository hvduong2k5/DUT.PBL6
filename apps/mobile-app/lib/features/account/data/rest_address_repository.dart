import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';
import 'dart:convert';
import 'package:uuid/uuid.dart';

class RestAddressRepository implements AddressRepository {
  RestAddressRepository(this._client);
  final ApiClient _client;
  final Map<String, String> _createKeys = {};

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
      final payload = address.toApiJson();
      final fingerprint = jsonEncode(payload);
      await _client.postJson(
        '/api/v1/profile/addresses',
        data: payload,
        headers: {
          'Idempotency-Key': _createKeys.putIfAbsent(
            fingerprint,
            () => const Uuid().v4(),
          ),
        },
      );
      _createKeys.remove(fingerprint);
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
