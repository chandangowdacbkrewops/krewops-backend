package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/repository"
)

var (
	ErrReviewWorkNotSettled    = errors.New("reviews are only allowed once the work has been paid out or closed")
	ErrDuplicateReview         = repository.ErrDuplicateReview
	ErrNotAuthorizedToReview   = errors.New("only the work owner or the assigned worker can leave this review")
	ErrInvalidReviewAssignment = errors.New("assignment cannot be reviewed from its current status")
)

// CreateReview records a 1-5 rating (+ optional comment) from one party of
// an assignment (owner or worker/contractor) about the other. A review is
// only allowed once the work posting has been settled (payment_pending,
// completed, or closed) and is immutable once created - at most one review
// per (assignment, reviewer).
func (s *WorkService) CreateReview(
	ctx context.Context,
	userID string,
	assignmentID string,
	rating int32,
	comment *string,
) (*model.WorkReview, error) {
	log.Printf("create review started: user_id=%s assignment_id=%s", userID, assignmentID)

	if !model.ValidRating(rating) {
		return nil, fmt.Errorf("rating must be between 1 and 5")
	}
	if !model.ValidReviewComment(comment) {
		return nil, fmt.Errorf("comment must be %d characters or fewer", model.ReviewCommentMaxLength)
	}

	assignment, work, err := s.requireAssignmentOwnerOrWorker(ctx, userID, assignmentID)
	if err != nil {
		return nil, err
	}

	if !model.IsActiveAssignment(assignment.Status) {
		return nil, ErrInvalidReviewAssignment
	}

	switch model.NormalizeWorkStatus(work.Status) {
	case model.StatusPaymentPending, model.StatusCompleted, model.StatusClosed:
		// settled, reviews allowed
	default:
		return nil, ErrReviewWorkNotSettled
	}

	var reviewerRole, revieweeID string
	if work.UserID == userID {
		reviewerRole = model.ReviewerRoleOwner
		revieweeID = assignment.PayeeID()
	} else if assignment.WorkerID == userID || assignment.PayeeID() == userID {
		reviewerRole = model.ReviewerRoleWorker
		revieweeID = work.UserID
	} else {
		return nil, ErrNotAuthorizedToReview
	}

	comment = trimmedOrNilReview(comment)

	review := &model.WorkReview{
		WorkID:       work.ID,
		AssignmentID: assignment.ID,
		ReviewerID:   userID,
		ReviewerRole: reviewerRole,
		RevieweeID:   revieweeID,
		Rating:       rating,
		Comment:      comment,
	}

	saved, err := s.reviewRepository.Insert(ctx, review)
	if err != nil {
		log.Printf("create review failed: user_id=%s assignment_id=%s error=%v", userID, assignmentID, err)
		return nil, err
	}
	log.Printf("create review succeeded: user_id=%s assignment_id=%s review_id=%s", userID, assignmentID, saved.ID)

	return s.decorateReview(ctx, saved), nil
}

// ListWorkReviews returns all reviews left on a work posting, visible to
// the work owner or any party with an assignment on that work.
func (s *WorkService) ListWorkReviews(
	ctx context.Context,
	userID string,
	workID string,
) ([]model.WorkReview, error) {
	if _, _, err := s.requireWorkViewer(ctx, userID, workID); err != nil {
		return nil, err
	}

	reviews, err := s.reviewRepository.ListByWorkID(ctx, workID)
	if err != nil {
		return nil, err
	}
	return s.decorateReviews(ctx, reviews), nil
}

// ListUserReviews returns the reviews a user has received across all of
// their work. This is public (any authenticated user may view it) since it
// is meant to be shown on a profile.
func (s *WorkService) ListUserReviews(
	ctx context.Context,
	targetUserID string,
) ([]model.WorkReview, error) {
	if strings.TrimSpace(targetUserID) == "" {
		return nil, fmt.Errorf("target_user_id is required")
	}

	reviews, err := s.reviewRepository.ListByRevieweeID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	return s.decorateReviews(ctx, reviews), nil
}

// GetUserRatingSummary returns the average rating and review count for a
// user, computed on read. Public (any authenticated user may view it).
func (s *WorkService) GetUserRatingSummary(
	ctx context.Context,
	targetUserID string,
) (*model.UserRatingSummary, error) {
	if strings.TrimSpace(targetUserID) == "" {
		return nil, fmt.Errorf("target_user_id is required")
	}

	return s.reviewRepository.AggregateRatingForUser(ctx, targetUserID)
}

func (s *WorkService) decorateReview(ctx context.Context, review *model.WorkReview) *model.WorkReview {
	reviews := s.decorateReviews(ctx, []model.WorkReview{*review})
	if len(reviews) == 0 {
		return review
	}
	return &reviews[0]
}

func (s *WorkService) decorateReviews(ctx context.Context, reviews []model.WorkReview) []model.WorkReview {
	if len(reviews) == 0 {
		return reviews
	}

	userIDs := make([]string, 0, len(reviews)*2)
	seen := map[string]struct{}{}
	for _, review := range reviews {
		for _, id := range []string{review.ReviewerID, review.RevieweeID} {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				userIDs = append(userIDs, id)
			}
		}
	}

	names, err := s.workOwnerRepository.FindDisplayNames(ctx, userIDs)
	if err != nil {
		log.Printf("review display names failed: error=%v", err)
		names = map[string]string{}
	}

	for i := range reviews {
		reviews[i].ReviewerName = names[reviews[i].ReviewerID]
		reviews[i].RevieweeName = names[reviews[i].RevieweeID]
	}
	return reviews
}

func trimmedOrNilReview(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
