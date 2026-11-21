package com.hvduong.catalog.grpc;

import com.hvduong.catalog.domain.entity.ChannelPrice;
import com.hvduong.catalog.domain.entity.Product;
import com.hvduong.catalog.domain.entity.ProductVariant;

import java.math.BigDecimal;
import java.util.Locale;

import org.springframework.stereotype.Component;

import com.omamx.proto.catalog.v1.OcopRating;
import com.omamx.proto.catalog.v1.PackagingType;
import com.omamx.proto.catalog.v1.ProductVariantDetail;
import com.omamx.proto.common.v1.Money;
import com.omamx.proto.common.v1.Weight;
import com.omamx.proto.common.v1.WeightUnit;

/** Mapper giữa Protobuf generated classes của omamx.catalog.v1 và Domain Entities. */
@Component
public class CatalogGrpcMapper {

  public ProductVariantDetail toVariantDetailProto(
      ProductVariant variant, Product product, ChannelPrice price) {
    var builder =
        ProductVariantDetail.newBuilder()
            .setVariantId(variant.getId() != null ? variant.getId().toString() : "")
            .setSkuCode(variant.getSkuCode() != null ? variant.getSkuCode() : "")
            .setProductId(variant.getProductId() != null ? variant.getProductId().toString() : "")
            .setProductName(product != null && product.getName() != null ? product.getName() : "")
            .setVariantName(variant.getVariantName() != null ? variant.getVariantName() : "")
            .setPackagingType(toPackagingTypeProto(variant.getPackagingType()))
            .setOcopRating(
                product != null
                    ? toOcopRatingProto(product.getOcopStar())
                    : OcopRating.OCOP_RATING_UNSPECIFIED)
            .setIsActive(
                variant.getListingStatus() != null
                    && "ACTIVE".equalsIgnoreCase(variant.getListingStatus().getValue()));

    if (variant.getShelfLifeDays() != null) {
      builder.setShelfLifeDays(variant.getShelfLifeDays());
    }

    if (price != null) {
      builder.setUnitPrice(
          Money.newBuilder()
              .setCurrencyCode(price.getCurrencyCode() != null ? price.getCurrencyCode() : "VND")
              .setUnits(price.getAmountUnits())
              .setNanos(price.getAmountNanos())
              .build());
    }

    if (variant.getWeightValue() != null) {
      builder.setNetWeight(toWeightProto(variant.getWeightValue(), variant.getWeightUnit()));
    }

    return builder.build();
  }

  public Money toMoneyProto(long units, int nanos, String currencyCode) {
    return Money.newBuilder()
        .setCurrencyCode(currencyCode != null ? currencyCode : "VND")
        .setUnits(units)
        .setNanos(nanos)
        .build();
  }

  public Weight toWeightProto(BigDecimal weightValue, String unit) {
    WeightUnit weightUnit = WeightUnit.WEIGHT_UNIT_GRAM;
    if (unit != null && unit.toUpperCase(Locale.ROOT).contains("KG")) {
      weightUnit = WeightUnit.WEIGHT_UNIT_KILOGRAM;
    }
    return Weight.newBuilder()
        .setDecimalValue(weightValue != null ? weightValue.toPlainString() : "0")
        .setUnit(weightUnit)
        .build();
  }

  public com.hvduong.catalog.common.enums.SalesChannel toSalesChannel(
      com.omamx.proto.catalog.v1.SalesChannel protoChannel) {
    if (protoChannel == null) return null;
    return switch (protoChannel) {
      case SALES_CHANNEL_D2C_WEB -> com.hvduong.catalog.common.enums.SalesChannel.D2C_WEB;
      case SALES_CHANNEL_MOBILE_APP -> com.hvduong.catalog.common.enums.SalesChannel.MOBILE_APP;
      case SALES_CHANNEL_B2B_WHOLESALE -> com.hvduong.catalog.common.enums.SalesChannel
          .B2B_WHOLESALE;
      case SALES_CHANNEL_POS_QUAY -> com.hvduong.catalog.common.enums.SalesChannel.POS_QUAY;
      case SALES_CHANNEL_SHOPEE -> com.hvduong.catalog.common.enums.SalesChannel.SHOPEE;
      case SALES_CHANNEL_TIKTOK -> com.hvduong.catalog.common.enums.SalesChannel.TIKTOK;
      default -> null;
    };
  }

  public PackagingType toPackagingTypeProto(String type) {
    if (type == null) return PackagingType.PACKAGING_TYPE_UNSPECIFIED;
    return switch (type.toUpperCase(Locale.ROOT)) {
      case "HOP_GIAY_KRAFT" -> PackagingType.PACKAGING_TYPE_HOP_GIAY_KRAFT;
      case "HOP_THIEC" -> PackagingType.PACKAGING_TYPE_HOP_THIEC;
      case "GOI_HUT_CHAN_KHONG" -> PackagingType.PACKAGING_TYPE_GOI_HUT_CHAN_KHONG;
      case "HOP_QUA_TANG" -> PackagingType.PACKAGING_TYPE_HOP_QUA_TANG;
      case "DANG_ROI" -> PackagingType.PACKAGING_TYPE_DANG_ROI;
      default -> PackagingType.PACKAGING_TYPE_UNSPECIFIED;
    };
  }

  public OcopRating toOcopRatingProto(Integer stars) {
    if (stars == null) return OcopRating.OCOP_RATING_UNSPECIFIED;
    return switch (stars) {
      case 3 -> OcopRating.OCOP_RATING_3_STAR;
      case 4 -> OcopRating.OCOP_RATING_4_STAR;
      case 5 -> OcopRating.OCOP_RATING_5_STAR;
      default -> OcopRating.OCOP_RATING_UNSPECIFIED;
    };
  }
}
