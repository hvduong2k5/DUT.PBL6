package com.hvduong.identity.security;

import com.hvduong.identity.application.account.AuthenticationService;
import com.hvduong.identity.infrastructure.session.BffSession;
import com.hvduong.identity.infrastructure.session.BffSessionStore;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.Cookie;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletRequestWrapper;
import jakarta.servlet.http.HttpServletResponse;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.annotation.Lazy;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpHeaders;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;
import java.util.*;

@Slf4j
@Component
@RequiredArgsConstructor
public class BffSessionAuthFilter extends OncePerRequestFilter {

    private final BffSessionStore bffSessionStore;

    @Lazy
    private final AuthenticationService authenticationService;

    @Value("${app.frontend-url:http://localhost:3000}")
    private String frontendUrl;

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain filterChain)
            throws ServletException, IOException {

        response.setHeader("Cache-Control", "no-store");
        if (request.getRequestURI().equals("/api/v1/public/auth/callback")) {
            String binding = cookie(request, "OMAMA_AUTH_STATE");
            String state = request.getParameter("state");
            if (binding == null || state == null || !java.security.MessageDigest.isEqual(
                    binding.getBytes(java.nio.charset.StandardCharsets.UTF_8), state.getBytes(java.nio.charset.StandardCharsets.UTF_8))) {
                response.sendError(400, "Invalid browser authentication state");
                return;
            }
        }
        if (extractSessionCookie(request) != null && !Set.of("GET", "HEAD", "OPTIONS").contains(request.getMethod())) {
            String origin = request.getHeader("Origin");
            try {
                java.net.URI source = java.net.URI.create(origin == null ? "" : origin);
                java.net.URI allowed = java.net.URI.create(frontendUrl);
                if (!Objects.equals(source.getScheme(), allowed.getScheme()) || !Objects.equals(source.getAuthority(), allowed.getAuthority())) {
                    response.sendError(403, "Invalid browser origin");
                    return;
                }
            } catch (IllegalArgumentException ex) {
                response.sendError(403, "Invalid browser origin");
                return;
            }
        }

        String authHeader = request.getHeader(HttpHeaders.AUTHORIZATION);
        if (authHeader != null && authHeader.toLowerCase().startsWith("bearer ")) {
            filterChain.doFilter(request, response);
            return;
        }

        String sessionCookie = extractSessionCookie(request);
        if (sessionCookie != null && !sessionCookie.isBlank()) {
            try {
                UUID sessionId = UUID.fromString(sessionCookie.trim());
                Optional<BffSession> sessionOpt = bffSessionStore.getSession(sessionId);

                if (sessionOpt.isPresent()) {
                    BffSession session = sessionOpt.get();
                    if (!session.isRefreshTokenExpired()) {
                        if (session.isAccessTokenExpired()) {
                            try {
                                session = authenticationService.refreshSession(sessionId);
                            } catch (Exception ex) {
                                log.debug("Failed to auto-refresh session in filter: {}", ex.getMessage());
                                session = null;
                            }
                        }

                        if (session != null && session.getAccessToken() != null) {
                            final String token = session.getAccessToken();
                            HttpServletRequest wrappedRequest = new HttpServletRequestWrapper(request) {
                                @Override
                                public String getHeader(String name) {
                                    if (HttpHeaders.AUTHORIZATION.equalsIgnoreCase(name)) {
                                        return "Bearer " + token;
                                    }
                                    return super.getHeader(name);
                                }

                                @Override
                                public Enumeration<String> getHeaders(String name) {
                                    if (HttpHeaders.AUTHORIZATION.equalsIgnoreCase(name)) {
                                        return Collections.enumeration(Collections.singletonList("Bearer " + token));
                                    }
                                    return super.getHeaders(name);
                                }

                                @Override
                                public Enumeration<String> getHeaderNames() {
                                    List<String> names = Collections.list(super.getHeaderNames());
                                    if (names.stream().noneMatch(n -> n.equalsIgnoreCase(HttpHeaders.AUTHORIZATION))) {
                                        names.add(HttpHeaders.AUTHORIZATION);
                                    }
                                    return Collections.enumeration(names);
                                }
                            };
                            filterChain.doFilter(wrappedRequest, response);
                            return;
                        }
                    }
                }
            } catch (IllegalArgumentException ex) {
                log.debug("Invalid session cookie UUID format: {}", sessionCookie);
            }
        }

        filterChain.doFilter(request, response);
    }

    private String extractSessionCookie(HttpServletRequest request) {
        return cookie(request, "OMAMA_SESSION");
    }

    private String cookie(HttpServletRequest request, String name) {
        if (request.getCookies() == null) return null;
        for (Cookie cookie : request.getCookies()) {
            if (name.equals(cookie.getName())) {
                return cookie.getValue();
            }
        }
        return null;
    }
}

