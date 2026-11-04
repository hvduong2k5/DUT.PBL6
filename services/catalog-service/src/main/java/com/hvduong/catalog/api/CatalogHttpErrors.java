package com.hvduong.catalog.api;
import com.hvduong.catalog.common.exception.CatalogException;
import org.springframework.web.bind.annotation.*;
import org.springframework.http.ResponseEntity;
import java.util.Map;
import java.util.UUID;
@RestControllerAdvice
public class CatalogHttpErrors {
    @ExceptionHandler(org.springframework.web.server.ResponseStatusException.class)
    public ResponseEntity<?> status(org.springframework.web.server.ResponseStatusException error) {
        String reason=error.getReason();
        String code=reason!=null && reason.matches("[A-Z_]{3,80}")?reason:"CATALOG_REQUEST_REJECTED";
        return ResponseEntity.status(error.getStatusCode()).body(Map.of("code",code,"message","Yêu cầu Catalog chưa thể thực hiện.","requestId",UUID.randomUUID().toString()));
    }
    @ExceptionHandler(CatalogException.class)
    public ResponseEntity<?> business(CatalogException error) {
        var code=error.getErrorCode();return ResponseEntity.status(code.getHttpStatus()).body(Map.of("code",code.name(),"message",code.getUserMessage(),"requestId",UUID.randomUUID().toString()));
    }
}
