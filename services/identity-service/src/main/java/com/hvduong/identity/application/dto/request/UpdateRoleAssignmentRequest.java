package com.hvduong.identity.application.dto.request;

import io.swagger.v3.oas.annotations.media.Schema;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.OffsetDateTime;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class UpdateRoleAssignmentRequest {

    @Schema(description = "Thời hạn hết hạn mới (null nếu không thay đổi hoặc dùng clearValidUntil để xóa hạn)")
    private OffsetDateTime validUntil;

    @Schema(description = "Đặt thành true nếu muốn xóa thời hạn kết thúc (chuyển sang vô thời hạn)")
    private Boolean clearValidUntil;

    @Schema(description = "Cập nhật lý do hoặc ghi chú phân công vai trò")
    private String reason;

    @Schema(description = "Phiên bản kỳ vọng cho optimistic locking")
    private Long expectedVersion;
}
