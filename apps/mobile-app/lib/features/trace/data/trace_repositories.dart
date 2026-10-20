import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/features/trace/domain/trace_repository.dart';

class RestTraceRepository implements TraceRepository {
  const RestTraceRepository(this._client);
  final ApiClient _client;
  @override
  Future<TraceRecord> lookup(String code) async => TraceRecord.fromJson(
    await _client.getJson('/api/v1/trace/${Uri.encodeComponent(code)}'),
  );
}

class MockTraceRepository implements TraceRepository {
  @override
  Future<TraceRecord> lookup(String code) async {
    if (code != 'QR-OMA-20261015-LOT08') {
      throw const ApiException(
        Failure(
          kind: FailureKind.notFound,
          message: 'Không tìm thấy mã truy xuất. Hãy kiểm tra mã trên bao bì.',
        ),
      );
    }
    return const TraceRecord(
      code: 'QR-OMA-20261015-LOT08',
      product: 'Kẹo Mè Xửng Giòn O Mạ Huế',
      batch: 'LOT-20261015-G08',
      stars: 4,
      certificate: 'OCOP-TTH-2024-089',
      productionDate: '2026-10-15',
      expiryDate: '2027-04-15',
      workshop: 'Xưởng Mè Xửng O Mạ, Hương Thủy, Huế',
      artisan: 'Nghệ nhân Lê Thị O Mạ',
      safety:
          'Dữ liệu chứng nhận mẫu từ API contract; chưa kiểm chứng chứng nhận thực tế.',
      origin: 'Mè vàng Quảng Điền, Đậu phụng A Lưới, Gừng tươi Bạch Mã',
      video: '',
    );
  }
}
