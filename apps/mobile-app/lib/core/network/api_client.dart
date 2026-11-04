import 'package:dio/dio.dart';
import 'package:mobile_app/core/config/api_config.dart';
import 'package:mobile_app/core/network/api_error_mapper.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/storage/token_store.dart';
import 'package:mobile_app/core/network/session_cookies.dart';
import 'package:mobile_app/core/error/failures.dart';

class ApiClient {
  ApiClient({
    Dio? dio,
    TokenStore? tokenStore,
    String? baseUrl,
    Future<String?> Function()? accessTokenReader,
    bool manageCookies = true,
  }) : _tokenStore = tokenStore ?? TokenStore(),
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
    _cookies = manageCookies ? SessionCookies(this.dio) : null;
    if (!manageCookies) {
      this.dio.options.followRedirects = false;
    }
    final authorizedBase = Uri.parse(baseUrl ?? this.dio.options.baseUrl);
    this.dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          if (accessTokenReader != null &&
              (options.uri.origin != authorizedBase.origin ||
                  !options.uri.path.startsWith('/api/v1/'))) {
            handler.reject(
              DioException(
                requestOptions: options,
                type: DioExceptionType.cancel,
                error: 'Credential destination denied',
              ),
            );
            return;
          }
          try {
            final token =
                await (accessTokenReader?.call() ??
                    _tokenStore.readAccessToken());
            if (token != null && token.isNotEmpty) {
              options.headers['Authorization'] = 'Bearer $token';
            } else if (accessTokenReader != null) {
              options.headers.remove('Authorization');
            }
            handler.next(options);
          } catch (_) {
            handler.reject(
              DioException(
                requestOptions: options,
                error: 'Session storage unavailable',
                type: DioExceptionType.unknown,
              ),
            );
          }
        },
      ),
    );
  }
  final Dio dio;
  final TokenStore _tokenStore;
  late final SessionCookies? _cookies;
  Future<bool> Function()? recoverSession;
  Future<bool>? _recovery;
  Future<void> clearSessionCookies() async => _cookies?.clear();

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
              operationId: failure.operationId,
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

  Future<List<Map<String, dynamic>>> getJsonList(String path) async {
    try {
      final response = await dio.get<Object?>(path);
      final data = response.data;
      if (data is! List || data.any((item) => item is! Map)) {
        throw const FormatException('Expected a JSON object array');
      }
      return data
          .map((item) => Map<String, dynamic>.from(item as Map))
          .toList();
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

  Future<void> putVoid(
    String path, {
    Object? data,
    Map<String, dynamic>? headers,
  }) async {
    try {
      await dio.put<Object?>(
        path,
        data: data,
        options: Options(headers: headers),
      );
    } on DioException catch (error) {
      throw await _failure(error);
    }
  }

  Future<void> deleteVoid(String path, {Map<String, dynamic>? headers}) async {
    try {
      await dio.delete<Object?>(path, options: Options(headers: headers));
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
