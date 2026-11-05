package com.hvduong.catalog.repository.mybatis.projection;

import lombok.Data;

/**
 * Projection cho thống kê Product theo trạng thái listing trong DB.
 */
@Data
public class AdminStatusCountRow {
    private String status;
    private long count;
}
