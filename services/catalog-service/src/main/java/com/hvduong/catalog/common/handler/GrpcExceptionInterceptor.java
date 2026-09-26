package com.hvduong.catalog.common.handler;

import com.hvduong.catalog.common.exception.CatalogException;
import com.hvduong.catalog.common.exception.ErrorCode;
import io.grpc.ForwardingServerCallListener;
import io.grpc.Metadata;
import io.grpc.ServerCall;
import io.grpc.ServerCallHandler;
import io.grpc.ServerInterceptor;
import io.grpc.Status;
import lombok.extern.slf4j.Slf4j;
import net.devh.boot.grpc.server.interceptor.GrpcGlobalServerInterceptor;

/**
 * Global gRPC exception interceptor — error boundary cho toàn bộ gRPC server.
 * <p>
 * Bắt mọi exception ném ra từ handler và chuyển đổi thành gRPC {@link Status}
 * tương ứng để trả về client thay vì để gRPC trả về UNKNOWN.
 * <p>
 * Thứ tự xử lý:
 * 1. {@link CatalogException} → map qua protoCode của {@link ErrorCode}
 * 2. {@link IllegalArgumentException} → INVALID_ARGUMENT
 * 3. Các exception khác → INTERNAL
 */
@Slf4j
@GrpcGlobalServerInterceptor
public class GrpcExceptionInterceptor implements ServerInterceptor {

    @Override
    public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
            ServerCall<ReqT, RespT> call,
            Metadata headers,
            ServerCallHandler<ReqT, RespT> next) {

        ServerCall.Listener<ReqT> delegate = next.startCall(call, headers);
        return new ExceptionHandlingServerCallListener<>(delegate, call, headers);
    }

    // ── Inner listener ────────────────────────────────────────────────────────

    private static class ExceptionHandlingServerCallListener<ReqT, RespT>
            extends ForwardingServerCallListener.SimpleForwardingServerCallListener<ReqT> {

        private final ServerCall<ReqT, RespT> serverCall;
        private final Metadata headers;

        ExceptionHandlingServerCallListener(ServerCall.Listener<ReqT> delegate,
                                            ServerCall<ReqT, RespT> serverCall,
                                            Metadata headers) {
            super(delegate);
            this.serverCall = serverCall;
            this.headers = headers;
        }

        @Override
        public void onHalfClose() {
            try {
                super.onHalfClose();
            } catch (Exception ex) {
                handleException(ex);
            }
        }

        @Override
        public void onReady() {
            try {
                super.onReady();
            } catch (Exception ex) {
                handleException(ex);
            }
        }

        private void handleException(Exception ex) {
            if (ex instanceof CatalogException ce) {
                log.warn("[gRPC] CatalogException: code={}, dev={}",
                        ce.getErrorCode(), ce.getDevMessage());
                serverCall.close(toGrpcStatus(ce.getErrorCode())
                        .withDescription(ce.getErrorCode().getUserMessage()), headers);

            } else if (ex instanceof IllegalArgumentException) {
                log.warn("[gRPC] IllegalArgumentException: {}", ex.getMessage());
                serverCall.close(Status.INVALID_ARGUMENT
                        .withDescription(ex.getMessage())
                        .withCause(ex), headers);

            } else {
                log.error("[gRPC] Unexpected exception", ex);
                serverCall.close(Status.INTERNAL
                        .withDescription("Hệ thống gặp sự cố, vui lòng thử lại sau.")
                        .withCause(ex), headers);
            }
        }

        private static Status toGrpcStatus(ErrorCode code) {
            return switch (code.getHttpStatus()) {
                case 400 -> Status.INVALID_ARGUMENT;
                case 401 -> Status.UNAUTHENTICATED;
                case 403 -> Status.PERMISSION_DENIED;
                case 404 -> Status.NOT_FOUND;
                case 409 -> Status.ALREADY_EXISTS;
                case 422 -> Status.FAILED_PRECONDITION;
                case 503 -> Status.UNAVAILABLE;
                case 504 -> Status.DEADLINE_EXCEEDED;
                default  -> Status.INTERNAL;
            };
        }
    }
}
