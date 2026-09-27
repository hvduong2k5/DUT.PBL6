import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';

class InMemoryAddressRepository implements AddressRepository {
  final List<Address> _items = [
    const Address(
      id: 'address-1',
      recipientName: 'Nguyễn Văn An',
      phoneNumber: '0901234567',
      streetAddress: '12 Đường Mẫu',
      ward: 'Phường Mẫu',
      district: 'Quận Mẫu',
      province: 'Huế',
      isDefault: true,
    ),
  ];

  @override
  Future<List<Address>> addresses() async => List.unmodifiable(_items);

  @override
  Future<void> delete(String id) async {
    _items.removeWhere((item) => item.id == id);
  }

  @override
  Future<void> save(Address address) async {
    if (address.isDefault) {
      for (var index = 0; index < _items.length; index++) {
        _items[index] = _items[index].copyWith(isDefault: false);
      }
    }
    final index = _items.indexWhere((item) => item.id == address.id);
    if (index < 0) {
      _items.add(address);
    } else {
      _items[index] = address;
    }
  }

  @override
  Future<void> setDefault(String id) async {
    for (var index = 0; index < _items.length; index++) {
      _items[index] = _items[index].copyWith(isDefault: _items[index].id == id);
    }
  }
}
