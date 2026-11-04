package com.hvduong.catalog.grpc;

import io.grpc.*;
import net.devh.boot.grpc.server.interceptor.GrpcGlobalServerInterceptor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.annotation.Order;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;

/**
 * Validates internal token for east-west gRPC service calls.
 */
@GrpcGlobalServerInterceptor
@Order(0)
public class CatalogAccess implements ServerInterceptor {

    @Value("${catalog.internal-token:}")
    private String internalToken;

    @Override
    public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
            ServerCall<ReqT, RespT> call, Metadata headers, ServerCallHandler<ReqT, RespT> next) {
        String method = call.getMethodDescriptor().getFullMethodName();
        if (method.startsWith("omamx.catalog.v1.CatalogService/") || method.endsWith("/ValidatePriceAndSku")) {
            if (internalToken != null && !internalToken.isBlank()) {
                String token = headers.get(Metadata.Key.of("x-internal-token", Metadata.ASCII_STRING_MARSHALLER));
                if (internalToken.length() < 32 || token == null || !MessageDigest.isEqual(internalToken.getBytes(StandardCharsets.UTF_8), token.getBytes(StandardCharsets.UTF_8))) {
                    call.close(Status.UNAUTHENTICATED, new Metadata());
                    return new ServerCall.Listener<>() {};
                }
            }
        }
        return next.startCall(call, headers);
    }
}
