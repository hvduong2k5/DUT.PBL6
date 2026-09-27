import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/features/account/data/in_memory_address_repository.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/account/presentation/addresses_cubit.dart';

void main() {
  test('address payload follows the OpenAPI field names', () {
    const address = Address(
      id: '1',
      recipientName: 'An',
      phoneNumber: '0901234567',
      streetAddress: '12 Đường Mẫu',
      ward: 'Phường Mẫu',
      district: 'Quận Mẫu',
      province: 'Huế',
      isDefault: true,
    );
    expect(
      address.toApiJson().keys,
      containsAll([
        'recipient_name',
        'phone_number',
        'street_address',
        'ward',
        'district',
        'province',
        'is_default',
      ]),
    );
  });

  test('saving a new default address clears the old default', () async {
    final repository = InMemoryAddressRepository();
    const address = Address(
      id: '2',
      recipientName: 'Bình',
      phoneNumber: '0912345678',
      streetAddress: '34 Đường Mới',
      ward: 'Phường Hai',
      district: 'Quận Hai',
      province: 'Đà Nẵng',
      isDefault: true,
    );
    await repository.save(address);
    final values = await repository.addresses();
    expect(values.where((item) => item.isDefault), [address]);
  });

  test('AddressesCubit reloads after delete', () async {
    final cubit = AddressesCubit(InMemoryAddressRepository());
    await cubit.load();
    await cubit.delete(cubit.state.addresses.single.id);
    expect(cubit.state.status, AddressesStatus.success);
    expect(cubit.state.addresses, isEmpty);
    await cubit.close();
  });
}
