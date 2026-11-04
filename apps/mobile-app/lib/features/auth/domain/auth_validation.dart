/// Customer Web validation rules, shared by registration and recovery.
abstract final class AuthValidation {
  static bool email(String value) =>
      RegExp(r'^[^\s@]+@[^\s@]+\.[^\s@]+$').hasMatch(value.trim());

  static String normalizePhone(String value) =>
      value.trim().replaceAll(RegExp(r'[ .-]'), '');

  static bool phone(String value) =>
      RegExp(r'^(?:\+84|0)[0-9]{9,10}$').hasMatch(normalizePhone(value));

  static Map<String, bool> passwordChecks(String value) => {
    'Từ 15 ký tự': value.runes.length >= 15,
    'Không quá 128 ký tự': value.runes.length <= 128,
    'Có chữ hoa': RegExp(r'\p{Lu}', unicode: true).hasMatch(value),
    'Có chữ thường': RegExp(r'\p{Ll}', unicode: true).hasMatch(value),
    'Có chữ số': RegExp(r'\p{Nd}', unicode: true).hasMatch(value),
    'Có ký tự đặc biệt': RegExp(r'[\p{P}\p{S}]', unicode: true).hasMatch(value),
  };

  static bool password(String value) =>
      passwordChecks(value).values.every((valid) => valid);
}
