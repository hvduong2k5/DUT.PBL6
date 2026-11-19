import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class OnboardingStore {
  OnboardingStore({FlutterSecureStorage? storage})
    : _storage = storage ?? const FlutterSecureStorage();

  static const _completedKey = 'oma_onboarding_completed';
  final FlutterSecureStorage _storage;

  Future<bool> isCompleted() async {
    final value = await _storage.read(key: _completedKey);
    return value == 'true';
  }

  Future<void> complete() => _storage.write(key: _completedKey, value: 'true');
}
