package com.hvduong.catalog.grpc;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

import com.hvduong.catalog.common.enums.ApprovalStatus;
import com.hvduong.catalog.common.enums.ListingStatus;
import com.hvduong.catalog.domain.entity.ChannelPrice;
import com.hvduong.catalog.domain.entity.Product;
import com.hvduong.catalog.domain.entity.ProductVariant;
import com.hvduong.catalog.repository.mybatis.ChannelPriceMapper;
import com.hvduong.catalog.repository.mybatis.ProductMapper;
import com.hvduong.catalog.repository.mybatis.ProductVariantMapper;

import java.math.BigDecimal;
import java.util.Optional;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicReference;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import com.omamx.proto.catalog.v1.*;
import com.omamx.proto.common.v1.Money;

import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;

@ExtendWith(MockitoExtension.class)
class CatalogGrpcServiceTest {

  @Mock private ProductVariantMapper variantMapper;
  @Mock private ProductMapper productMapper;
  @Mock private ChannelPriceMapper channelPriceMapper;

  private CatalogGrpcMapper grpcMapper;
  private CatalogGrpcService grpcService;

  @BeforeEach
  void setUp() {
    grpcMapper = new CatalogGrpcMapper();
    grpcService =
        new CatalogGrpcService(variantMapper, productMapper, channelPriceMapper, grpcMapper);
  }

  // ── ValidateCheckoutPricing Tests ─────────────────────────────────────────

  @Test
  void validateCheckoutPricing_success() {
    UUID productId = UUID.randomUUID();
    UUID variantId = UUID.randomUUID();
    String sku = "MX-GION-500G";

    Product product =
        Product.builder()
            .id(productId)
            .name("Mè xửng giòn Huế")
            .approvalStatus(ApprovalStatus.APPROVED)
            .listingStatus(ListingStatus.ACTIVE)
            .version(2L)
            .build();

    ProductVariant variant =
        ProductVariant.builder()
            .id(variantId)
            .productId(productId)
            .skuCode(sku)
            .variantName("Hộp 500g")
            .listingStatus(ListingStatus.ACTIVE)
            .weightValue(new BigDecimal("500"))
            .weightUnit("GRAM")
            .packagingType("HOP_THIEC")
            .version(1L)
            .priceRevision(5L)
            .build();

    ChannelPrice price =
        ChannelPrice.builder()
            .id(UUID.randomUUID())
            .variantId(variantId)
            .channel(com.hvduong.catalog.common.enums.SalesChannel.D2C_WEB)
            .amountUnits(100000L)
            .amountNanos(0)
            .currencyCode("VND")
            .build();

    when(variantMapper.findBySkuCode(sku)).thenReturn(Optional.of(variant));
    when(productMapper.findById(productId)).thenReturn(Optional.of(product));
    when(channelPriceMapper.findActiveByVariantAndChannel(variantId, "D2C_WEB"))
        .thenReturn(Optional.of(price));

    ValidateCheckoutPricingRequest request =
        ValidateCheckoutPricingRequest.newBuilder()
            .setChannel(SalesChannel.SALES_CHANNEL_D2C_WEB)
            .addItems(
                CheckoutItemInput.newBuilder()
                    .setSkuCode(sku)
                    .setQuantity(2)
                    .setClientUnitPrice(
                        Money.newBuilder().setCurrencyCode("VND").setUnits(100000L).setNanos(0))
                    .build())
            .build();

    AtomicReference<ValidateCheckoutPricingResponse> responseRef = new AtomicReference<>();
    AtomicReference<Throwable> errorRef = new AtomicReference<>();

    grpcService.validateCheckoutPricing(
        request,
        new StreamObserver<>() {
          @Override
          public void onNext(ValidateCheckoutPricingResponse value) {
            responseRef.set(value);
          }

          @Override
          public void onError(Throwable t) {
            errorRef.set(t);
          }

          @Override
          public void onCompleted() {}
        });

    assertNull(errorRef.get());
    assertNotNull(responseRef.get());
    ValidateCheckoutPricingResponse response = responseRef.get();
    assertTrue(response.getIsValid());
    assertEquals(200000L, response.getCanonicalSubtotal().getUnits());
    assertEquals("VND", response.getCanonicalSubtotal().getCurrencyCode());
    assertEquals(0, response.getDiscrepanciesCount());
    assertEquals(1, response.getItemsCount());

    ValidatedCheckoutItem item = response.getItems(0);
    assertEquals(sku, item.getSkuCode());
    assertEquals("Mè xửng giòn Huế", item.getProductName());
    assertEquals(100000L, item.getAuthoritativeUnitPrice().getUnits());
    assertTrue(item.getIsSellable());
    assertEquals(5L, item.getPriceVersion());
    assertEquals(PackagingType.PACKAGING_TYPE_HOP_THIEC, item.getPackagingType());
  }

