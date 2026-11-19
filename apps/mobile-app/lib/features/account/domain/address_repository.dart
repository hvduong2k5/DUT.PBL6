import 'package:mobile_app/features/account/domain/address.dart';

abstract interface class AddressRepository {
  Future<List<Address>> addresses();
  Future<void> save(Address address);
  Future<void> delete(String id);
  Future<void> setDefault(String id);
}
