package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMediaPolicyUsesMIMESpecificLimits(t *testing.T) {
	ticketPolicy, err := PolicyForPurpose(MediaPurposeTicketAttachment)
	require.NoError(t, err)
	require.NoError(t, ticketPolicy.Validate("proof.pdf", "application/pdf", 10<<20))
	require.ErrorIs(t, ticketPolicy.Validate("proof.pdf", "application/pdf", (10<<20)+1), ErrMediaPolicy)
	require.NoError(t, ticketPolicy.Validate("proof.mp4", "video/mp4", 100<<20))

	reviewPolicy, err := PolicyForPurpose(MediaPurposeReviewMedia)
	require.NoError(t, err)
	require.True(t, errors.Is(reviewPolicy.Validate("proof.pdf", "application/pdf", 100), ErrMediaPolicy))
}

func TestValidateTicketInput(t *testing.T) {
	require.NoError(t, ValidateTicketInput("Need help", TicketCategoryOther, "Please help me"))
	require.ErrorIs(t, ValidateTicketInput("x", TicketCategoryOther, "message"), ErrInvalidArgument)
	require.ErrorIs(t, ValidateTicketInput("Valid subject", TicketCategory("UNKNOWN"), "message"), ErrInvalidArgument)
}
