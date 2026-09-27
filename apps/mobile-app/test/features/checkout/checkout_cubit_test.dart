import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/features/cart/data/in_memory_cart_repository.dart';
import 'package:mobile_app/features/account/data/in_memory_address_repository.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';
import 'package:mobile_app/features/catalog/data/mock_catalog_repository.dart';
import 'package:mobile_app/features/checkout/data/mock_checkout_repository.dart';
import 'package:mobile_app/features/checkout/presentation/checkout_cubit.dart';

Future<CheckoutCubit> createCubit(MockCheckoutRepository checkout) async {
  final product = await MockCatalogRepository().product('PROD-MX-GION-500');
  final cartRepository = InMemoryCartRepository();
  final cart = await cartRepository.add(product, product.variants.first, 1);
  return CheckoutCubit(checkout, cart, idempotencyKey: 'stable-key');
}

void main() {
  test('validation prevents request and maps fields', () async {
    final repository = MockCheckoutRepository();
    final cubit = await createCubit(repository);
    await cubit.submit();
    expect(
      cubit.state.fieldErrors.keys,
      containsAll([
        'fullName',
        'phone',
        'streetAddress',
        'ward',
        'district',
        'province',
      ]),
    );
    expect(repository.submitCount, 0);
    await cubit.close();
  });
  test('success uses one request and duplicate tap is prevented', () async {
    final repository = MockCheckoutRepository();
    final cubit = await createCubit(repository);
    cubit.setName('Nguyễn Văn An');
    cubit.setPhone('0905123456');
    cubit.setStreetAddress('123 Lê Duẩn');
    cubit.setWard('Phường Thuận Hòa');
    cubit.setDistrict('Thành phố Huế');
    cubit.setProvince('Thừa Thiên Huế');
    final first = cubit.submit();
    final duplicate = cubit.submit();
    await Future.wait([first, duplicate]);
    expect(repository.submitCount, 1);
    expect(cubit.state.status, CheckoutStatus.success);
    expect(cubit.state.confirmation, isNotNull);
    await cubit.close();
  });

  test(
    'saved address prefills fields and remains editable when unchecked',
    () async {
      final cubit = await createCubit(MockCheckoutRepository());
      await cubit.loadSavedAddress(InMemoryAddressRepository());

      expect(cubit.state.savedAddress, isNotNull);
      expect(cubit.state.useAccountAddress, isFalse);

      cubit.useSavedAddress(true);
      expect(cubit.state.useAccountAddress, isTrue);
      expect(cubit.state.fullName, 'Nguyễn Văn An');
      expect(cubit.state.streetAddress, '12 Đường Mẫu');
      expect(cubit.state.ward, 'Phường Mẫu');

      cubit.useSavedAddress(false);
      cubit.setStreetAddress('45 Đường Mới');
      expect(cubit.state.useAccountAddress, isFalse);
      expect(cubit.state.streetAddress, '45 Đường Mới');
      await cubit.close();
    },
  );

  test(
    'missing saved address keeps account-address option unavailable',
    () async {
      final cubit = await createCubit(MockCheckoutRepository());
      await cubit.loadSavedAddress(_EmptyAddressRepository());

      expect(cubit.state.checkedSavedAddresses, isTrue);
      expect(cubit.state.savedAddress, isNull);
      expect(cubit.state.useAccountAddress, isFalse);
      await cubit.close();
    },
  );
}

class _EmptyAddressRepository implements AddressRepository {
  @override
  Future<List<Address>> addresses() async => const [];

  @override
  Future<void> delete(String id) async {}

  @override
  Future<void> save(Address address) async {}

  @override
  Future<void> setDefault(String id) async {}
}
