package com.hvduong.marketing;

import jakarta.servlet.http.HttpServletRequest;
import java.util.List;
import java.util.UUID;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.transaction.PlatformTransactionManager;
import org.springframework.transaction.TransactionDefinition;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1/admin/marketing/campaigns")
class CampaignController {
    record Create(String departmentId, CampaignPlan plan) {}
    record Edit(long expectedVersion, CampaignPlan plan) {}
    record Revise(UUID baseRevisionId, long expectedVersion) {}
    record Review(long expectedVersion, UUID approvalRequestId, String reason) {}

    private final ManualExecutions executions;
    private final MarketingPeers peers;
    private final CampaignCommands commands;
    private final CampaignStore store;
    private final String adminOrigin;
    private final TransactionTemplate readTransaction;

    CampaignController(MarketingPeers peers, CampaignCommands commands, CampaignStore store, ManualExecutions executions,
                       PlatformTransactionManager transactions, @Value("${marketing.admin-origin:}") String adminOrigin) {
        this.peers = peers;
        this.executions = executions;
        this.commands = commands;
        this.store = store;
        this.adminOrigin = adminOrigin;
        readTransaction = new TransactionTemplate(transactions);
        readTransaction.setReadOnly(true);
        readTransaction.setIsolationLevel(TransactionDefinition.ISOLATION_REPEATABLE_READ);
    }

    record Page(List<CampaignStore.Summary> items, long total, String nextCursor) {}

    @GetMapping
    Page list(HttpServletRequest request, @RequestParam(required = false) String departmentId,
              @RequestParam(required = false) ReviewWorkflow.State state,
              @RequestParam(required = false) String cursor, @RequestParam(defaultValue = "25") int limit) {
        var actor = staff(request);
        if (limit < 1 || limit > 100 || (departmentId != null
                && (departmentId.isBlank() || departmentId.length() > 200))) {
            throw new MarketingFailure("BAD_REQUEST", 400);
        }
        String permission = "MARKETING_CAMPAIGN_VIEW";
        boolean global = actor.can(permission, "");
        var departments = actor.grants().stream()
                .filter(grant -> permission.equals(grant.permissionCode()) && "DEPARTMENT".equals(grant.scopeType()))
                .map(StaffActor.Grant::scopeId).filter(java.util.Objects::nonNull).distinct().sorted().toList();
        if ((!global && departments.isEmpty()) || (departmentId != null && !actor.can(permission, departmentId))) {
            throw new MarketingFailure("FORBIDDEN", 403);
        }
        String stateName = state == null ? null : state.name();
        String binding = ReviewStore.hash(actor.id() + "|" + stateName + "|"
                + (departmentId == null ? "all" : "department:" + departmentId));
        var position = decodeCursor(cursor, binding);
        return readTransaction.execute(status -> {
            var rows = store.list(departments, global, departmentId, stateName,
                    position == null ? null : position.time(), position == null ? null : position.id(), limit + 1);
            var items = rows.subList(0, Math.min(limit, rows.size()));
            String next = null;
            if (rows.size() > limit) {
                var last = items.getLast();
                next = java.util.Base64.getUrlEncoder().withoutPadding().encodeToString(
                        (binding + "|" + last.updatedAt() + "|" + last.campaignId())
                                .getBytes(java.nio.charset.StandardCharsets.UTF_8));
            }
            return new Page(items, store.count(departments, global, departmentId, stateName), next);
        });
    }

    private record Cursor(java.time.Instant time, UUID id) {}

    private Cursor decodeCursor(String encoded, String binding) {
        if (encoded == null) {
            return null;
        }
        if (encoded.length() <= 2048) {
            try {
                String[] fields = new String(java.util.Base64.getUrlDecoder().decode(encoded),
                        java.nio.charset.StandardCharsets.UTF_8).split("\\|", -1);
                if (fields.length == 3 && fields[0].equals(binding)) {
                    return new Cursor(java.time.Instant.parse(fields[1]), UUID.fromString(fields[2]));
                }
            } catch (IllegalArgumentException | java.time.DateTimeException error) {
                throw new MarketingFailure("BAD_REQUEST", 400);
            }
        }
        throw new MarketingFailure("BAD_REQUEST", 400);
    }

