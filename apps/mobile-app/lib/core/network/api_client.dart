import 'package:dio/dio.dart';
import 'package:mobile_app/core/config/api_config.dart';
import 'package:mobile_app/core/network/api_error_mapper.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/storage/token_store.dart';

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
      throw ApiException(ApiErrorMapper.fromDio(error));
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
      throw ApiException(ApiErrorMapper.fromDio(error));
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
      throw ApiException(ApiErrorMapper.fromDio(error));
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
      throw ApiException(ApiErrorMapper.fromDio(error));
    }
  }

  static Map<String, dynamic> _asMap(Object? value) =>
      value is Map ? Map<String, dynamic>.from(value) : <String, dynamic>{};
}
