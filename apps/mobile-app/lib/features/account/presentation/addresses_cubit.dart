import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';
import 'package:mobile_app/core/network/api_exception.dart';

enum AddressesStatus { initial, loading, success, failure }

class AddressesState extends Equatable {
  const AddressesState({
    this.status = AddressesStatus.initial,
    this.addresses = const [],
    this.message,
    this.pending = false,
  });
  final AddressesStatus status;
  final List<Address> addresses;
  final String? message;
  final bool pending;
  @override
  List<Object?> get props => [status, addresses, message, pending];
}

class AddressesCubit extends Cubit<AddressesState> {
  AddressesCubit(this._repository) : super(const AddressesState());
  final AddressRepository _repository;

  Future<void> load() async {
    emit(
      AddressesState(
        status: AddressesStatus.loading,
        addresses: state.addresses,
        pending: true,
      ),
    );
    try {
      final addresses = await _repository.addresses();
      if (isClosed) return;
      emit(
        AddressesState(status: AddressesStatus.success, addresses: addresses),
      );
    } catch (error) {
      if (isClosed) return;
      emit(
        AddressesState(
          status: AddressesStatus.failure,
          addresses: state.addresses,
          message: error is ApiException
              ? error.failure.message
              : 'Chưa tải được địa chỉ. Vui lòng thử lại.',
        ),
      );
    }
  }

  Future<void> delete(String id) => _mutate(() => _repository.delete(id));

  Future<void> setDefault(String id) =>
      _mutate(() => _repository.setDefault(id));
  Future<void> _mutate(Future<void> Function() operation) async {
    if (state.pending) return;
    emit(
      AddressesState(
        status: state.status,
        addresses: state.addresses,
        pending: true,
      ),
    );
    try {
      await operation();
      if (!isClosed) await load();
    } catch (error) {
      if (!isClosed) {
        emit(
          AddressesState(
            status: state.status,
            addresses: state.addresses,
            message: error is ApiException
                ? error.failure.message
                : 'Chưa cập nhật được địa chỉ. Vui lòng thử lại.',
          ),
        );
      }
    }
  }
}
