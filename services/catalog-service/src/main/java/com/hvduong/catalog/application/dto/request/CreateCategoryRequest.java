package com.hvduong.catalog.application.dto.request;

import java.util.UUID;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.PositiveOrZero;
import jakarta.validation.constraints.Size;

public record CreateCategoryRequest(
    @NotBlank @Size(max = 255) String name,
    @NotBlank @Size(max = 255) @Pattern(regexp = "[a-z0-9]+(?:-[a-z0-9]+)*") String slug,
    UUID parentId,
    @Size(max = 4000) String description,
    @Size(max = 2048) String imageUrl,
    @PositiveOrZero Integer position) {

  public CreateCategoryRequest(
      String name, String slug, UUID parentId, String description, Integer position) {
    this(name, slug, parentId, description, null, position);
  }
}
