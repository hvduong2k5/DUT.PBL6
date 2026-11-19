import 'dart:convert';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/data/mock_auth_repository.dart';
import 'package:mobile_app/features/auth/data/local_account_store.dart';
import 'package:mobile_app/features/auth/data/local_password_hasher.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/auth_validation.dart';
import 'package:mobile_app/features/auth/domain/recovery_verification.dart';

String _phone(String value) =>
    AuthValidation.normalizePhone(value).replaceFirst(RegExp(r'^\+84'), '0');

class _Account {
  const _Account(this.user, this.verifier);
  factory _Account.fromJson(Map<String, dynamic> json) {
    final user = AuthUser.fromJson(
      Map<String, dynamic>.from(json['user'] as Map),
    );
    final validId =
        RegExp(
          r'^[0-9a-fA-F]{8}(-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}$',
        ).hasMatch(user.id) ||
        user.id == 'CUST-MOCK' && user.phoneNumber == '0905123456' ||
        user.id == MockAuthRepository.b2bFixtureId &&
            user.phoneNumber == MockAuthRepository.b2bFixturePhone;
    if (!validId ||
        user.fullName.trim().isEmpty ||
        !AuthValidation.phone(user.phoneNumber) ||
        user.phoneNumber != _phone(user.phoneNumber) ||
        !AuthValidation.email(user.email ?? '')) {
      throw const FormatException('Invalid local account');
    }
    return _Account(
      user,
      LocalPasswordVerifier.fromJson(
        Map<String, dynamic>.from(json['verifier'] as Map),
      ),
    );
  }
  final AuthUser user;
  final LocalPasswordVerifier verifier;
  Map<String, dynamic> toJson() => {
    'user': {
      'customer_id': user.id,
      'phone_number': user.phoneNumber,
      'full_name': user.fullName,
      'email': user.email,
    },
    'verifier': verifier.toJson(),
  };
}

class _Database {
  _Database(this.accounts, {this.activeId, this.expiresAt});
  factory _Database.decode(String value) {
    final json = jsonDecode(value) as Map;
    if (json['version'] != 1 || json['accounts'] is! List) {
      throw const FormatException('Invalid local vault');
    }
    final accounts = (json['accounts'] as List)
        .map((v) => _Account.fromJson(Map<String, dynamic>.from(v as Map)))
        .toList();
    if (accounts.map((a) => a.user.id).toSet().length != accounts.length ||
        accounts.map((a) => a.user.phoneNumber).toSet().length !=
            accounts.length ||
        accounts.map((a) => a.user.email!.toLowerCase()).toSet().length !=
            accounts.length) {
      throw const FormatException('Duplicate local identity');
    }
    final active = json['active_id'] as String?,
        expiry = json['expires_at'] as String?;
    if ((active == null) != (expiry == null) ||
        expiry != null && !expiry.endsWith('Z') ||
        active != null && !accounts.any((a) => a.user.id == active)) {
      throw const FormatException('Invalid local session');
    }
    return _Database(
      accounts,
      activeId: active,
      expiresAt: expiry == null ? null : DateTime.parse(expiry),
    );
  }
  final List<_Account> accounts;
  String? activeId;
  DateTime? expiresAt;
  String encode() => jsonEncode({
    'version': 1,
    'accounts': accounts.map((a) => a.toJson()).toList(),
    'active_id': activeId,
    'expires_at': expiresAt?.toUtc().toIso8601String(),
  });
}

// Strict offline QA accounts; never used by REST/Identity authentication.
class LocalAuthRepository extends MockAuthRepository {
  LocalAuthRepository({
    LocalAccountStore? store,
    LocalPasswordHasher? hasher,
    DateTime Function()? now,
  }) : _store = store ?? const SecureLocalAccountStore(),
       _hasher = hasher ?? const Pbkdf2LocalPasswordHasher(),
       _now = now ?? DateTime.now;
  // Reserved demo accounts only; arbitrary/new phones are never auto-registered.
  static const fixturePassword = 'local-fixture';
  final LocalAccountStore _store;
  final LocalPasswordHasher _hasher;
  final DateTime Function() _now;
  static final _fixtures = [
    _Account(
      const AuthUser(
        id: 'CUST-MOCK',
        phoneNumber: '0905123456',
        fullName: 'Nguyễn Văn An',
        email: 'an.nguyen@example.com',
      ),
      const LocalPasswordVerifier(
        salt: 'ZBGjssWNQJ/iOXnIR69rkA==',
        digest: 'OPyR2i32nBICHKyV4NaJ9iOMjBHIvzV3M4ovYp8QHJg=',
      ),
    ),
    _Account(
      const AuthUser(
        id: MockAuthRepository.b2bFixtureId,
        phoneNumber: MockAuthRepository.b2bFixturePhone,
        fullName: 'Doanh nghiệp thử nghiệm local',
        email: 'b2b.fixture@example.invalid',
      ),
      const LocalPasswordVerifier(
        salt: 'fqXY995wQrGpGTx00o5QZg==',
        digest: 'amtmSOSBl36nf/hAwp+/K1/A+YwYTqRzXpvjPYY1GOE=',
      ),
    ),
  ];
  Future<_Database> _load() async {
    final value = await _store.read();
    if (value != null) return _Database.decode(value);
    final db = _Database([..._fixtures]);
    await _store.write(db.encode());
    return db;
  }

