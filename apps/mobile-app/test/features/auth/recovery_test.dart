import 'dart:async';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/data/mock_auth_repository.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/auth/domain/auth_validation.dart';
import 'package:mobile_app/features/auth/domain/recovery_verification.dart';
import 'package:mobile_app/features/auth/presentation/auth_return_location.dart';
import 'package:mobile_app/features/auth/presentation/recovery_cubit.dart';

class RecoveryRepositoryMock extends Mock implements AuthRepository {}

void main() {
  const strong = 'TriKyMatKhau!2026';
  test(
    'Web password policy counts Unicode and rejects whitespace as special',
    () {
      expect(AuthValidation.password(strong), isTrue);
      expect(AuthValidation.password('Ngắn!12'), isFalse);
      expect(AuthValidation.password('MatKhauDaiCoSo123 '), isFalse);
      expect(AuthValidation.password('ĐâyLàMậtKhẩuMới123!'), isTrue);
      expect(AuthValidation.password('${'A' * 125}a1!'), isTrue);
      expect(AuthValidation.password('${'A' * 126}a1!'), isFalse);
      expect(AuthValidation.phone('+84 905-123-456'), isTrue);
    },
  );

  test(
    'auth return target rejects external, auth loops and unknown routes',
    () {
      expect(safeAuthReturnLocation('/checkout'), '/checkout');
      for (final target in [
        'https://evil.test',
        '//evil.test',
        '/login',
        '/account-recovery',
        '/unknown',
        '/%2f%2fevil.test',
        '/orders/../login',
      ]) {
        expect(safeAuthReturnLocation(target), '/account', reason: target);
      }
    },
  );

  test(
    'neutral request, proof, validation, reset and consumed proof',
    () async {
      final repository = MockAuthRepository();
      final cubit = RecoveryCubit(repository);
      await cubit.request('invalid');
      expect(cubit.state.fieldErrors, contains('email'));
      await cubit.request('unknown@example.com');
      expect(cubit.state.phase, RecoveryPhase.sent);
      await cubit.verify('mock-token');
      expect(cubit.state.phase, RecoveryPhase.reset);
      await cubit.reset('short', 'different');
      expect(
        cubit.state.fieldErrors.keys,
        containsAll(['newPassword', 'newPasswordConfirmation']),
      );
      await cubit.reset(strong, strong);
      expect(cubit.state.phase, RecoveryPhase.success);
      final verification = await repository.verifyRecovery(
        'mock-token',
        idempotencyKey: 'a',
      );
      await repository.resetPassword(
        verification.resetProof,
        strong,
        idempotencyKey: 'b',
      );
      await expectLater(
        repository.resetPassword(
          verification.resetProof,
          strong,
          idempotencyKey: 'c',
        ),
        throwsA(isA<ApiException>()),
      );
      await cubit.close();
    },
  );

  test('expired link returns to request with recovery action', () async {
    final cubit = RecoveryCubit(MockAuthRepository());
    await cubit.verify('expired');
    expect(cubit.state.phase, RecoveryPhase.request);
    expect(cubit.state.message, contains('hết hạn'));
    expect(cubit.state.pending, isFalse);
    await cubit.close();
  });

  test(
    'identical network retry keeps key; deliberate resend gets a new key',
    () async {
      final repository = RecoveryRepositoryMock();
      final keys = <String>[];
      when(
        () => repository.requestRecovery(
          any(),
          idempotencyKey: any(named: 'idempotencyKey'),
        ),
      ).thenAnswer((invocation) async {
        keys.add(invocation.namedArguments[#idempotencyKey] as String);
        if (keys.length == 1) throw StateError('Network failure');
      });
      final cubit = RecoveryCubit(repository);
      await cubit.request('an@example.com');
      await cubit.request('an@example.com');
      expect(keys[0], keys[1]);
      cubit.newRequest();
      await cubit.request('an@example.com');
      expect(keys[2], isNot(keys[1]));
      await cubit.close();
    },
  );

  test('429 cooldown and pending state prevent duplicate calls', () async {
    final repository = RecoveryRepositoryMock();
    final completion = Completer<void>();
    var calls = 0;
    when(
      () => repository.requestRecovery(
        any(),
        idempotencyKey: any(named: 'idempotencyKey'),
      ),
    ).thenAnswer((_) {
      calls++;
      return completion.future;
    });
    final cubit = RecoveryCubit(repository);
    final pending = cubit.request('an@example.com');
    await cubit.request('an@example.com');
    expect(calls, 1);
    completion.completeError(
      const ApiException(
        Failure(
          kind: FailureKind.rateLimited,
          message: 'Chờ trước khi thử lại',
          retryAfterSeconds: 30,
        ),
      ),
    );
    await pending;
    expect(cubit.canSubmit, isFalse);
    await cubit.request('an@example.com');
    expect(calls, 1);
    expect(cubit.state.retryAt, isNotNull);
    await cubit.close();
  });

  test('expired proof never submits a new password', () async {
    final repository = RecoveryRepositoryMock();
    when(
      () => repository.verifyRecovery(
        any(),
        idempotencyKey: any(named: 'idempotencyKey'),
      ),
    ).thenAnswer(
      (_) async => RecoveryVerification(
        resetProof: 'expired',
        expiresAt: DateTime.now().subtract(const Duration(minutes: 1)),
      ),
    );
    final cubit = RecoveryCubit(repository);
    await cubit.verify('token');
    await cubit.reset(strong, strong);
    verifyNever(
      () => repository.resetPassword(
        any(),
        any(),
        idempotencyKey: any(named: 'idempotencyKey'),
      ),
    );
    expect(cubit.state.phase, RecoveryPhase.request);
    await cubit.close();
  });
}