  @Test
  void validateCheckoutPricing_priceDiscrepancy() {
    UUID productId = UUID.randomUUID();
    UUID variantId = UUID.randomUUID();
    String sku = "MX-DAO-250G";

    Product product =
        Product.builder()
            .id(productId)
            .name("Mè xửng dẻo")
            .approvalStatus(ApprovalStatus.APPROVED)
            .listingStatus(ListingStatus.ACTIVE)
            .version(1L)
            .build();

    ProductVariant variant =
        ProductVariant.builder()
            .id(variantId)
            .productId(productId)
            .skuCode(sku)
            .listingStatus(ListingStatus.ACTIVE)
            .priceRevision(3L)
            .build();

    ChannelPrice price =
        ChannelPrice.builder()
            .id(UUID.randomUUID())
            .variantId(variantId)
            .channel(com.hvduong.catalog.common.enums.SalesChannel.D2C_WEB)
            .amountUnits(90000L)
            .amountNanos(0)
            .currencyCode("VND")
            .build();

    when(variantMapper.findBySkuCode(sku)).thenReturn(Optional.of(variant));
    when(productMapper.findById(productId)).thenReturn(Optional.of(product));
    when(channelPriceMapper.findActiveByVariantAndChannel(variantId, "D2C_WEB"))
        .thenReturn(Optional.of(price));

    ValidateCheckoutPricingRequest request =
        ValidateCheckoutPricingRequest.newBuilder()
            .setChannel(SalesChannel.SALES_CHANNEL_D2C_WEB)
            .addItems(
                CheckoutItemInput.newBuilder()
                    .setSkuCode(sku)
                    .setQuantity(1)
                    .setClientUnitPrice(
                        Money.newBuilder().setCurrencyCode("VND").setUnits(80000L).setNanos(0))
                    .build())
            .build();

    AtomicReference<ValidateCheckoutPricingResponse> responseRef = new AtomicReference<>();
    grpcService.validateCheckoutPricing(
        request,
        new StreamObserver<>() {
          @Override
          public void onNext(ValidateCheckoutPricingResponse value) {
            responseRef.set(value);
          }

          @Override
          public void onError(Throwable t) {}

          @Override
          public void onCompleted() {}
        });

    ValidateCheckoutPricingResponse response = responseRef.get();
    assertNotNull(response);
    assertFalse(response.getIsValid());
    assertEquals(1, response.getDiscrepanciesCount());
    PriceDiscrepancy disc = response.getDiscrepancies(0);
    assertEquals(sku, disc.getSkuCode());
    assertEquals(80000L, disc.getClientUnitPrice().getUnits());
    assertEquals(90000L, disc.getAuthoritativeUnitPrice().getUnits());
    assertEquals(ValidationIssue.VALIDATION_ISSUE_PRICE_CHANGED, disc.getDiscrepancyReason());
    assertTrue(response.getItems(0).getIsSellable());
  }

