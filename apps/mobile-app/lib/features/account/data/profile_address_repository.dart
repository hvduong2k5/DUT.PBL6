import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';

/// Profile owns address UUIDs, administrative codes and CAS versions.
class ProfileAddressRepository implements AddressRepository {
  ProfileAddressRepository(this.client);
  final ApiClient client;
  final _known = <String, int>{};
  @override
  Future<List<Address>> addresses() async {
    final rows = await client.getJsonList('/api/v1/profile/addresses');
    final addresses = <Address>[];
    _known.clear();
    for (final row in rows) {
      final id = row['id'] as String;
      _known[id] = row['version'] as int;
      addresses.add(
        Address(
          id: id,
          recipientName: row['recipient_name'] as String,
          phoneNumber: row['phone_number'] as String,
          streetAddress: row['street_address'] as String,
          ward: row['ward_name'] as String,
          province: row['province_name'] as String,
          wardCode: row['ward_code'] as String,
          provinceCode: row['province_code'] as String,
          version: row['version'] as int,
          isDefault: row['is_default'] == true,
        ),
      );
    }
    return addresses;
  }

  @override
  Future<void> save(Address address) async {
    if (address.wardCode == null || address.provinceCode == null) {
      throw const FormatException(
        'Chọn tỉnh/phường trong danh sách giao hàng.',
      );
    }
    final body = <String, dynamic>{
      'recipient_name': address.recipientName,
      'phone_number': address.phoneNumber,
      'street_address': address.streetAddress,
      'ward_code': address.wardCode,
      'province_code': address.provinceCode,
      'ward_name': address.ward,
      'province_name': address.province,
      'label': 'HOME',
    };
    final version = address.version ?? _known[address.id];
    if (version == null) {
      await client.postJson(
        '/api/v1/profile/addresses',
        data: {...body, 'is_default': address.isDefault},
      );
    } else {
      await client.putJson(
        '/api/v1/profile/addresses/${address.id}',
        data: body,
        headers: {'If-Match': '"$version"'},
      );
    }
    final saved = await addresses();
    if (version != null &&
        address.isDefault &&
        !saved.firstWhere((item) => item.id == address.id).isDefault) {
      await setDefault(address.id);
    }
  }

  Future<int> _version(String id) async {
    if (!_known.containsKey(id)) await addresses();
    final version = _known[id];
    if (version == null) {
      throw const FormatException('Địa chỉ không còn tồn tại.');
    }
    return version;
  }

  @override
  Future<void> delete(String id) async {
    final version = await _version(id);
    await client.deleteVoid(
      '/api/v1/profile/addresses/$id',
      headers: {'If-Match': '"$version"'},
    );
    await addresses();
  }

  @override
  Future<void> setDefault(String id) async {
    final version = await _version(id);
    await client.putVoid(
      '/api/v1/profile/addresses/$id/default',
      headers: {'If-Match': '"$version"'},
    );
    await addresses();
  }
}
