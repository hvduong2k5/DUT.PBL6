package com.hvduong.catalog.grpc;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.grpc.*;
import net.devh.boot.grpc.server.interceptor.GrpcGlobalServerInterceptor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.annotation.Order;
import java.net.URI;
import java.net.http.*;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.time.Duration;

/** Identity's existing ADMIN role is the MVP admin boundary; caller actor_id is never authority. */
@GrpcGlobalServerInterceptor
@Order(0)
public class CatalogAccess implements ServerInterceptor {
    private static final Context.Key<String> ACTOR = Context.key("catalog-verified-actor");
    @Value("${catalog.internal-token:}") private String internalToken;
    @Value("${catalog.identity-url:}") private String identityUrl;
    private final ObjectMapper json;
    private final HttpClient http = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(2)).followRedirects(HttpClient.Redirect.NEVER).build();
    public CatalogAccess(ObjectMapper json) { this.json = json; }
    public static String actorId() {
        String actor=ACTOR.get(); if(actor==null) throw Status.PERMISSION_DENIED.asRuntimeException(); return actor;
    }
    @Override
    public <ReqT,RespT> ServerCall.Listener<ReqT> interceptCall(ServerCall<ReqT,RespT> call, Metadata headers, ServerCallHandler<ReqT,RespT> next) {
        String method=call.getMethodDescriptor().getFullMethodName();
        if (method.startsWith("omamx.catalog.v1.CatalogService/") || method.endsWith("/ValidatePriceAndSku")) {
            String token=headers.get(Metadata.Key.of("x-internal-token",Metadata.ASCII_STRING_MARSHALLER));
            if (internalToken.length()<32 || token==null || !MessageDigest.isEqual(internalToken.getBytes(StandardCharsets.UTF_8),token.getBytes(StandardCharsets.UTF_8))) {
                call.close(Status.UNAUTHENTICATED,new Metadata());return new ServerCall.Listener<>() {};
            }
        }
        if (!method.startsWith("hvduong.catalog.v1.CatalogAdminService/")) return next.startCall(call,headers);
        String authorization=headers.get(Metadata.Key.of("authorization",Metadata.ASCII_STRING_MARSHALLER));
        if (authorization==null || !authorization.startsWith("Bearer ") || authorization.length()>8192) {call.close(Status.UNAUTHENTICATED,new Metadata());return new ServerCall.Listener<>() {};}
        if(identityUrl.isBlank()){call.close(Status.UNAVAILABLE,new Metadata());return new ServerCall.Listener<>() {};}
        try {
            var response=http.send(HttpRequest.newBuilder(URI.create(identityUrl.replaceAll("/$", "")+"/api/v1/users/me"))
                    .header("Authorization",authorization).timeout(Duration.ofSeconds(3)).GET().build(),HttpResponse.BodyHandlers.ofString());
            if(response.statusCode()!=200){call.close(response.statusCode()==401?Status.UNAUTHENTICATED:Status.PERMISSION_DENIED,new Metadata());return new ServerCall.Listener<>() {};}
            var envelope=json.readTree(response.body());var data=envelope.path("data");boolean admin=false;
            for(var role:data.path("roles")) if("ADMIN".equals(role.path("roleCode").asText()))admin=true;
            if(!envelope.path("success").asBoolean() || !"EMPLOYEE".equals(data.path("principalType").asText()) || !"ACTIVE".equals(data.path("provisioningStatus").asText()) || !admin) {
                call.close(Status.PERMISSION_DENIED,new Metadata());return new ServerCall.Listener<>() {};
            }
            String actor=java.util.UUID.fromString(data.path("principalId").asText()).toString();
            return Contexts.interceptCall(Context.current().withValue(ACTOR,actor),call,headers,next);
        } catch (Exception error) {call.close(Status.UNAVAILABLE.withDescription("IDENTITY_UNAVAILABLE"),new Metadata());return new ServerCall.Listener<>() {};}
    }
}