  @Test
  void validateCheckoutPricing_rejectsDuplicateSku() {
    ValidateCheckoutPricingRequest request =
        ValidateCheckoutPricingRequest.newBuilder()
            .setChannel(SalesChannel.SALES_CHANNEL_D2C_WEB)
            .addItems(CheckoutItemInput.newBuilder().setSkuCode("SKU-DUP").setQuantity(1).build())
            .addItems(CheckoutItemInput.newBuilder().setSkuCode("SKU-DUP").setQuantity(2).build())
            .build();

    AtomicReference<Throwable> errorRef = new AtomicReference<>();
    grpcService.validateCheckoutPricing(
        request,
        new StreamObserver<>() {
          @Override
          public void onNext(ValidateCheckoutPricingResponse value) {}

          @Override
          public void onError(Throwable t) {
            errorRef.set(t);
          }

          @Override
          public void onCompleted() {}
        });

    assertNotNull(errorRef.get());
    assertInstanceOf(StatusRuntimeException.class, errorRef.get());
    assertEquals(
        Status.Code.INVALID_ARGUMENT,
        ((StatusRuntimeException) errorRef.get()).getStatus().getCode());
  }

  @Test
  void validateCheckoutPricing_rejectsUnspecifiedChannel() {
    ValidateCheckoutPricingRequest request =
        ValidateCheckoutPricingRequest.newBuilder()
            .setChannel(SalesChannel.SALES_CHANNEL_UNSPECIFIED)
            .addItems(CheckoutItemInput.newBuilder().setSkuCode("SKU-1").setQuantity(1).build())
            .build();

    AtomicReference<Throwable> errorRef = new AtomicReference<>();
    grpcService.validateCheckoutPricing(
        request,
        new StreamObserver<>() {
          @Override
          public void onNext(ValidateCheckoutPricingResponse value) {}

          @Override
          public void onError(Throwable t) {
            errorRef.set(t);
          }

          @Override
          public void onCompleted() {}
        });

    assertNotNull(errorRef.get());
    assertInstanceOf(StatusRuntimeException.class, errorRef.get());
    assertEquals(
        Status.Code.INVALID_ARGUMENT,
        ((StatusRuntimeException) errorRef.get()).getStatus().getCode());
  }

  @Test
  void validateCheckoutPricing_skuNotFound() {
    String sku = "UNKNOWN-SKU";
    when(variantMapper.findBySkuCode(sku)).thenReturn(Optional.empty());

    ValidateCheckoutPricingRequest request =
        ValidateCheckoutPricingRequest.newBuilder()
            .setChannel(SalesChannel.SALES_CHANNEL_D2C_WEB)
            .addItems(CheckoutItemInput.newBuilder().setSkuCode(sku).setQuantity(1).build())
            .build();

    AtomicReference<ValidateCheckoutPricingResponse> responseRef = new AtomicReference<>();
    grpcService.validateCheckoutPricing(
        request,
        new StreamObserver<>() {
          @Override
          public void onNext(ValidateCheckoutPricingResponse value) {
            responseRef.set(value);
          }

          @Override
          public void onError(Throwable t) {}

          @Override
          public void onCompleted() {}
        });

    ValidateCheckoutPricingResponse response = responseRef.get();
    assertNotNull(response);
    assertFalse(response.getIsValid());
    assertEquals(1, response.getDiscrepanciesCount());
    assertEquals(
        ValidationIssue.VALIDATION_ISSUE_SKU_NOT_FOUND,
        response.getDiscrepancies(0).getDiscrepancyReason());
    assertFalse(response.getItems(0).getIsSellable());
  }

  // ── GetProductVariant Tests ───────────────────────────────────────────────

