package com.hvduong.catalog.infrastructure.inventory;

import com.omamx.proto.inventory.v1.*;
import io.grpc.*;
import io.grpc.stub.MetadataUtils;
import io.grpc.netty.shaded.io.grpc.netty.GrpcSslContexts;
import io.grpc.netty.shaded.io.grpc.netty.NettyChannelBuilder;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import java.nio.file.*;
import java.util.*;
import java.util.concurrent.TimeUnit;

/** Live display availability; reservation remains the final Inventory decision. */
@Component
public class InventoryStockClient {
    @Value("${CATALOG_INVENTORY_TARGET:}") private String target;
    @Value("${CATALOG_INVENTORY_TOKEN_FILE:}") private String tokenFile;
    @Value("${CATALOG_INVENTORY_CA_FILE:}") private String caFile;
    @Value("${CATALOG_INVENTORY_SERVER_NAME:}") private String serverName;
    private ManagedChannel channel;
    private InventoryServiceGrpc.InventoryServiceBlockingStub client;
    @PostConstruct
    void configure() throws Exception {
        if (target.isBlank()) return;
        if (caFile.isBlank() || serverName.isBlank()) throw new IllegalArgumentException("Inventory requires pinned CA and server name");
        String token=Files.readString(Path.of(tokenFile)).trim();
        if(token.length()<32) throw new IllegalArgumentException("Inventory credential is too short");
        channel=NettyChannelBuilder.forTarget(target).sslContext(GrpcSslContexts.forClient().trustManager(Path.of(caFile).toFile()).build()).overrideAuthority(serverName).build();
        var headers=new Metadata();headers.put(Metadata.Key.of("authorization",Metadata.ASCII_STRING_MARSHALLER),"Bearer "+token);
        client=InventoryServiceGrpc.newBlockingStub(channel).withInterceptors(MetadataUtils.newAttachHeadersInterceptor(headers));
    }
    public boolean configured() { return client!=null; }
    public Map<String,Long> available(Collection<String> skus) {
        Map<String,Long> quantities=new HashMap<>();skus.forEach(sku->quantities.put(sku,null));
        if(skus.isEmpty())return quantities;
        try {
            var response=client.withDeadlineAfter(2,TimeUnit.SECONDS).getStockLevel(GetStockLevelRequest.newBuilder().addAllSkuCodes(skus).build());
            for(var item:response.getItemsList()) if(quantities.containsKey(item.getSkuCode()) && item.getAvailableQuantity()>=0)
                quantities.put(item.getSkuCode(),"OUT_OF_STOCK".equals(item.getStockStatus())?0L:(long)item.getAvailableQuantity());
        } catch (StatusRuntimeException unavailable) { /* Preserve unknown; never turn an outage into zero stock or stale availability. */ }
        return quantities;
    }
    @PreDestroy void close() { if(channel!=null)channel.shutdownNow(); }
}
