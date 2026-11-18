import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Keeps the 5 most recent product searches on this device only.
class SearchHistoryStore {
  SearchHistoryStore({FlutterSecureStorage? storage})
    : _storage = storage ?? const FlutterSecureStorage();

  static const _key = 'goi_hue_search_history';
  static const limit = 5;
  final FlutterSecureStorage _storage;

  Future<List<String>> read() async {
    try {
      final raw = await _storage.read(key: _key);
      if (raw == null) return const [];
      return (jsonDecode(raw) as List).whereType<String>().take(limit).toList();
    } catch (_) {
      return const [];
    }
  }

  /// Moves [query] to the front, removes duplicates (case-insensitive).
  Future<List<String>> add(String query) async {
    final value = query.trim();
    if (value.isEmpty) return read();
    final next = [
      value,
      ...(await read()).where((q) => q.toLowerCase() != value.toLowerCase()),
    ].take(limit).toList();
    await _write(next);
    return next;
  }

  Future<List<String>> remove(String query) async {
    final next = (await read()).where((q) => q != query).toList();
    await _write(next);
    return next;
  }

  Future<void> clear() => _storage.delete(key: _key);

  Future<void> _write(List<String> values) async {
    try {
      await _storage.write(key: _key, value: jsonEncode(values));
    } catch (_) {
      // History is a convenience; failing to save must not block searching.
    }
  }
}
