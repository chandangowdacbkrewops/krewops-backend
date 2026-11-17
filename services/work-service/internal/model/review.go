package model

import (
	"strings"
	"time"
)

const (
	ReviewerRoleOwner  = "OWNER"
	ReviewerRoleWorker = "WORKER"

	ReviewCommentMaxLength = 500
)

// ValidRating reports whether rating is within the allowed 1-5 star range.
func ValidRating(rating int32) bool {
	return rating >= 1 && rating <= 5
}

// ValidReviewComment reports whether comment (if present) is within the
// allowed length, mirroring the payment_notes/description length checks
// used elsewhere in work-service.
func ValidReviewComment(comment *string) bool {
	if comment == nil {
		return true
	}
	return len(strings.TrimSpace(*comment)) <= ReviewCommentMaxLength
}

// WorkReview is a single rating + optional comment left by one party of a
// work assignment (the owner or the worker/contractor) about the other,
// once that assignment's work has been settled.
type WorkReview struct {
	ID           string `json:"id"`
	WorkID       string `json:"work_id"`
	AssignmentID string `json:"assignment_id"`

	ReviewerID   string `json:"reviewer_id"`
	ReviewerRole string `json:"reviewer_role"`
	ReviewerName string `json:"reviewer_name,omitempty"`

	RevieweeID   string `json:"reviewee_id"`
	RevieweeName string `json:"reviewee_name,omitempty"`

	Rating  int32   `json:"rating"`
	Comment *string `json:"comment,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// UserRatingSummary is the aggregate rating exposed for a user, computed
// on read from the work_reviews collection (no denormalized counter).
type UserRatingSummary struct {
	UserID        string  `json:"user_id"`
	AverageRating float64 `json:"average_rating"`
	ReviewCount   int32   `json:"review_count"`
}
