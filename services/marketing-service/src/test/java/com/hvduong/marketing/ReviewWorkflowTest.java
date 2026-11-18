package com.hvduong.marketing;

import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.util.UUID;
import org.junit.jupiter.api.Test;

class ReviewWorkflowTest {
    private final UUID author = UUID.randomUUID();
    private final UUID submitter = UUID.randomUUID();
    private final UUID reviewer = UUID.randomUUID();
    private final UUID request = UUID.randomUUID();
    private final ReviewWorkflow review = new ReviewWorkflow(
            ReviewWorkflow.State.IN_REVIEW, 2, author, submitter, request, 100);

    @Test
    void approvesFrozenBudgetWithMatchingVersionAndRequest() {
        var result = review.apply(ReviewWorkflow.Action.APPROVE, 2, reviewer, request, null, 100L);
        assertThat(result.state()).isEqualTo(ReviewWorkflow.State.APPROVED);
        assertThat(result.version()).isEqualTo(3);
        assertThat(result.approvedBudgetVnd()).isEqualTo(100L);
    }

    @Test
    void rejectsSelfApprovalAndBudgetsAboveCeiling() {
        for (UUID actor : new UUID[]{author, submitter}) {
            var rejection = assertThrows(ReviewWorkflow.Rejected.class,
                    () -> review.apply(ReviewWorkflow.Action.APPROVE, 2, actor, request, null, null));
            assertThat(rejection.error()).isEqualTo(ReviewWorkflow.Error.FORBIDDEN);
        }
        var rejection = assertThrows(ReviewWorkflow.Rejected.class,
                () -> review.apply(ReviewWorkflow.Action.APPROVE, 2, reviewer, request, null, 99L));
        assertThat(rejection.error()).isEqualTo(ReviewWorkflow.Error.FORBIDDEN);
    }

    @Test
    void rejectsStaleVersionAndDifferentApprovalRequest() {
        var stale = assertThrows(ReviewWorkflow.Rejected.class,
                () -> review.apply(ReviewWorkflow.Action.APPROVE, 1, reviewer, request, null, null));
        assertThat(stale.error()).isEqualTo(ReviewWorkflow.Error.STALE_REVISION);
        var wrongRequest = assertThrows(ReviewWorkflow.Rejected.class,
                () -> review.apply(ReviewWorkflow.Action.APPROVE, 2, reviewer, UUID.randomUUID(), null, null));
        assertThat(wrongRequest.error()).isEqualTo(ReviewWorkflow.Error.STATE_CONFLICT);
    }

    @Test
    void withdrawsWithReasonAndAllowsResubmissionAtNewVersion() {
        var result = review.apply(ReviewWorkflow.Action.WITHDRAW, 2, submitter, request, "Update scope", null);
        var draft = new ReviewWorkflow(result.state(), result.version(), author, null, null, 100);
        assertThat(draft.apply(ReviewWorkflow.Action.SUBMIT, 3, submitter, null, null, null).state())
                .isEqualTo(ReviewWorkflow.State.IN_REVIEW);
    }
}
