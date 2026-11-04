import 'dart:async';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/auth/domain/auth_validation.dart';
import 'package:mobile_app/features/auth/domain/browser_auth_repository.dart';
import 'package:mobile_app/features/auth/domain/identity_persona.dart';
import 'package:mobile_app/features/auth/domain/recovery_verification.dart';
import 'identity_api.dart';
import 'identity_session_store.dart';
import 'local_account_store.dart';

typedef IdentityBrowser = Future<Uri> Function(Uri authorizationUrl);

/// Target contract: user-supplied Identity brief (backend acceptance pending).
class IdentityAuthRepository
    implements AuthRepository, BrowserAuthRepository, SessionRecoverySource {
  IdentityAuthRepository({
    required this.api,
    required this.store,
    required this.openBrowser,
    required Set<String> authorizationOrigins,
    Set<String> providers = const {},
    DateTime Function()? now,
  }) : authorizationOrigins = Set.unmodifiable(authorizationOrigins),
       providers = Set.unmodifiable(providers),
       _now = now ?? DateTime.now;
  static const redirectUri = 'omama://auth/callback';
  final IdentityApi api;
  final IdentitySessionStore store;
  final IdentityBrowser openBrowser;
  final Set<String> authorizationOrigins;
  @override
  final Set<String> providers;
  final DateTime Function() _now;
  final _lock = LocalAccountLock();
  AuthUser? currentUser;
  bool _browserPending = false, _blocked = false;
  int _epoch = 0;
  Future<AuthUser?>? _restoring;
  Future<bool>? _refreshing;
  void Function(AuthUser?)? _observer;
  @override
  void observeSessionRecovery(void Function(AuthUser?)? observer) =>
      _observer = observer;
  ApiException _error(
    String message, [
    FailureKind kind = FailureKind.validation,
  ]) => ApiException(Failure(kind: kind, message: message));
  void _checkEpoch(int epoch) {
    if (epoch != _epoch) {
      throw _error('Phiên đã thay đổi. Vui lòng đăng nhập lại.');
    }
  }

  Future<String?> accessToken() async =>
      _blocked ? null : (await store.read())?.accessToken;
  AuthSession _session(AuthUser user, IdentitySessionRecord record) =>
      AuthSession(
        user: user,
        accessToken: record.accessToken,
        refreshToken: record.refreshToken,
      );
  Future<AuthUser> _profile(IdentitySessionRecord record) async {
    final reply = await api.request('GET', '/users/me', session: record);
    final persona = IdentityPersona.fromJson(reply.data);
    return AuthUser(
      id: persona.principalId,
      phoneNumber: '',
      fullName: persona.displayName.isEmpty
          ? persona.email
          : persona.displayName,
      email: persona.email,
      persona: persona,
    );
  }

  Future<void> _commit(
    int epoch,
    IdentitySessionRecord record,
    AuthUser user,
  ) => _lock.run(() async {
    _checkEpoch(epoch);
    await store.write(record);
    _checkEpoch(epoch);
    currentUser = user;
    _blocked = false;
  });
  Future<void> _clear(int epoch) => _lock.run(() async {
    _checkEpoch(epoch);
    currentUser = null;
    _blocked = true;
    await store.write(null);
  });

  @override
  Future<AuthSession> signIn({String? provider}) async {
    if (_browserPending || currentUser != null) {
      throw _error('Đăng nhập đang được xử lý hoặc tài khoản đã đăng nhập.');
    }
    if (provider != null && !providers.contains(provider)) {
      throw _error('Nhà cung cấp đăng nhập chưa được bật.');
    }
    _browserPending = true;
    final epoch = ++_epoch;
    try {
      final start = await api.request(
        'GET',
        '/public/auth/login',
        query: {
          'redirect_uri': redirectUri,
          'format': 'json',
          'provider': ?provider,
        },
      );
      _checkEpoch(epoch);
      final rawUrl = start.data['authorizationUrl'];
      if (rawUrl is! String) {
        throw const FormatException('Missing authorization URL');
      }
      final url = Uri.parse(rawUrl);
      final query = url.queryParametersAll;
      final state = query['state']?.singleOrNull;
      if (!authorizationOrigins.contains(url.origin) ||
          url.userInfo.isNotEmpty ||
          url.hasFragment ||
          !(url.scheme == 'https' ||
              api.allowInsecureLocal && url.scheme == 'http') ||
          state == null ||
          state.isEmpty ||
          query['response_type']?.singleOrNull != 'code' ||
          query['redirect_uri']?.singleOrNull != redirectUri ||
          query['code_challenge_method']?.singleOrNull != 'S256' ||
          !RegExp(
            r'^[A-Za-z0-9_-]{43}$',
          ).hasMatch(query['code_challenge']?.singleOrNull ?? '')) {
        throw const FormatException('Invalid authorization transaction');
      }
      final started = _now();
      final callback = await openBrowser(url);
      _checkEpoch(epoch);
      final result = callback.queryParametersAll;
      if (callback.scheme != 'omama' ||
          callback.host != 'auth' ||
          callback.path != '/callback' ||
          callback.userInfo.isNotEmpty ||
          callback.hasPort ||
          callback.hasFragment ||
          _now().difference(started) > const Duration(minutes: 10) ||
          result['state']?.singleOrNull != state ||
          result.containsKey('error') ||
          (result['code']?.singleOrNull ?? '').isEmpty) {
        throw _error('Không thể xác minh kết quả đăng nhập. Vui lòng thử lại.');
      }
      final reply = await api.request(
        'GET',
        '/public/auth/callback',
        query: {
          'code': result['code']!.single,
          'state': state,
          'format': 'json',
        },
        session: start.sessionId == null
            ? null
            : IdentitySessionRecord(
                accessToken: '',
                refreshToken: '',
                sessionId: start.sessionId,
              ),
      );
      _checkEpoch(epoch);
      final record = IdentitySessionRecord.fromJson({
        ...reply.data,
        'sessionId': reply.sessionId ?? reply.data['sessionId'],
      });
      if (record.sessionId == null) {
        throw const FormatException('Missing BFF session');
      }
      final user = await _profile(record);
      await _commit(epoch, record, user);
      return _session(user, record);
    } finally {
      _browserPending = false;
    }
  }

  @override
  Future<AuthUser?> restoreSession() =>
      _restoring ??= _restore().whenComplete(() => _restoring = null);
  Future<AuthUser?> _restore() async {
    final epoch = _epoch, record = await store.read();
    if (record == null || _blocked) return null;
    try {
      final reply = await api.request(
        'GET',
        '/public/auth/session',
        session: record,
      );
      _checkEpoch(epoch);
      if (reply.data['authenticated'] is! bool) {
        throw const FormatException('Invalid Identity session status');
      }
      if (reply.data['authenticated'] != true) {
        await _clear(epoch);
        return null;
      }
      final user = await _profile(record);
      await _commit(epoch, record, user);
      return user;
    } on ApiException catch (e) {
      if (e.failure.kind != FailureKind.unauthenticated) rethrow;
      return await refreshSession() ? currentUser : null;
    }
  }

  Future<bool> refreshSession() =>
      _refreshing ??= _refresh().whenComplete(() => _refreshing = null);
  Future<bool> _refresh() async {
    final epoch = _epoch, previous = await store.read();
    if (previous == null || _blocked) return false;
    try {
      final reply = await api.request(
        'POST',
        '/public/auth/tokens/refresh',
        session: previous,
      );
      _checkEpoch(epoch);
      final record = IdentitySessionRecord.fromJson({
        'accessToken': reply.data['accessToken'],
        'refreshToken': reply.data['refreshToken'] ?? previous.refreshToken,
        'sessionId':
            reply.sessionId ?? reply.data['sessionId'] ?? previous.sessionId,
      });
      final user = await _profile(record);
      await _commit(epoch, record, user);
      _observer?.call(user);
      return true;
    } on ApiException catch (e) {
      if (e.failure.kind != FailureKind.unauthenticated) rethrow;
      await _clear(epoch);
      _observer?.call(null);
      return false;
    }
  }

  @override
  Future<void> logout() async {
    final epoch = ++_epoch;
    _blocked = true;
    currentUser = null;
    _observer?.call(null);
    final previous = await store.read();
    await _clear(epoch);
    if (previous != null) {
      try {
        await api.request('POST', '/public/auth/logout', session: previous);
      } catch (_) {
        throw _error(
          'Đã đăng xuất trên thiết bị. Chưa xác nhận kết thúc phiên phía máy chủ.',
          FailureKind.network,
        );
      }
    }
  }

  @override
  Future<void> registerCustomer({
    required String firstName,
    required String lastName,
    required String email,
    required String password,
    required String idempotencyKey,
  }) async {
    if (firstName.trim().isEmpty ||
        lastName.trim().isEmpty ||
        !AuthValidation.email(email) ||
        password.isEmpty ||
        idempotencyKey.isEmpty) {
      throw _error('Vui lòng kiểm tra thông tin đăng ký.');
    }
    await api.request(
      'POST',
      '/public/customer-registrations',
      body: {
        'firstName': firstName.trim(),
        'lastName': lastName.trim(),
        'email': email.trim().toLowerCase(),
        'rawPassword': password,
        'idempotencyKey': idempotencyKey,
        'requestFingerprint': email.trim().toLowerCase(),
      },
      headers: {'Idempotency-Key': idempotencyKey},
    );
  }

  @override
  Future<void> changePassword({
    required String currentPassword,
    required String newPassword,
  }) async {
    final record = await store.read();
    if (_blocked || currentUser == null || record == null) {
      throw _error('Vui lòng đăng nhập lại.', FailureKind.unauthenticated);
    }
    if (currentPassword.isEmpty || newPassword.isEmpty) {
      throw _error('Vui lòng nhập đủ mật khẩu.');
    }
    await api.request(
      'PUT',
      '/users/me/password',
      session: record,
      body: {'currentPassword': currentPassword, 'newPassword': newPassword},
    );
  }

  // Identity owns password entry in its system-browser login page.
  @override
  Future<AuthSession> login({
    required String phoneNumber,
    required String password,
  }) => signIn();
  @override
  Future<AuthSession> register({
    required String phoneNumber,
    required String password,
    required String fullName,
    String? email,
  }) => throw _error('Vui lòng dùng biểu mẫu đăng ký Identity.');
  @override
  Future<void> requestRecovery(
    String email, {
    required String idempotencyKey,
  }) async =>
      throw _error('Vui lòng khôi phục mật khẩu trên trang đăng nhập bảo mật.');
  @override
  Future<RecoveryVerification> verifyRecovery(
    String token, {
    required String idempotencyKey,
  }) async =>
      throw _error('Liên kết khôi phục này thuộc luồng đăng nhập khác.');
  @override
  Future<void> resetPassword(
    String proof,
    String password, {
    required String idempotencyKey,
  }) async =>
      throw _error('Vui lòng khôi phục mật khẩu trên trang đăng nhập bảo mật.');
}
