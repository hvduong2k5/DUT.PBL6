import 'package:dio/dio.dart';
import 'package:mobile_app/core/config/api_config.dart';
import 'package:mobile_app/core/network/api_error_mapper.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/storage/token_store.dart';
import 'package:mobile_app/core/network/session_cookies.dart';
import 'package:mobile_app/core/error/failures.dart';

class ApiClient {
  ApiClient({Dio? dio, TokenStore? tokenStore, String? baseUrl})
    : _tokenStore = tokenStore ?? TokenStore(),
      dio =
          dio ??
          Dio(
            BaseOptions(
              baseUrl: baseUrl ?? ApiConfig.baseUrl,
              connectTimeout: ApiConfig.connectTimeout,
              receiveTimeout: ApiConfig.receiveTimeout,
              headers: const {'Accept': 'application/json'},
            ),
          ) {
    _cookies = SessionCookies(this.dio);
    this.dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          final token = await _tokenStore.readAccessToken();
          if (token != null && token.isNotEmpty) {
            options.headers['Authorization'] = 'Bearer $token';
          }
          handler.next(options);
        },
      ),
    );
  }
  final Dio dio;
  final TokenStore _tokenStore;
  late final SessionCookies _cookies;
  Future<bool> Function()? recoverSession;
  Future<bool>? _recovery;
  Future<void> clearSessionCookies() => _cookies.clear();

  Future<ApiException> _failure(DioException error) async {
    final failure = ApiErrorMapper.fromDio(error);
    if (failure.kind == FailureKind.unauthenticated &&
        !error.requestOptions.path.startsWith('/api/v1/auth/') &&
        recoverSession != null) {
      try {
        final recovered = await (_recovery ??= recoverSession!().whenComplete(
          () => _recovery = null,
        ));
        if (recovered) {
          return ApiException(
            Failure(
              kind: failure.kind,
              message: 'Phiên vừa được làm mới. Vui lòng thử lại thao tác.',
              code: failure.code,
              requestId: failure.requestId,
            ),
          );
        }
      } catch (_) {
        // The original failure remains actionable; no write is replayed.
      }
    }
    return ApiException(failure);
  }

  Future<Map<String, dynamic>> getJson(
    String path, {
    Map<String, dynamic>? query,
    Map<String, dynamic>? headers,
  }) async {
    try {
      final response = await dio.get<Object?>(
        path,
        queryParameters: query,
        options: Options(headers: headers),
      );
      return _asMap(response.data);
    } on DioException catch (error) {
      throw await _failure(error);
    }
  }

  Future<Map<String, dynamic>> postJson(
    String path, {
    Object? data,
    Map<String, dynamic>? headers,
  }) async {
    try {
      final response = await dio.post<Object?>(
        path,
        data: data,
        options: Options(headers: headers),
      );
      return _asMap(response.data);
    } on DioException catch (error) {
      throw await _failure(error);
    }
  }

  Future<Map<String, dynamic>> putJson(
    String path, {
    Object? data,
    Map<String, dynamic>? headers,
  }) async {
    try {
      final response = await dio.put<Object?>(
        path,
        data: data,
        options: Options(headers: headers),
      );
      return _asMap(response.data);
    } on DioException catch (error) {
      throw await _failure(error);
    }
  }

  Future<Map<String, dynamic>> deleteJson(
    String path, {
    Object? data,
    Map<String, dynamic>? headers,
  }) async {
    try {
      final response = await dio.delete<Object?>(
        path,
        data: data,
        options: Options(headers: headers),
      );
      return _asMap(response.data);
    } on DioException catch (error) {
      throw await _failure(error);
    }
  }

  static Map<String, dynamic> _asMap(Object? value) =>
      value is Map ? Map<String, dynamic>.from(value) : <String, dynamic>{};
}
