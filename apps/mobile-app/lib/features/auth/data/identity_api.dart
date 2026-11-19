import 'package:dio/dio.dart';
import 'package:mobile_app/core/network/api_error_mapper.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'identity_session_store.dart';

class IdentityReply {
  const IdentityReply(this.data, this.sessionId);
  final Map<String, dynamic> data;
  final String? sessionId;
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
  String? _authStateCookie;
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
            if (session != null &&
                session.accessToken.isNotEmpty &&
                !path.startsWith('/public/auth/'))
              'Authorization': 'Bearer ${session.accessToken}',
            if (session?.sessionId != null ||
                path == '/public/auth/callback' && _authStateCookie != null)
              'Cookie': [
                if (session?.sessionId != null)
                  'OMAMA_SESSION=${session!.sessionId}',
                if (path == '/public/auth/callback' && _authStateCookie != null)
                  'OMAMA_AUTH_STATE=$_authStateCookie',
              ].join('; '),
          },
        ),
      );
      final envelope = response.data;
      if (envelope is! Map ||
          envelope['success'] != true ||
          envelope['data'] != null && envelope['data'] is! Map) {
        throw const FormatException('Invalid Identity response envelope');
      }
      String? cookie;
      for (final value in response.headers['set-cookie'] ?? <String>[]) {
        if (value.startsWith('OMAMA_AUTH_STATE=')) {
          final state = value
              .substring('OMAMA_AUTH_STATE='.length)
              .split(';')
              .first;
          _authStateCookie = state.isEmpty ? null : state;
        }
        if (value.startsWith('OMAMA_SESSION=')) {
          cookie = value.substring('OMAMA_SESSION='.length).split(';').first;
          if (cookie.isEmpty) cookie = null;
        }
      }
      if (cookie != null &&
          !RegExp(r'^[A-Za-z0-9._~+/%=-]+$').hasMatch(cookie)) {
        throw const FormatException('Invalid BFF cookie');
      }
      return IdentityReply(
        Map<String, dynamic>.from(envelope['data'] as Map? ?? {}),
        cookie,
      );
    } on DioException catch (error) {
      throw ApiException(ApiErrorMapper.fromDio(error));
    } finally {
      if (path == '/public/auth/callback') _authStateCookie = null;
    }
  }
}
