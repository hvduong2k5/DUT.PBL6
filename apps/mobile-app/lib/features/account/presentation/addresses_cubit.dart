import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';

enum AddressesStatus { initial, loading, success, failure }

class AddressesState extends Equatable {
  const AddressesState({
    this.status = AddressesStatus.initial,
    this.addresses = const [],
  });
  final AddressesStatus status;
  final List<Address> addresses;
  @override
  List<Object?> get props => [status, addresses];
}

class AddressesCubit extends Cubit<AddressesState> {
  AddressesCubit(this._repository) : super(const AddressesState());
  final AddressRepository _repository;

  Future<void> load() async {
    emit(
      AddressesState(
        status: AddressesStatus.loading,
        addresses: state.addresses,
      ),
    );
    try {
      emit(
        AddressesState(
          status: AddressesStatus.success,
          addresses: await _repository.addresses(),
        ),
      );
    } catch (_) {
      emit(
        AddressesState(
          status: AddressesStatus.failure,
          addresses: state.addresses,
        ),
      );
    }
  }

  Future<void> delete(String id) async {
    await _repository.delete(id);
    await load();
  }

  Future<void> setDefault(String id) async {
    await _repository.setDefault(id);
    await load();
  }
}
