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
  late final TextEditingController _district;
  late final TextEditingController _ward;
  late final TextEditingController _street;
  late bool _isDefault;
  bool _saving = false;
  bool _submitted = false;

  @override
  void initState() {
    super.initState();
    final value = widget.address;
    _name = TextEditingController(text: value?.recipientName);
    _phone = TextEditingController(text: value?.phoneNumber);
    _province = TextEditingController(text: value?.province);
    _district = TextEditingController(text: value?.district);
    _ward = TextEditingController(text: value?.ward);
    _street = TextEditingController(text: value?.streetAddress);
    _isDefault = value?.isDefault ?? false;
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
        _district,
        _ward,
        _street,
      ].every((field) => field.text.trim().isNotEmpty) &&
      _phoneError == null;

  Future<void> _save() async {
    setState(() => _submitted = true);
    if (!_valid) return;
    setState(() => _saving = true);
    final address = Address(
      id: widget.address?.id ?? const Uuid().v4(),
      recipientName: _name.text.trim(),
      phoneNumber: _phone.text.trim(),
      streetAddress: _street.text.trim(),
      ward: _ward.text.trim(),
      district: _district.text.trim(),
      province: _province.text.trim(),
      isDefault: _isDefault,
    );
    await context.read<AddressRepository>().save(address);
    if (mounted) context.pop();
  }

  @override
  void dispose() {
    for (final controller in [
      _name,
      _phone,
      _province,
      _district,
      _ward,
      _street,
    ]) {
      controller.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
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
          errorText: _required(_name.text),
        ),
        const SizedBox(height: AppSpacing.sm),
        AppTextField(
          controller: _phone,
          label: 'Số điện thoại *',
          keyboardType: TextInputType.phone,
          errorText: _phoneError,
        ),
        const SizedBox(height: AppSpacing.sm),
        AppTextField(
          controller: _province,
          label: 'Tỉnh/Thành phố *',
          errorText: _required(_province.text),
        ),
        const SizedBox(height: AppSpacing.sm),
        AppTextField(
          controller: _district,
          label: 'Quận/Huyện *',
          errorText: _required(_district.text),
        ),
        const SizedBox(height: AppSpacing.sm),
        AppTextField(
          controller: _ward,
          label: 'Phường/Xã *',
          errorText: _required(_ward.text),
        ),
        const SizedBox(height: AppSpacing.sm),
        AppTextField(
          controller: _street,
          label: 'Địa chỉ chi tiết *',
          hint: 'Số nhà, tên đường…',
          errorText: _required(_street.text),
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
        AppPrimaryButton(
          label: 'Lưu địa chỉ',
          loading: _saving,
          onPressed: _save,
        ),
      ],
    ),
  );
}