  @Test
  void getProductVariant_successWithChannel() {
    UUID productId = UUID.randomUUID();
    UUID variantId = UUID.randomUUID();
    String sku = "MX-GION-500G";

    Product product = Product.builder().id(productId).name("Mè xửng giòn Huế").ocopStar(4).build();

    ProductVariant variant =
        ProductVariant.builder()
            .id(variantId)
            .productId(productId)
            .skuCode(sku)
            .variantName("Hộp 500g")
            .listingStatus(ListingStatus.ACTIVE)
            .weightValue(new BigDecimal("500"))
            .weightUnit("GRAM")
            .packagingType("HOP_THIEC")
            .shelfLifeDays(180)
            .build();

    ChannelPrice price =
        ChannelPrice.builder()
            .id(UUID.randomUUID())
            .variantId(variantId)
            .channel(com.hvduong.catalog.common.enums.SalesChannel.D2C_WEB)
            .amountUnits(100000L)
            .amountNanos(0)
            .currencyCode("VND")
            .build();

    when(variantMapper.findBySkuCode(sku)).thenReturn(Optional.of(variant));
    when(productMapper.findById(productId)).thenReturn(Optional.of(product));
    when(channelPriceMapper.findActiveByVariantAndChannel(variantId, "D2C_WEB"))
        .thenReturn(Optional.of(price));

    GetProductVariantRequest request =
        GetProductVariantRequest.newBuilder()
            .setSkuCode(sku)
            .setChannel(SalesChannel.SALES_CHANNEL_D2C_WEB)
            .build();

    AtomicReference<GetProductVariantResponse> responseRef = new AtomicReference<>();
    grpcService.getProductVariant(
        request,
        new StreamObserver<>() {
          @Override
          public void onNext(GetProductVariantResponse value) {
            responseRef.set(value);
          }

          @Override
          public void onError(Throwable t) {}

          @Override
          public void onCompleted() {}
        });

    GetProductVariantResponse response = responseRef.get();
    assertNotNull(response);
    ProductVariantDetail detail = response.getVariant();
    assertEquals(sku, detail.getSkuCode());
    assertEquals("Mè xửng giòn Huế", detail.getProductName());
    assertTrue(detail.hasUnitPrice());
    assertEquals(100000L, detail.getUnitPrice().getUnits());
    assertEquals(OcopRating.OCOP_RATING_4_STAR, detail.getOcopRating());
    assertEquals(PackagingType.PACKAGING_TYPE_HOP_THIEC, detail.getPackagingType());
    assertEquals(180, detail.getShelfLifeDays());
  }

  @Test
  void getProductVariant_withoutChannel_noUnitPrice() {
    UUID productId = UUID.randomUUID();
    UUID variantId = UUID.randomUUID();
    String sku = "MX-GION-500G";

    Product product = Product.builder().id(productId).name("Mè xửng giòn Huế").build();
    ProductVariant variant =
        ProductVariant.builder()
            .id(variantId)
            .productId(productId)
            .skuCode(sku)
            .listingStatus(ListingStatus.ACTIVE)
            .build();

    when(variantMapper.findBySkuCode(sku)).thenReturn(Optional.of(variant));
    when(productMapper.findById(productId)).thenReturn(Optional.of(product));

    GetProductVariantRequest request =
        GetProductVariantRequest.newBuilder()
            .setSkuCode(sku)
            .setChannel(SalesChannel.SALES_CHANNEL_UNSPECIFIED)
            .build();

    AtomicReference<GetProductVariantResponse> responseRef = new AtomicReference<>();
    grpcService.getProductVariant(
        request,
        new StreamObserver<>() {
          @Override
          public void onNext(GetProductVariantResponse value) {
            responseRef.set(value);
          }

          @Override
          public void onError(Throwable t) {}

          @Override
          public void onCompleted() {}
        });

    GetProductVariantResponse response = responseRef.get();
    assertNotNull(response);
    assertFalse(response.getVariant().hasUnitPrice());
  }

  // ── GetTraceabilityBinding Tests ──────────────────────────────────────────

