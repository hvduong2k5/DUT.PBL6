import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_web_auth_2/flutter_web_auth_2.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';

Future<Uri> openNativeIdentityBrowser(Uri url) async {
  if (kIsWeb ||
      !{
        TargetPlatform.android,
        TargetPlatform.iOS,
      }.contains(defaultTargetPlatform)) {
    throw const ApiException(
      Failure(
        kind: FailureKind.validation,
        message: 'Đăng nhập Identity hiện hỗ trợ Android và iOS.',
      ),
    );
  }
  try {
    return Uri.parse(
      await FlutterWebAuth2.authenticate(
        url: url.toString(),
        callbackUrlScheme: 'omama',
      ),
    );
  } on PlatformException catch (error) {
    throw ApiException(
      Failure(
        kind: FailureKind.validation,
        message: error.code == 'CANCELED'
            ? 'Đã hủy đăng nhập. Bạn có thể thử lại.'
            : 'Chưa mở được trình duyệt đăng nhập. Vui lòng thử lại.',
      ),
    );
  }
}
