import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/checkout/domain/checkout_location.dart';
import 'package:mobile_app/features/checkout/domain/checkout_repository.dart';

class AddressFormPage extends StatefulWidget {
  const AddressFormPage({super.key, this.address});
  final Address? address;

  @override
  State<AddressFormPage> createState() => _AddressFormPageState();
}

class _AddressFormPageState extends State<AddressFormPage> {
  late final TextEditingController _name;
  late final TextEditingController _phone;
  late final TextEditingController _province;
  late final TextEditingController _ward;
  late final TextEditingController _street;
  late bool _isDefault;
  bool _saving = false;
  bool _submitted = false;
  late final String _newId = const Uuid().v4();
  bool _loadingLocations = true;
  List<CheckoutProvince> _locations = [];
  String? _locationsError;
  String? _message;
  Map<String, String> _fieldErrors = {};

  @override
  void initState() {
    super.initState();
    final value = widget.address;
    _name = TextEditingController(text: value?.recipientName);
    _phone = TextEditingController(text: value?.phoneNumber);
    _province = TextEditingController(text: value?.province);
    _ward = TextEditingController(text: value?.ward);
    _street = TextEditingController(text: value?.streetAddress);
    _isDefault = value?.isDefault ?? false;
    _loadLocations();
  }

  Future<void> _loadLocations() async {
    setState(() {
      _loadingLocations = true;
      _locationsError = null;
    });
    try {
      final values = await context.read<CheckoutRepository>().locations();
      if (mounted) {
        setState(() {
          _locations = values;
          _loadingLocations = false;
          _locationsError = values.isEmpty
              ? 'Chưa có khu vực giao hàng khả dụng.'
              : null;
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() {
          _loadingLocations = false;
          _locationsError = error is ApiException
              ? error.failure.message
              : 'Chưa tải được khu vực giao hàng.';
        });
      }
    }
  }

  String? _required(String value) =>
      _submitted && value.trim().isEmpty ? 'Bắt buộc' : null;
  String? get _phoneError {
    if (!_submitted) return null;
    if (_phone.text.trim().isEmpty) return 'Bắt buộc';
    return RegExp(r'^0\d{9}$').hasMatch(_phone.text.trim())
        ? null
        : 'Không hợp lệ';
  }

  bool get _valid =>
      [
        _name,
        _province,
        _ward,
        _street,
      ].every((field) => field.text.trim().isNotEmpty) &&
      _phoneError == null;

  Future<void> _save() async {
    if (_saving) return;
    setState(() => _submitted = true);
    final province = _locations
        .where((item) => item.name == _province.text)
        .firstOrNull;
    if (!_valid ||
        province == null ||
        !province.wards.any((ward) => ward.name == _ward.text)) {
      return;
    }
    setState(() {
      _saving = true;
      _message = null;
      _fieldErrors = {};
    });
    final address = Address(
      id: widget.address?.id ?? _newId,
      recipientName: _name.text.trim(),
      phoneNumber: _phone.text.trim(),
      streetAddress: _street.text.trim(),
      ward: _ward.text.trim(),
      province: _province.text.trim(),
      isDefault: _isDefault,
    );
    try {
      await context.read<AddressRepository>().save(address);
      if (mounted) {
        if (context.canPop()) {
          context.pop();
        } else {
          context.go(AppRoutes.addresses);
        }
      }
    } catch (error) {
      if (mounted) {
        setState(() {
          _saving = false;
          _message = error is ApiException
              ? error.failure.message
              : 'Chưa thể lưu địa chỉ. Vui lòng thử lại.';
          _fieldErrors = error is ApiException ? error.failure.fieldErrors : {};
        });
      }
    }
  }

  @override
  void dispose() {
    for (final controller in [_name, _phone, _province, _ward, _street]) {
      controller.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final province = _locations
        .where((item) => item.name == _province.text)
        .firstOrNull;
    return Scaffold(
      appBar: AppMobileHeader(
        title: widget.address == null ? 'Thêm địa chỉ' : 'Sửa địa chỉ',
        showBack: true,
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          AppTextField(
            controller: _name,
            label: 'Họ và tên *',
            enabled: !_saving,
            errorText: _fieldErrors['recipient_name'] ?? _required(_name.text),
          ),
          const SizedBox(height: AppSpacing.sm),
          AppTextField(
            controller: _phone,
            label: 'Số điện thoại *',
            keyboardType: TextInputType.phone,
            enabled: !_saving,
            errorText: _fieldErrors['phone_number'] ?? _phoneError,
          ),
          const SizedBox(height: AppSpacing.sm),
          if (_loadingLocations) const LinearProgressIndicator(),
          if (_locationsError != null) ...[
            Text(_locationsError!),
            TextButton(
              onPressed: _loadLocations,
              child: const Text('Tải lại khu vực'),
            ),
          ],
          DropdownButtonFormField<String>(
            key: ValueKey(
              'address-province-${_province.text}-${_locations.length}',
            ),
            initialValue: province?.name,
            isExpanded: true,
            decoration: InputDecoration(
              labelText: 'Tỉnh/Thành phố *',
              errorText:
                  _fieldErrors['province'] ??
                  (_submitted && province == null
                      ? 'Vui lòng chọn tỉnh/thành.'
                      : null),
            ),
            items: _locations
                .map(
                  (item) => DropdownMenuItem(
                    value: item.name,
                    child: Text(item.name),
                  ),
                )
                .toList(),
            onChanged: _saving
                ? null
                : (value) => setState(() {
                    _province.text = value ?? '';
                    _ward.clear();
                  }),
          ),
          const SizedBox(height: AppSpacing.sm),
          DropdownButtonFormField<String>(
            key: ValueKey('address-ward-${_province.text}-${_ward.text}'),
            initialValue:
                province?.wards.any((ward) => ward.name == _ward.text) == true
                ? _ward.text
                : null,
            isExpanded: true,
            decoration: InputDecoration(
              labelText: 'Phường/Xã *',
              errorText:
                  _fieldErrors['ward'] ??
                  (_submitted &&
                          (province == null ||
                              !province.wards.any(
                                (ward) => ward.name == _ward.text,
                              ))
                      ? 'Vui lòng chọn phường/xã.'
                      : null),
            ),
            items: (province?.wards ?? [])
                .map(
                  (ward) => DropdownMenuItem(
                    value: ward.name,
                    child: Text(ward.name),
                  ),
                )
                .toList(),
            onChanged: _saving || province == null
                ? null
                : (value) => setState(() => _ward.text = value ?? ''),
          ),
          const SizedBox(height: AppSpacing.sm),
          AppTextField(
            controller: _street,
            label: 'Địa chỉ chi tiết *',
            hint: 'Số nhà, tên đường…',
            enabled: !_saving,
            errorText:
                _fieldErrors['street_address'] ?? _required(_street.text),
          ),
          const SizedBox(height: 16),
          SwitchListTile(
            title: const Text('Đặt làm địa chỉ mặc định'),
            value: _isDefault,
            onChanged: _saving
                ? null
                : (value) => setState(() => _isDefault = value),
          ),
          const SizedBox(height: 18),
          if (_message != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Semantics(liveRegion: true, child: Text(_message!)),
            ),
          AppPrimaryButton(
            label: 'Lưu địa chỉ',
            loading: _saving,
            onPressed: _loadingLocations || _locationsError != null
                ? null
                : _save,
          ),
        ],
      ),
    );
  }
}
