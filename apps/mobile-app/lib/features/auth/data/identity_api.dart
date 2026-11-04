import 'package:dio/dio.dart';
import 'package:mobile_app/core/network/api_error_mapper.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'identity_session_store.dart';

class IdentityReply {
  const IdentityReply(this.data, this.sessionId, this.authStateCookie);
  final Map<String, dynamic> data;
  final String? sessionId;
  final String? authStateCookie;
}

/// Dedicated BFF transport: cookies go only to this configured origin/path.
class IdentityApi {
  IdentityApi({required Uri baseUri, Dio? dio, this.allowInsecureLocal = false})
    : baseUri = _validateBase(baseUri, allowInsecureLocal),
      dio = dio ?? Dio() {
    this.dio.options.connectTimeout = const Duration(seconds: 15);
    this.dio.options.receiveTimeout = const Duration(seconds: 15);
    this.dio.options.followRedirects = false;
  }
  final Uri baseUri;
  final Dio dio;
  final bool allowInsecureLocal;
  static Uri _validateBase(Uri uri, bool allowLocal) {
    if (!uri.hasAuthority ||
        uri.userInfo.isNotEmpty ||
        uri.hasQuery ||
        uri.hasFragment ||
        !(uri.scheme == 'https' || allowLocal && uri.scheme == 'http') ||
        uri.path != '/api/v1' && uri.path != '/api/v1/') {
      throw ArgumentError('Identity base URL must end in /api/v1');
    }
    return uri.replace(path: '/api/v1/');
  }

  Future<IdentityReply> request(
    String method,
    String path, {
    Map<String, dynamic>? query,
    Object? body,
    IdentitySessionRecord? session,
    String? authStateCookie,
    Map<String, String>? headers,
  }) async {
    if (!RegExp(r'^/[a-z0-9/-]+$').hasMatch(path) ||
        path.startsWith('//') ||
        path.contains('..') ||
        Uri.parse(path).hasQuery ||
        Uri.parse(path).hasFragment) {
      throw ArgumentError('Invalid Identity route');
    }
    final uri = baseUri.resolve(path.substring(1));
    if (authStateCookie != null &&
        (path != '/public/auth/callback' || !_validCookie(authStateCookie))) {
      throw ArgumentError('Invalid authorization transaction cookie');
    }
    final cookieOnly = path.startsWith('/public/auth/');
    try {
      final response = await dio.requestUri<Object?>(
        uri.replace(queryParameters: query),
        data: body,
        options: Options(
          method: method,
          followRedirects: false,
          headers: {
            'Accept': 'application/json',
            'Content-Type': 'application/json',
            ...?headers,
            if (!cookieOnly &&
                session != null &&
                session.accessToken.isNotEmpty)
              'Authorization': 'Bearer ${session.accessToken}',
            if (authStateCookie != null)
              'Cookie': 'OMAMA_AUTH_STATE=$authStateCookie'
            else if (session?.sessionId != null)
              'Cookie': 'OMAMA_SESSION=${session!.sessionId}',
          },
        ),
      );
      final envelope = response.data;
      if (envelope is! Map ||
          envelope['success'] != true ||
          envelope['data'] != null && envelope['data'] is! Map) {
        throw const FormatException('Invalid Identity response envelope');
      }
      String? readCookie(String name) {
        String? cookie;
        var found = false;
        for (final value in response.headers['set-cookie'] ?? <String>[]) {
          final pair = value.trim().split(';').first;
          if (!pair.startsWith('$name=')) continue;
          if (found) throw const FormatException('Duplicate Identity cookie');
          found = true;
          cookie = pair.substring(name.length + 1);
          if (cookie.isEmpty) return null;
          if (!_validCookie(cookie)) {
            throw const FormatException('Invalid Identity cookie');
          }
        }
        return cookie;
      }

      return IdentityReply(
        Map<String, dynamic>.from(envelope['data'] as Map? ?? {}),
        readCookie('OMAMA_SESSION'),
        readCookie('OMAMA_AUTH_STATE'),
      );
    } on DioException catch (error) {
      throw ApiException(ApiErrorMapper.fromDio(error));
    }
  }

  static bool _validCookie(String value) =>
      RegExp(r'^[A-Za-z0-9._~+/%=-]+$').hasMatch(value);
}
