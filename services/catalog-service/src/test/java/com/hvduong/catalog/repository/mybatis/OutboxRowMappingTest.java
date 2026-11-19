package com.hvduong.catalog.repository.mybatis;

import static org.assertj.core.api.Assertions.assertThat;

import com.hvduong.catalog.config.UuidTypeHandler;
import com.hvduong.catalog.infrastructure.outbox.OutboxRow;

import java.io.InputStream;
import java.lang.reflect.RecordComponent;
import java.util.Arrays;
import java.util.List;

import org.apache.ibatis.builder.xml.XMLMapperBuilder;
import org.apache.ibatis.io.Resources;
import org.apache.ibatis.mapping.ResultMap;
import org.apache.ibatis.mapping.ResultMapping;
import org.apache.ibatis.session.Configuration;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

/**
 * Regression test cho F-21: {@code OutboxRowMap} khai {@code javaType="int"} cho cột {@code
 * schema_version}. Trong MyBatis, alias {@code "int"} trỏ tới {@code Integer.class} trong khi
 * record {@link OutboxRow} khai component {@code int} nguyên thủy — constructor lookup thất bại
 * với {@code NoSuchMethodException} ("Error instantiating class … OutboxRow"), khiến outbox
 * publisher không đọc được dòng nào và sự kiện bị kẹt. Alias đúng cho kiểu nguyên thủy là {@code
 * "_int"}.
 *
 * <p>Test nạp mapper XML thật và đối chiếu từng javaType trong {@code <constructor>} với đúng
 * record component theo thứ tự — đây là phép tra cứu MyBatis thực hiện khi khởi tạo OutboxRow.
 */
class OutboxRowMappingTest {

  private final Configuration configuration = new Configuration();

  @BeforeEach
  void setUp() {
    // application.yml khai báo type-handlers-package: com.hvduong.catalog.config
    configuration.getTypeHandlerRegistry().register(new UuidTypeHandler());
  }

  @Test
  void outboxRowConstructorArgsMatchRecordComponentsExactly() throws Exception {
    String resource = "mapper/sql/OutboxEventMapper.xml";
    try (InputStream xml = Resources.getResourceAsStream(resource)) {
      new XMLMapperBuilder(xml, configuration, resource, configuration.getSqlFragments()).parse();
    }

    ResultMap resultMap =
        configuration.getResultMapNames().stream()
            .filter(name -> name.endsWith(".OutboxRowMap"))
            .findFirst()
            .map(configuration::getResultMap)
            .orElseThrow(() -> new AssertionError("Không tìm thấy resultMap OutboxRowMap"));

    List<Class<?>> declaredTypes =
        resultMap.getConstructorResultMappings().stream().map(ResultMapping::getJavaType).toList();
    List<Class<?>> recordTypes =
        Arrays.stream(OutboxRow.class.getRecordComponents())
            .map(RecordComponent::getType)
            .toList();

    assertThat(declaredTypes)
        .as("javaType khai trong <constructor> phải khớp chính xác record component (kể cả int nguyên thủy)")
        .containsExactlyElementsOf(recordTypes);
    // Phép tra cứu MyBatis thực hiện lúc runtime — trước bản sửa nó ném NoSuchMethodException.
    assertThat(OutboxRow.class.getDeclaredConstructor(declaredTypes.toArray(new Class<?>[0])))
        .isNotNull();
  }
}
