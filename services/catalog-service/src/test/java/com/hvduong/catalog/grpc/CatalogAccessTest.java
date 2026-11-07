package com.hvduong.catalog.grpc;

import static org.mockito.Mockito.*;

import org.junit.jupiter.api.Test;
import org.springframework.test.util.ReflectionTestUtils;

import com.omamx.proto.catalog.v1.CatalogServiceGrpc;

import io.grpc.*;

class CatalogAccessTest {

  @Test
  @SuppressWarnings({"unchecked", "rawtypes"})
  void rejectsInternalCheckoutSnapshotWithoutValidInternalToken() {
    var access = new CatalogAccess();
    ReflectionTestUtils.setField(
        access, "internalToken", "very-long-internal-secret-token-32-chars");

    var call = mock(ServerCall.class);
    when(call.getMethodDescriptor()).thenReturn(CatalogServiceGrpc.getGetCheckoutSnapshotMethod());
    var handler = mock(ServerCallHandler.class);
    access.interceptCall(call, new Metadata(), handler);
    verify(call).close(eq(Status.UNAUTHENTICATED), any(Metadata.class));
    verifyNoInteractions(handler);
  }

  @Test
  @SuppressWarnings({"unchecked", "rawtypes"})
  void allowsInternalCheckoutSnapshotWithValidInternalToken() {
    var access = new CatalogAccess();
    String validToken = "very-long-internal-secret-token-32-chars";
    ReflectionTestUtils.setField(access, "internalToken", validToken);

    var call = mock(ServerCall.class);
    when(call.getMethodDescriptor()).thenReturn(CatalogServiceGrpc.getGetCheckoutSnapshotMethod());
    var handler = mock(ServerCallHandler.class);
    when(handler.startCall(any(), any())).thenReturn(new ServerCall.Listener() {});

    var headers = new Metadata();
    headers.put(Metadata.Key.of("x-internal-token", Metadata.ASCII_STRING_MARSHALLER), validToken);

    access.interceptCall(call, headers, handler);
    verify(handler).startCall(call, headers);
    verify(call, never()).close(any(), any());
  }
}