    @PostMapping
    ResponseEntity<String> create(HttpServletRequest request, @RequestBody Create body,
                                  @RequestHeader("Idempotency-Key") String key) {
        StaffActor actor = staff(request);
        if (body.departmentId() == null || body.departmentId().isBlank()
                || body.departmentId().length() > 200 || body.plan() == null) {
            throw new MarketingFailure("VALIDATION_FAILED", 422);
        }
        return response(commands.create(actor, body.departmentId().strip(), body.plan(), key, requestId(request)));
    }

    @GetMapping("/{campaignId}")
    Object current(HttpServletRequest request, @PathVariable UUID campaignId) {
        return read(request, campaignId, null);
    }

    @GetMapping("/{campaignId}/revisions/{revisionId}")
    Object revision(HttpServletRequest request, @PathVariable UUID campaignId, @PathVariable UUID revisionId) {
        return read(request, campaignId, revisionId);
    }

    @PatchMapping("/{campaignId}/revisions/{revisionId}")
    Object edit(HttpServletRequest request, @PathVariable UUID campaignId, @PathVariable UUID revisionId,
                @RequestBody Edit body) {
        var actor = staff(request);
        checkVersion(body.expectedVersion());
        if (body.plan() == null) {
            throw new MarketingFailure("VALIDATION_FAILED", 422);
        }
        return commands.edit(actor, campaignId, revisionId, body.expectedVersion(), body.plan(), requestId(request));
    }

    @PostMapping("/{campaignId}/revisions")
    ResponseEntity<String> revise(HttpServletRequest request, @PathVariable UUID campaignId, @RequestBody Revise body,
                                  @RequestHeader("Idempotency-Key") String key) {
        var actor = staff(request);
        checkVersion(body.expectedVersion());
        if (body.baseRevisionId() == null) {
            throw new MarketingFailure("BAD_REQUEST", 400);
        }
        return response(commands.revise(actor, campaignId, body.baseRevisionId(), body.expectedVersion(),
                key, requestId(request)));
    }

    @PostMapping("/{campaignId}/revisions/{revisionId}/{action:submit|withdraw|approve|reject|request-changes}")
    ResponseEntity<String> review(HttpServletRequest request, @PathVariable UUID campaignId,
                                  @PathVariable UUID revisionId, @PathVariable String action, @RequestBody Review body,
                                  @RequestHeader("Idempotency-Key") String key) {
        var actor = staff(request);
        checkVersion(body.expectedVersion());
        var operation = ReviewWorkflow.Action.valueOf(action.replace('-', '_').toUpperCase(java.util.Locale.ROOT));
        boolean submit = operation == ReviewWorkflow.Action.SUBMIT;
        if ((!submit && body.approvalRequestId() == null) || (body.reason() != null && body.reason().length() > 200)
                || ((submit || operation == ReviewWorkflow.Action.APPROVE) && body.reason() != null)
                || (submit && body.approvalRequestId() != null)) {
            throw new MarketingFailure("BAD_REQUEST", 400);
        }
        List<ReviewStore.ProductEvidence> evidence = submit
                ? evidence(request, actor, campaignId, revisionId, body.expectedVersion()) : List.of();
        return response(commands.review(actor, campaignId, revisionId, body.expectedVersion(), operation,
                body.approvalRequestId(), body.reason(), key, requestId(request), evidence));
    }

    @GetMapping("/{campaignId}/revisions/{revisionId}/executions")
    ManualExecutions.Page executions(HttpServletRequest request, @PathVariable UUID campaignId,
                                     @PathVariable UUID revisionId) {
        return executions.list(staff(request), campaignId, revisionId);
    }

