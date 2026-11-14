package com.hvduong.marketing;

import java.util.List;
import java.util.UUID;
import java.util.function.Supplier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
class CampaignCommands {
    private final CampaignStore campaigns;
    private final ReviewStore reviews;
    private final Long budgetCeiling;

    CampaignCommands(CampaignStore campaigns, ReviewStore reviews,
                     @Value("${marketing.budget-ceiling-vnd:}") Long budgetCeiling) {
        this.campaigns = campaigns;
        this.reviews = reviews;
        this.budgetCeiling = budgetCeiling;
        if (budgetCeiling != null && budgetCeiling < 0) {
            throw new IllegalArgumentException("Marketing budget ceiling must be nonnegative");
        }
    }

    @Transactional
    public ReviewStore.Receipt create(StaffActor actor, String department, CampaignPlan plan, String key, String requestId) {
        require(actor, "PLAN_EDIT", department);
        require(actor, "BUDGET_EDIT", department);
        return keyed(actor, department, "CREATE", key, List.of(plan), 201, () -> {
            UUID campaignId = UUID.randomUUID();
            UUID revisionId = UUID.randomUUID();
            campaigns.createCampaign(campaignId, department, revisionId);
            campaigns.insertRevision(campaignId, revisionId, null, 1, actor.id(), plan);
            var campaign = campaigns.campaign(campaignId, false);
            var revision = campaigns.revision(campaignId, revisionId);
            audit(actor, campaign, revision, null, "CREATE", requestId, null, null, List.of("plan"));
            return view(campaign, revision);
        });
    }

    @Transactional
    public Object edit(StaffActor actor, UUID campaignId, UUID revisionId, long expectedVersion,
                       CampaignPlan plan, String requestId) {
        var campaign = campaigns.campaign(campaignId, true);
        require(actor, "PLAN_EDIT", campaign.departmentId());
        var before = current(campaign, revisionId);
        requireVersion(before, expectedVersion);
        if (before.state() != ReviewWorkflow.State.DRAFT) {
            throw new MarketingFailure("STATE_CONFLICT", 409);
        }
        if (before.plan().plannedBudgetVnd() != plan.plannedBudgetVnd()) {
            require(actor, "BUDGET_EDIT", campaign.departmentId());
        }
        campaigns.replacePlan(revisionId, plan);
        campaigns.touch(campaignId);
        var after = campaigns.revision(campaignId, revisionId);
        audit(actor, campaign, after, null, "EDIT", requestId, before.state(), null, List.of("plan"));
        return view(campaign, after);
    }

    @Transactional
    public ReviewStore.Receipt revise(StaffActor actor, UUID campaignId, UUID baseId, long expectedVersion,
                                     String key, String requestId) {
        var campaign = campaigns.campaign(campaignId, true);
        require(actor, "PLAN_EDIT", campaign.departmentId());
        return keyed(actor, campaign.departmentId(), "REVISE", key, List.of(campaignId, baseId, expectedVersion), 201, () -> {
            var base = current(campaign, baseId);
            requireVersion(base, expectedVersion);
            if (base.state() == ReviewWorkflow.State.DRAFT || base.state() == ReviewWorkflow.State.IN_REVIEW
                    || campaigns.hasUnfinishedExecutions(campaignId)) {
                throw new MarketingFailure("STATE_CONFLICT", 409);
            }
            UUID id = UUID.randomUUID();
            campaigns.insertRevision(campaignId, id, baseId, campaign.nextRevisionNumber(), actor.id(), base.plan());
            campaigns.makeCurrent(campaignId, id);
            var revision = campaigns.revision(campaignId, id);
            audit(actor, campaign, revision, null, "REVISE", requestId, null, null, List.of("plan"));
            return view(campaign, revision);
        });
    }