  @Test
  void getTraceabilityBinding_success() {
    UUID productId = UUID.randomUUID();
    UUID variantId = UUID.randomUUID();
    String sku = "MX-GION-500G";

    ProductVariant variant = new ProductVariant();
    variant.setId(variantId);
    variant.setProductId(productId);
    variant.setSkuCode(sku);
    variant.setVersion(3L);

    Product product = new Product();
    product.setId(productId);
    product.setVersion(2L);

    when(variantMapper.findBySkuCode(sku)).thenReturn(Optional.of(variant));
    when(productMapper.findById(productId)).thenReturn(Optional.of(product));

    AtomicReference<GetTraceabilityBindingResponse> responseRef = new AtomicReference<>();
    AtomicReference<Throwable> errorRef = new AtomicReference<>();

    grpcService.getTraceabilityBinding(
        GetTraceabilityBindingRequest.newBuilder()
            .setProductId(productId.toString())
            .setSkuCode(sku)
            .build(),
        new StreamObserver<>() {
          @Override
          public void onNext(GetTraceabilityBindingResponse value) {
            responseRef.set(value);
          }

          @Override
          public void onError(Throwable t) {
            errorRef.set(t);
          }

          @Override
          public void onCompleted() {}
        });

    assertNull(errorRef.get());
    assertNotNull(responseRef.get());
    assertEquals(productId.toString(), responseRef.get().getProductId());
    assertEquals(sku, responseRef.get().getSkuCode());
    assertEquals(variantId.toString() + ":2:3", responseRef.get().getVersionFingerprint());
    assertEquals(2L, responseRef.get().getProductVersion());
    assertEquals(3L, responseRef.get().getVariantVersion());
    assertEquals(variantId.toString(), responseRef.get().getVariantId());
  }

  @Test
  void getTraceabilityBinding_skuNotFound() {
    UUID productId = UUID.randomUUID();
    String sku = "UNKNOWN-SKU";

    when(variantMapper.findBySkuCode(sku)).thenReturn(Optional.empty());

    AtomicReference<Throwable> errorRef = new AtomicReference<>();

    grpcService.getTraceabilityBinding(
        GetTraceabilityBindingRequest.newBuilder()
            .setProductId(productId.toString())
            .setSkuCode(sku)
            .build(),
        new StreamObserver<>() {
          @Override
          public void onNext(GetTraceabilityBindingResponse value) {}

          @Override
          public void onError(Throwable t) {
            errorRef.set(t);
          }

          @Override
          public void onCompleted() {}
        });

    assertNotNull(errorRef.get());
    assertInstanceOf(StatusRuntimeException.class, errorRef.get());
    assertEquals(
        Status.Code.NOT_FOUND, ((StatusRuntimeException) errorRef.get()).getStatus().getCode());
  }

  @Test
  void getTraceabilityBinding_productMismatch() {
    UUID productId = UUID.randomUUID();
    UUID differentProductId = UUID.randomUUID();
    String sku = "MX-GION-500G";

    ProductVariant variant = new ProductVariant();
    variant.setId(UUID.randomUUID());
    variant.setProductId(differentProductId);
    variant.setSkuCode(sku);

    when(variantMapper.findBySkuCode(sku)).thenReturn(Optional.of(variant));

    AtomicReference<Throwable> errorRef = new AtomicReference<>();

    grpcService.getTraceabilityBinding(
        GetTraceabilityBindingRequest.newBuilder()
            .setProductId(productId.toString())
            .setSkuCode(sku)
            .build(),
        new StreamObserver<>() {
          @Override
          public void onNext(GetTraceabilityBindingResponse value) {}

          @Override
          public void onError(Throwable t) {
            errorRef.set(t);
          }

          @Override
          public void onCompleted() {}
        });

    assertNotNull(errorRef.get());
    assertInstanceOf(StatusRuntimeException.class, errorRef.get());
    assertEquals(
        Status.Code.NOT_FOUND, ((StatusRuntimeException) errorRef.get()).getStatus().getCode());
  }

  @Test
  void getTraceabilityBinding_invalidUuid() {
    AtomicReference<Throwable> errorRef = new AtomicReference<>();

    grpcService.getTraceabilityBinding(
        GetTraceabilityBindingRequest.newBuilder()
            .setProductId("not-a-valid-uuid")
            .setSkuCode("MX-GION-500G")
            .build(),
        new StreamObserver<>() {
          @Override
          public void onNext(GetTraceabilityBindingResponse value) {}

          @Override
          public void onError(Throwable t) {
            errorRef.set(t);
          }

          @Override
          public void onCompleted() {}
        });

    assertNotNull(errorRef.get());
    assertInstanceOf(StatusRuntimeException.class, errorRef.get());
    assertEquals(
        Status.Code.INVALID_ARGUMENT,
        ((StatusRuntimeException) errorRef.get()).getStatus().getCode());
  }
}
