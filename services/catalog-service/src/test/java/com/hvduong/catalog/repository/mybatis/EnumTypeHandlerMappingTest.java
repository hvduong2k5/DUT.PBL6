package com.hvduong.catalog.repository.mybatis;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

import com.hvduong.catalog.common.enums.ApprovalStatus;
import com.hvduong.catalog.common.enums.ListingStatus;
import com.hvduong.catalog.common.enums.SalesChannel;
import com.hvduong.catalog.config.UuidTypeHandler;

import java.io.InputStream;
import java.sql.ResultSet;
import java.util.List;

import org.apache.ibatis.builder.xml.XMLMapperBuilder;
import org.apache.ibatis.io.Resources;
import org.apache.ibatis.mapping.ResultMap;
import org.apache.ibatis.mapping.ResultMapping;
import org.apache.ibatis.session.Configuration;
import org.apache.ibatis.type.EnumTypeHandler;
import org.apache.ibatis.type.TypeHandler;
import org.apache.ibatis.type.TypeHandlerRegistry;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

/**
 * Regression test cho lỗi MyBatis enum type handler bị tráo: khai báo tường minh {@code
 * typeHandler="org.apache.ibatis.type.EnumTypeHandler"} khiến mọi enum dùng chung instance đầu tiên
 * được đăng ký (ví dụ {@code EnumTypeHandler<SalesChannel>}), nên cột {@code approval_status} bị đọc
 * thành {@code SalesChannel.APPROVED} và ném IllegalArgumentException.
 *
 * <p>Test nạp chính các mapper XML thật trong {@code src/main/resources/mapper/sql}.
 */
class EnumTypeHandlerMappingTest {

  private static final String MAPPER_PATH = "mapper/sql/";

  @SuppressWarnings("unchecked")
  private static final Class<? extends TypeHandler<?>> ENUM_TYPE_HANDLER =
      (Class<? extends TypeHandler<?>>) (Class<?>) EnumTypeHandler.class;

  private final Configuration configuration = new Configuration();
  private final TypeHandlerRegistry registry = configuration.getTypeHandlerRegistry();

  @BeforeEach
  void setUp() {
    // application.yml khai báo type-handlers-package: com.hvduong.catalog.config
    registry.register(new UuidTypeHandler());
  }

  @Test
  void resultMapsResolveEnumsByPropertyTypeInsteadOfSharedEnumTypeHandler() throws Exception {
    // Thứ tự mybatis.mapper-locations (classpath:mapper/sql/*.xml) đọc ChannelPriceMapper trước.
    // Khi parse statement insert với parameterType=ChannelPrice, ParameterMapping của #{channel}
    // có javaType=SalesChannel nên MyBatis đăng ký EnumTypeHandler<SalesChannel> vào
    // allTypeHandlersMap - đây chính là điều kiện gây lỗi cho các resultMap khai báo typeHandler
    // tường minh "org.apache.ibatis.type.EnumTypeHandler" được parse sau đó.
    parseMapper("ChannelPriceMapper.xml");
    assertThat(registry.getMappingTypeHandler(ENUM_TYPE_HANDLER)).isNotNull();

    parseMapper("ProductMapper.xml");
    parseMapper("ProductVariantMapper.xml");

    assertThat(readEnum("PriceResultMap", "channel", SalesChannel.D2C_WEB.name()))
        .isEqualTo(SalesChannel.D2C_WEB);
    assertThat(readEnum("ProductResultMap", "approvalStatus", ApprovalStatus.APPROVED.name()))
        .isEqualTo(ApprovalStatus.APPROVED);
    assertThat(readEnum("ProductResultMap", "listingStatus", ListingStatus.ACTIVE.name()))
        .isEqualTo(ListingStatus.ACTIVE);
    assertThat(readEnum("VariantResultMap", "listingStatus", ListingStatus.ACTIVE.name()))
        .isEqualTo(ListingStatus.ACTIVE);
  }

  private void parseMapper(String fileName) throws Exception {
    String resource = MAPPER_PATH + fileName;
    try (InputStream xml = Resources.getResourceAsStream(resource)) {
      new XMLMapperBuilder(xml, configuration, resource, configuration.getSqlFragments()).parse();
    }
  }

  private Object readEnum(String resultMapId, String property, String columnValue) throws Exception {
    ResultMapping mapping = findMapping(findResultMap(resultMapId), property);
    TypeHandler<?> typeHandler = mapping.getTypeHandler();
    if (typeHandler == null) {
      // Không khai báo typeHandler: MyBatis tự chọn theo javaType của property.
      typeHandler = registry.getTypeHandler(mapping.getJavaType());
    }
    assertThat(typeHandler).as("type handler cho %s", property).isNotNull();

    ResultSet rows = mock(ResultSet.class);
    when(rows.getString(mapping.getColumn())).thenReturn(columnValue);
    return typeHandler.getResult(rows, mapping.getColumn());
  }

  private ResultMap findResultMap(String resultMapId) {
    return configuration.getResultMapNames().stream()
        .filter(name -> name.endsWith("." + resultMapId))
        .findFirst()
        .map(configuration::getResultMap)
        .orElseThrow(() -> new AssertionError("Không tìm thấy resultMap " + resultMapId));
  }

  private static ResultMapping findMapping(ResultMap resultMap, String property) {
    List<ResultMapping> mappings = resultMap.getResultMappings();
    return mappings.stream()
        .filter(mapping -> property.equals(mapping.getProperty()))
        .findFirst()
        .orElseThrow(() -> new AssertionError("Không tìm thấy mapping cho " + property));
  }
}
