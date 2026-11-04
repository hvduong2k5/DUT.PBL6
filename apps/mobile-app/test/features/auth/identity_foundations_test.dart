import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:mobile_app/features/auth/data/identity_api.dart';
import 'package:mobile_app/features/auth/data/identity_session_store.dart';
import 'package:mobile_app/features/auth/domain/identity_persona.dart';

Map<String, dynamic> profile({
  String type = 'CUSTOMER',
  String status = 'ACTIVE',
}) => {
  'principalId': 'principal-test',
  'email': 'qa@example.invalid',
  'firstName': 'Synthetic',
  'lastName': 'Customer',
  'principalType': type,
  'provisioningStatus': status,
  'roles': <Object?>[],
  'permissions': <String>[],
};
void main() {
  test(
    'only active Customer can purchase; permissions never elevate other personas',
    () {
      expect(IdentityPersona.fromJson(profile()).canPurchase, isTrue);
      for (final type in ['EMPLOYEE', 'UNKNOWN', 'customer']) {
        final json = profile(type: type)..['permissions'] = ['CHECKOUT_CREATE'];
        expect(IdentityPersona.fromJson(json).canPurchase, isFalse);
      }
      for (final status in ['SUSPENDED', 'PENDING', 'active']) {
        expect(
          IdentityPersona.fromJson(profile(status: status)).canPurchase,
          isFalse,
        );
      }
    },
  );
  test('incomplete or privileged Customer projections are rejected', () {
    for (final key in [
      'principalId',
      'email',
      'principalType',
      'provisioningStatus',
      'roles',
      'permissions',
    ]) {
      expect(
        () => IdentityPersona.fromJson(profile()..remove(key)),
        throwsFormatException,
      );
    }
    expect(
      () => IdentityPersona.fromJson(profile()..['roles'] = ['ADMIN']),
      throwsFormatException,
    );
    expect(
      () => IdentityPersona.fromJson(profile()..['permissions'] = [1]),
      throwsFormatException,
    );
  });
  test('persona collections cannot be mutated via caller aliases', () {
    final roles = <Object?>['ADMIN'], permissions = <String>[];
    final persona = IdentityPersona.fromJson(
      profile(type: 'EMPLOYEE')
        ..['roles'] = roles
        ..['permissions'] = permissions,
    );
    roles.clear();
    permissions.add('CHECKOUT_CREATE');
    expect(persona.roles, ['ADMIN']);
    expect(persona.permissions, isEmpty);
    expect(
      () => persona.permissions.add('CHECKOUT_CREATE'),
      throwsUnsupportedError,
    );
    expect(persona.canPurchase, isFalse);
  });
  test(
    'saved Identity session is atomic, scoped, restored and durably cleared',
    () async {
      final a = SecureIdentitySessionStore('https://a.invalid/api/v1');
      const record = IdentitySessionRecord(
        accessToken: 'access.fixture',
        refreshToken: 'refresh.fixture',
        sessionId: 'cookie.fixture',
      );
      await a.write(record);
      expect(
        (await SecureIdentitySessionStore(
          'https://a.invalid/api/v1',
        ).read())!.sessionId,
        'cookie.fixture',
      );
      expect(
        await SecureIdentitySessionStore('https://b.invalid/api/v1').read(),
        isNull,
      );
      await a.write(null);
      expect(await a.read(), isNull);
      expect(
        await const FlutterSecureStorage().read(key: 'oma_local_accounts_v1'),
        isNull,
      );
    },
  );
  test('unsafe Identity origins/paths cannot receive credentials', () async {
    for (final url in [
      'http://identity.invalid/api/v1',
      'https://user@identity.invalid/api/v1',
      'https://identity.invalid/',
    ]) {
      expect(() => IdentityApi(baseUri: Uri.parse(url)), throwsArgumentError);
    }
    final api = IdentityApi(
      baseUri: Uri.parse('https://identity.invalid/api/v1'),
    );
    for (final path in [
      '//evil.invalid',
      '/https://evil.invalid',
      '/../users/me',
      '/users/me?secret=x',
    ]) {
      await expectLater(api.request('GET', path), throwsArgumentError);
    }
  });
  test('invalid tokens or cookie injection never become saved credentials', () {
    expect(
      () => IdentitySessionRecord.fromJson({
        'accessToken': '',
        'refreshToken': 'refresh',
      }),
      throwsFormatException,
    );
    expect(
      () => IdentitySessionRecord.fromJson({
        'accessToken': 'a',
        'refreshToken': 'r',
        'sessionId': 'x; evil=1',
      }),
      throwsFormatException,
    );
  });
}