  Future<T> _authenticate<T>(Future<T> Function() action) =>
      _store.exclusive(() async {
        try {
          return await action();
        } catch (_) {
          currentUser = null;
          rethrow;
        }
      });
  ApiException _failure(FailureKind kind, String message) =>
      ApiException(Failure(kind: kind, message: message));
  AuthSession _session(AuthUser user) => AuthSession(
    user: user,
    accessToken: 'local-only',
    refreshToken: 'local-only',
  );
  Future<void> _activate(_Database db, AuthUser user) async {
    db.activeId = user.id;
    db.expiresAt = _now().toUtc().add(const Duration(days: 7));
    await _store.write(db.encode());
    currentUser = user;
  }

  @override
  Future<AuthUser?> restoreSession() => _store.exclusive(() async {
    currentUser = null;
    final db = await _load();
    if (db.activeId == null) {
      currentUser = null;
      return null;
    }
    if (!db.expiresAt!.isAfter(_now())) {
      db.activeId = null;
      db.expiresAt = null;
      await _store.write(db.encode());
      currentUser = null;
      return null;
    }
    return currentUser = db.accounts
        .singleWhere((a) => a.user.id == db.activeId)
        .user;
  });
  @override
  Future<AuthSession> register({
    required String phoneNumber,
    required String password,
    required String fullName,
    String? email,
  }) => _authenticate(() async {
    final phone = _phone(phoneNumber),
        name = fullName.trim(),
        mail = email?.trim().toLowerCase() ?? '';
    if (!AuthValidation.phone(phone) ||
        !AuthValidation.password(password) ||
        name.isEmpty ||
        !AuthValidation.email(mail)) {
      throw _failure(
        FailureKind.validation,
        'Vui lòng kiểm tra thông tin đăng ký.',
      );
    }
    final db = await _load();
    if (db.accounts.any(
      (a) => a.user.phoneNumber == phone || a.user.email!.toLowerCase() == mail,
    )) {
      throw _failure(
        FailureKind.conflict,
        'Số điện thoại hoặc email đã được đăng ký trên thiết bị này.',
      );
    }
    final user = AuthUser(
      id: const Uuid().v4(),
      phoneNumber: phone,
      fullName: name,
      email: mail,
    );
    db.accounts.add(_Account(user, await _hasher.hash(password)));
    await _activate(db, user);
    return _session(user);
  });
  @override
  Future<AuthSession> login({
    required String phoneNumber,
    required String password,
  }) => _authenticate(() async {
    final db = await _load(), phone = _phone(phoneNumber);
    final account = db.accounts
        .where((a) => a.user.phoneNumber == phone)
        .firstOrNull;
    final matches = await _hasher.verify(
      password,
      account?.verifier ?? _fixtures.first.verifier,
    );
    if (account == null || !matches) {
      db.activeId = null;
      db.expiresAt = null;
      await _store.write(db.encode());
      currentUser = null;
      throw _failure(
        FailureKind.unauthenticated,
        'Số điện thoại hoặc mật khẩu không đúng.',
      );
    }
    await _activate(db, account.user);
    return _session(account.user);
  });
  @override
  Future<void> logout() => _store.exclusive(() async {
    try {
      final db = await _load();
      db.activeId = null;
      db.expiresAt = null;
      await _store.write(db.encode());
    } catch (_) {
      throw _failure(
        FailureKind.unknown,
        'Đã thoát trong phiên này nhưng chưa lưu được đăng xuất. Vui lòng thử lại trước khi đóng app.',
      );
    } finally {
      currentUser = null;
    }
  });
  @override
  Future<void> updateUser(AuthUser user) => _store.exclusive(() async {
    if (currentUser?.id != user.id) {
      throw _failure(FailureKind.forbidden, 'Phiên tài khoản đã thay đổi.');
    }
    final db = await _load();
    final index = db.accounts.indexWhere((a) => a.user.id == user.id);
    final email = user.email?.trim().toLowerCase() ?? '';
    if (index < 0 ||
        user.phoneNumber != db.accounts[index].user.phoneNumber ||
        user.fullName.trim().isEmpty ||
        !AuthValidation.email(email) ||
        db.accounts.any(
          (a) => a.user.id != user.id && a.user.email!.toLowerCase() == email,
        )) {
      throw _failure(
        FailureKind.validation,
        'Thông tin tài khoản chưa hợp lệ.',
      );
    }
    final updated = AuthUser(
      id: user.id,
      phoneNumber: user.phoneNumber,
      fullName: user.fullName.trim(),
      email: email,
    );
    db.accounts[index] = _Account(updated, db.accounts[index].verifier);
    await _store.write(db.encode());
    currentUser = updated;
  });
  @override
  Future<void> requestRecovery(
    String email, {
    required String idempotencyKey,
  }) async {}
  @override
  Future<RecoveryVerification> verifyRecovery(
    String token, {
    required String idempotencyKey,
  }) async => throw _failure(
    FailureKind.expired,
    'Tài khoản local chưa có dịch vụ xác minh khôi phục mật khẩu.',
  );
  @override
  Future<void> resetPassword(
    String proof,
    String password, {
    required String idempotencyKey,
  }) async => throw _failure(
    FailureKind.expired,
    'Chưa có xác minh khôi phục hợp lệ cho tài khoản local.',
  );
}