    @Transactional
    public ReviewStore.Receipt review(StaffActor actor, UUID campaignId, UUID revisionId, long expectedVersion,
                                      ReviewWorkflow.Action action, UUID approvalId, String reason, String key,
                                      String requestId, List<ReviewStore.ProductEvidence> evidence) {
        var campaign = campaigns.campaign(campaignId, true);
        boolean submit = action == ReviewWorkflow.Action.SUBMIT;
        String permission = submit || action == ReviewWorkflow.Action.WITHDRAW ? "PLAN_SUBMIT" : "PLAN_APPROVE";
        require(actor, permission, campaign.departmentId());
        String normalizedReason = reason == null ? "" : reason.strip();
        List<Object> payload = List.of(campaignId, revisionId, expectedVersion,
                approvalId == null ? "" : approvalId, normalizedReason);
        return keyed(actor, campaign.departmentId(), action.name(), key, payload, 200, () -> {
            var before = current(campaign, revisionId);
            var open = reviews.openRequest(revisionId);
            if (submit && open != null && open.submittedVersion() == expectedVersion) {
                return view(campaign, before);
            }
            var workflow = new ReviewWorkflow(before.state(), before.version(), before.createdBy(),
                    open == null ? null : open.submittedBy(), open == null ? null : open.id(),
                    open == null ? before.plan().plannedBudgetVnd() : open.snapshot().submittedPlan().plannedBudgetVnd());
            var transition = workflow.apply(action, expectedVersion, actor.id(), approvalId, normalizedReason,
                    open == null ? null : open.snapshot().budgetCeilingVnd());
            UUID request = approvalId;
            if (submit) {
                before.plan().validateForSubmission();
                var expectedProducts = before.plan().products().stream().map(CampaignPlan.Product::productId).sorted().toList();
                var verifiedProducts = evidence.stream().map(ReviewStore.ProductEvidence::productId).sorted().toList();
                if (!expectedProducts.equals(verifiedProducts)) {
                    throw new MarketingFailure("VALIDATION_FAILED", 422);
                }
                request = UUID.randomUUID();
                reviews.open(campaignId, revisionId, request, expectedVersion, actor.id(),
                        new ReviewStore.Snapshot(before.plan(), List.copyOf(evidence), budgetCeiling));
            } else {
                String closed = action == ReviewWorkflow.Action.WITHDRAW ? "WITHDRAWN" : transition.state().name();
                reviews.close(request, closed, actor.id(), normalizedReason.isEmpty() ? null : normalizedReason,
                        transition.approvedBudgetVnd());
            }
            campaigns.changeState(revisionId, transition);
            campaigns.touch(campaignId);
            var after = campaigns.revision(campaignId, revisionId);
            audit(actor, campaign, after, request, action.name(), requestId, before.state(),
                    normalizedReason.isEmpty() ? null : normalizedReason, List.of("state"));
            return view(campaign, after);
        });
    }

    ReviewStore.Receipt keyed(StaffActor actor, String department, String action, String key, Object payload,
                                      int status, Supplier<Object> write) {
        if (key == null || !key.matches("[!-~]{16,128}")) {
            throw new MarketingFailure("BAD_REQUEST", 400);
        }
        String hash = ReviewStore.hash(reviews.encode(payload));
        reviews.lockKey(actor.id(), department, action, key);
        var previous = reviews.receipt(actor.id(), department, action, key);
        if (previous != null) {
            if (!hash.equals(previous.hash())) {
                throw new MarketingFailure("IDEMPOTENCY_KEY_REUSED", 409);
            }
            return previous;
        }
        var receipt = new ReviewStore.Receipt(hash, reviews.encode(write.get()), status);
        reviews.saveReceipt(actor.id(), department, action, key, receipt);
        return receipt;
    }

    private CampaignStore.Revision current(CampaignStore.Campaign campaign, UUID revisionId) {
        var revision = campaigns.revision(campaign.id(), revisionId);
        if (!campaign.currentRevisionId().equals(revisionId)) {
            throw new MarketingFailure("STATE_CONFLICT", 409);
        }
        return revision;
    }

    void requireVersion(CampaignStore.Revision revision, long version) {
        if (revision.version() != version) {
            throw new MarketingFailure("STALE_REVISION", 409);
        }
    }

    private void require(StaffActor actor, String permission, String department) {
        if (!actor.can("MARKETING_" + permission, department)) {
            throw new MarketingFailure("FORBIDDEN", 403);
        }
    }

    private void audit(StaffActor actor, CampaignStore.Campaign campaign, CampaignStore.Revision revision,
                       UUID approval, String action, String requestId, ReviewWorkflow.State before,
                       String reason, List<String> changed) {
        reviews.audit(campaign, revision.id(), approval, action, actor.id(), requestId,
                before == null ? null : before.name(), revision.state().name(), revision.version(), reason, changed);
    }

    Object view(CampaignStore.Campaign campaign, CampaignStore.Revision revision) {
        var result = new java.util.LinkedHashMap<String, Object>();
        result.put("campaignId", campaign.id());
        result.put("departmentId", campaign.departmentId());
        result.put("revisionId", revision.id());
        result.put("revisionNumber", revision.number());
        result.put("version", revision.version());
        result.put("state", revision.state());
        result.put("createdBy", revision.createdBy());
        result.put("plan", revision.plan());
        var approval = reviews.latestView(revision.id());
        if (approval != null) {
            result.put("approvalRequest", approval);
        }
        return result;
    }
}
