package com.hvduong.catalog.application.dto.request;

import jakarta.validation.constraints.Size;
import lombok.Data;

import java.util.UUID;

/** Input DTO để cập nhật Product — patch-style (chỉ field khác null mới được cập nhật). */
@Data
public class UpdateProductRequest {

    private UUID categoryId;

    @Size(max = 500)
    private String name;

    @Size(max = 500)
    private String slug;

    @Size(max = 5000)
    private String description;

    @Size(max = 2000)
    private String ingredients;

    @Size(max = 1000)
    private String storageGuide;

    private Integer ocopStar;

    @Size(max = 100)
    private String ocopCertificateNo;

    @Size(max = 5000)
    private String story;

    /** Bắt buộc để optimistic locking. */
    private long version;
}
