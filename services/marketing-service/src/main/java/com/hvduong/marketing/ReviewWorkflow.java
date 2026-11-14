package com.hvduong.marketing;

import java.util.Objects;
import java.util.UUID;

record ReviewWorkflow(
        State state,
        long version,
        UUID author,
        UUID submittedBy,
        UUID openRequestId,
        long plannedBudgetVnd
) {
    enum State { DRAFT, IN_REVIEW, APPROVED, REJECTED, NEEDS_CHANGES }
    enum Action { SUBMIT, WITHDRAW, APPROVE, REJECT, REQUEST_CHANGES }
    enum Error { STALE_REVISION, STATE_CONFLICT, FORBIDDEN, VALIDATION_FAILED }

    static final class Rejected extends RuntimeException {
        private final Error error;

        Rejected(Error error) {
            super(error.name());
            this.error = error;
        }

        Error error() {
            return error;
        }
    }

    record Transition(State state, long version, Long approvedBudgetVnd) {}

    Transition apply(Action action, long expectedVersion, UUID actor, UUID requestId,
                     String reason, Long budgetCeilingVnd) {
        if (expectedVersion != version) {
            throw new Rejected(Error.STALE_REVISION);
        }
        if (action == Action.SUBMIT) {
            if (state != State.DRAFT) {
                throw new Rejected(Error.STATE_CONFLICT);
            }
            return new Transition(State.IN_REVIEW, version + 1, null);
        }
        if (state != State.IN_REVIEW || requestId == null || !requestId.equals(openRequestId)) {
            throw new Rejected(Error.STATE_CONFLICT);
        }
        if (action != Action.WITHDRAW && (actor.equals(author) || actor.equals(submittedBy))) {
            throw new Rejected(Error.FORBIDDEN);
        }
        if (action == Action.APPROVE) {
            if (budgetCeilingVnd != null && plannedBudgetVnd > budgetCeilingVnd) {
                throw new Rejected(Error.FORBIDDEN);
            }
            return new Transition(State.APPROVED, version + 1, plannedBudgetVnd);
        }
        if (Objects.isNull(reason) || reason.isBlank()) {
            throw new Rejected(Error.VALIDATION_FAILED);
        }
        State next = switch (action) {
            case WITHDRAW -> State.DRAFT;
            case REJECT -> State.REJECTED;
            case REQUEST_CHANGES -> State.NEEDS_CHANGES;
            default -> throw new IllegalStateException("Unexpected review action: " + action);
        };
        return new Transition(next, version + 1, null);
    }
}
