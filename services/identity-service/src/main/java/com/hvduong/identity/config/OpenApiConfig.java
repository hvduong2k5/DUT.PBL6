package com.hvduong.identity.config;

import io.swagger.v3.oas.models.Components;
import io.swagger.v3.oas.models.OpenAPI;
import io.swagger.v3.oas.models.info.Info;
import io.swagger.v3.oas.models.info.License;
import io.swagger.v3.oas.models.security.SecurityRequirement;
import io.swagger.v3.oas.models.security.SecurityScheme;
import io.swagger.v3.oas.models.servers.Server;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.Collections;

@Configuration
public class OpenApiConfig {

    @Value("${app.openapi.server-url:}")
    private String serverUrl;

    @Value("${app.openapi.url:${app.frontend-url:http://localhost:3000}}")
    private String appUrl;

    @Bean
    public OpenAPI customOpenAPI() {
        final String securitySchemeName = "bearerAuth";
        OpenAPI openAPI = new OpenAPI()
                .info(new Info()
                        .title("OMAMA PBL6 - Identity & IAM Service API")
                        .version("v1.0.0")
                        .description("REST API documentation for Identity Management, RBAC, User Lifecycle and Authentication.")
                        .license(new License().name("Apache 2.0").url(appUrl)))
                .addSecurityItem(new SecurityRequirement().addList(securitySchemeName))
                .components(new Components()
                        .addSecuritySchemes(securitySchemeName,
                                new SecurityScheme()
                                        .name(securitySchemeName)
                                        .type(SecurityScheme.Type.HTTP)
                                        .scheme("bearer")
                                        .bearerFormat("JWT")));

        if (serverUrl != null && !serverUrl.isBlank()) {
            openAPI.setServers(Collections.singletonList(new Server().url(serverUrl).description("API Server")));
        }

        return openAPI;
    }
}

