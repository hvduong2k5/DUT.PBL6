package com.hvduong.identity.infrastructure.persistence.mybatis;

import com.hvduong.identity.application.registration.RegisterPrincipalUseCase;
import com.hvduong.identity.application.dto.request.RegistrationRequest;
import com.hvduong.identity.infrastructure.keycloak.KeycloakAdminClient;
import com.hvduong.identity.infrastructure.persistence.mybatis.mapper.OutboxEventMapper;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.AfterEach;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.boot.test.mock.mockito.SpyBean;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.context.ActiveProfiles;
import java.util.UUID;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicInteger;
import static org.assertj.core.api.Assertions.*;
import static org.mockito.Mockito.*;
import static org.mockito.ArgumentMatchers.*;

@SpringBootTest(properties={"identity.registration.fingerprint-key=atomicity-test-key-with-at-least-32-bytes", "grpc.server.port=-1"})
@ActiveProfiles("test")
class RegistrationAtomicityIntegrationTest {
 @Autowired RegisterPrincipalUseCase registration;
 @Autowired JdbcTemplate jdbc;
 @MockBean KeycloakAdminClient keycloak;
 @SpyBean OutboxEventMapper outbox;
 private final java.util.Set<String> keys=java.util.concurrent.ConcurrentHashMap.newKeySet();

 @AfterEach void removeOnlyOwnedFixtureRows() {
  for(String key:keys) {
   var ids=jdbc.queryForList("SELECT principal_id FROM registration_operations WHERE idempotency_key=?",UUID.class,key);
   for(UUID id:ids) {
    jdbc.update("DELETE FROM outbox_events WHERE aggregate_id=?",id.toString());
    jdbc.update("DELETE FROM registration_operations WHERE principal_id=?",id);
    jdbc.update("DELETE FROM identity_principals WHERE principal_id=?",id);
   }
  }
 }

 RegistrationRequest input(String key) {
  keys.add(key);
  return RegistrationRequest.builder().email(key+"@example.test").firstName("Synthetic").lastName("Customer").rawPassword("Synthetic-Password-2026").idempotencyKey(key).requestFingerprint("client-forged-fingerprint").build();
 }

 @Test void finalizationRollsBackIfOutboxWriteFails() {
  String key=UUID.randomUUID().toString();
  when(keycloak.createUser(any(),anyString(),anyString(),anyString(),anyString())).thenReturn("test-subject-"+key);
  doThrow(new IllegalStateException("injected outbox outage")).when(outbox).insert(any());
  try {
   assertThatThrownBy(()->registration.registerCustomer(input(key))).isInstanceOf(IllegalStateException.class);
   assertThat(jdbc.queryForObject("SELECT p.provisioning_status FROM identity_principals p JOIN registration_operations r USING(principal_id) WHERE r.idempotency_key=?",String.class,key)).isEqualTo("PENDING");
   assertThat(jdbc.queryForObject("SELECT operation_status FROM registration_operations WHERE idempotency_key=?",String.class,key)).isEqualTo("PENDING");
   assertThat(jdbc.queryForObject("SELECT count(*) FROM outbox_events WHERE aggregate_id IN (SELECT principal_id::text FROM registration_operations WHERE idempotency_key=?)",Integer.class,key)).isZero();
  } finally { reset(outbox); }
 }

 @Test void concurrentReplayCreatesOnePrincipalAndIgnoresClientFingerprint() throws Exception {
  int before=jdbc.queryForObject("SELECT count(*) FROM identity_principals",Integer.class);
  String key=UUID.randomUUID().toString();AtomicInteger externalCalls=new AtomicInteger();
  when(keycloak.createUser(any(),anyString(),anyString(),anyString(),anyString())).thenAnswer(call->{externalCalls.incrementAndGet();Thread.sleep(100);return "test-subject-"+key;});
  try(var executor=Executors.newFixedThreadPool(2)) {
   var a=executor.submit(()->registration.registerCustomer(input(key)));
   var b=executor.submit(()->registration.registerCustomer(input(key)));
   assertThat(a.get().getPrincipalId()).isEqualTo(b.get().getPrincipalId());
  }
  assertThat(externalCalls.get()).isEqualTo(1);
  assertThat(jdbc.queryForObject("SELECT count(*) FROM identity_principals",Integer.class)).isEqualTo(before+1);
  assertThat(jdbc.queryForObject("SELECT count(*) FROM registration_operations WHERE idempotency_key=?",Integer.class,key)).isEqualTo(1);
  var changed=input(key);changed.setRawPassword("Other-Synthetic-Password");
  assertThat(registration.registerCustomer(changed).getStatus()).isEqualTo("CONFLICT");
  assertThat(registration.registerCustomer(input(key)).getStatus()).isEqualTo("COMPLETED");
 }
}
