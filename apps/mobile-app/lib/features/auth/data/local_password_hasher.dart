import 'dart:convert';
import 'dart:isolate';
import 'dart:math';
import 'package:cryptography/cryptography.dart';

class LocalPasswordVerifier {
  const LocalPasswordVerifier({
    required this.salt,
    required this.digest,
    this.iterations = 120000,
  });
  factory LocalPasswordVerifier.fromJson(Map<String, dynamic> json) {
    if (json['algorithm'] != 'pbkdf2-sha256-v1' ||
        json['iterations'] is! int ||
        (json['iterations'] as int) < 10000 ||
        (json['iterations'] as int) > 1000000 ||
        json['salt'] is! String ||
        json['digest'] is! String) {
      throw const FormatException('Invalid local credential record');
    }
    final value = LocalPasswordVerifier(
      salt: json['salt'] as String,
      digest: json['digest'] as String,
      iterations: json['iterations'] as int,
    );
    if (base64Decode(value.salt).length != 16 ||
        base64Decode(value.digest).length != 32) {
      throw const FormatException('Invalid local credential length');
    }
    return value;
  }
  final String salt, digest;
  final int iterations;
  Map<String, dynamic> toJson() => {
    'algorithm': 'pbkdf2-sha256-v1',
    'iterations': iterations,
    'salt': salt,
    'digest': digest,
  };
}

abstract interface class LocalPasswordHasher {
  Future<LocalPasswordVerifier> hash(String password);
  Future<bool> verify(String password, LocalPasswordVerifier verifier);
}

class Pbkdf2LocalPasswordHasher implements LocalPasswordHasher {
  const Pbkdf2LocalPasswordHasher();
  static Future<String> _derive(String password, String salt, int iterations) =>
      Isolate.run(() async {
        final algorithm = Pbkdf2(
          macAlgorithm: Hmac.sha256(),
          iterations: iterations,
          bits: 256,
        );
        final key = await algorithm.deriveKeyFromPassword(
          password: password,
          nonce: base64Decode(salt),
        );
        return base64Encode(await key.extractBytes());
      });
  @override
  Future<LocalPasswordVerifier> hash(String password) async {
    final random = Random.secure();
    final salt = base64Encode(
      List<int>.generate(16, (_) => random.nextInt(256)),
    );
    return LocalPasswordVerifier(
      salt: salt,
      digest: await _derive(password, salt, 120000),
    );
  }

  @override
  Future<bool> verify(String password, LocalPasswordVerifier verifier) async {
    final actual = base64Decode(
      await _derive(password, verifier.salt, verifier.iterations),
    );
    final expected = base64Decode(verifier.digest);
    var difference = actual.length ^ expected.length;
    for (var i = 0; i < expected.length; i++) {
      difference |= actual[i] ^ expected[i];
    }
    return difference == 0;
  }
}