    @PostMapping("/{campaignId}/revisions/{revisionId}/executions")
    ResponseEntity<String> schedule(HttpServletRequest request, @PathVariable UUID campaignId,
                                    @PathVariable UUID revisionId, @RequestBody ManualExecutions.Schedule body,
                                    @RequestHeader("Idempotency-Key") String key) {
        var actor = staff(request);
        checkVersion(body.expectedVersion());
        return response(executions.schedule(actor, campaignId, revisionId, body, key, requestId(request)));
    }

    @PostMapping("/{campaignId}/revisions/{revisionId}/executions/{executionId}/actions")
    ResponseEntity<String> act(HttpServletRequest request, @PathVariable UUID campaignId,
                               @PathVariable UUID revisionId, @PathVariable UUID executionId,
                               @RequestBody ManualExecutions.Change body, @RequestHeader("Idempotency-Key") String key) {
        var actor = staff(request);
        checkVersion(body.expectedVersion());
        return response(executions.act(actor, campaignId, revisionId, executionId, body, key, requestId(request)));
    }


    private List<ReviewStore.ProductEvidence> evidence(HttpServletRequest request, StaffActor actor, UUID campaignId,
                                                       UUID revisionId, long expectedVersion) {
        var products = readTransaction.execute(status -> {
            var campaign = store.campaign(campaignId, false);
            if (!actor.can("MARKETING_PLAN_SUBMIT", campaign.departmentId())) {
                throw new MarketingFailure("FORBIDDEN", 403);
            }
            var revision = store.revision(campaignId, revisionId);
            if (!campaign.currentRevisionId().equals(revisionId) || revision.version() != expectedVersion
                    || revision.state() != ReviewWorkflow.State.DRAFT) {
                return List.<CampaignPlan.Product>of();
            }
            revision.plan().validateForSubmission();
            return revision.plan().products();
        });
        String bearer = request.getHeader("Authorization");
        if (!products.isEmpty() && (bearer == null || bearer.isBlank())) {
            throw new MarketingFailure("CATALOG_AUTHORIZATION_REQUIRED", 403);
        }
        return products.stream().map(product -> peers.verifyProduct(product, bearer)).toList();
    }

    private Object read(HttpServletRequest request, UUID campaignId, UUID revisionId) {
        var actor = staff(request);
        return readTransaction.execute(status -> {
            var campaign = store.campaign(campaignId, false);
            if (!actor.can("MARKETING_CAMPAIGN_VIEW", campaign.departmentId())) {
                throw new MarketingFailure("RESOURCE_NOT_FOUND", 404);
            }
            return commands.view(campaign, store.revision(campaignId,
                    revisionId == null ? campaign.currentRevisionId() : revisionId));
        });
    }

    private StaffActor staff(HttpServletRequest request) {
        String bearer = request.getHeader("Authorization");
        String origin = request.getHeader("Origin");
        String session = null;
        if (bearer == null && request.getCookies() != null) {
            for (var cookie : request.getCookies()) {
                if ("OMAMA_SESSION".equals(cookie.getName())) {
                    session = cookie.getValue();
                    break;
                }
            }
        }
        if (session != null) {
            if (adminOrigin.isBlank() || (!"GET".equals(request.getMethod()) && !adminOrigin.equals(origin))
                    || (origin != null && !adminOrigin.equals(origin))) {
                throw new MarketingFailure("FORBIDDEN", 403);
            }
            if (!session.matches("[0-9a-fA-F-]{36}")) {
                throw new MarketingFailure("AUTHENTICATION_REQUIRED", 401);
            }
        }
        return peers.staff(bearer, session, origin);
    }

    static String requestId(HttpServletRequest request) {
        String id = (String) request.getAttribute("marketingRequestId");
        if (id == null) {
            id = UUID.randomUUID().toString();
            request.setAttribute("marketingRequestId", id);
        }
        return id;
    }

    private void checkVersion(long version) {
        if (version < 1 || version > 9_007_199_254_740_991L) {
            throw new MarketingFailure("BAD_REQUEST", 400);
        }
    }

    private ResponseEntity<String> response(ReviewStore.Receipt receipt) {
        return ResponseEntity.status(receipt.status()).contentType(MediaType.APPLICATION_JSON).body(receipt.body());
    }
}
