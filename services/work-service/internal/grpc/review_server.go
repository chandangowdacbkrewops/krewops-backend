package grpcserver

import (
	"context"

	workv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/work/v1"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
)

func (s *WorkServer) CreateReview(
	ctx context.Context,
	req *workv1.CreateReviewRequest,
) (*workv1.CreateReviewResponse, error) {
	result, err := s.workService.CreateReview(
		ctx,
		req.GetUserId(),
		req.GetAssignmentId(),
		req.GetRating(),
		req.Comment,
	)
	if err != nil {
		return nil, mapApplicationError(err, "unable to submit review")
	}
	return &workv1.CreateReviewResponse{Review: toProtoReview(result)}, nil
}

func (s *WorkServer) ListWorkReviews(
	ctx context.Context,
	req *workv1.ListWorkReviewsRequest,
) (*workv1.ListWorkReviewsResponse, error) {
	reviews, err := s.workService.ListWorkReviews(ctx, req.GetUserId(), req.GetWorkId())
	if err != nil {
		return nil, mapApplicationError(err, "unable to list reviews")
	}
	return &workv1.ListWorkReviewsResponse{Reviews: toProtoReviews(reviews)}, nil
}

func (s *WorkServer) ListUserReviews(
	ctx context.Context,
	req *workv1.ListUserReviewsRequest,
) (*workv1.ListUserReviewsResponse, error) {
	reviews, err := s.workService.ListUserReviews(ctx, req.GetTargetUserId())
	if err != nil {
		return nil, mapApplicationError(err, "unable to list reviews")
	}
	return &workv1.ListUserReviewsResponse{Reviews: toProtoReviews(reviews)}, nil
}

func (s *WorkServer) GetUserRatingSummary(
	ctx context.Context,
	req *workv1.GetUserRatingSummaryRequest,
) (*workv1.GetUserRatingSummaryResponse, error) {
	summary, err := s.workService.GetUserRatingSummary(ctx, req.GetTargetUserId())
	if err != nil {
		return nil, mapApplicationError(err, "unable to get rating summary")
	}
	return &workv1.GetUserRatingSummaryResponse{
		UserId:        summary.UserID,
		AverageRating: summary.AverageRating,
		ReviewCount:   summary.ReviewCount,
	}, nil
}

func toProtoReviews(reviews []model.WorkReview) []*workv1.WorkReview {
	mapped := make([]*workv1.WorkReview, 0, len(reviews))
	for i := range reviews {
		mapped = append(mapped, toProtoReview(&reviews[i]))
	}
	return mapped
}

func toProtoReview(review *model.WorkReview) *workv1.WorkReview {
	if review == nil {
		return nil
	}
	return &workv1.WorkReview{
		Id:           review.ID,
		WorkId:       review.WorkID,
		AssignmentId: review.AssignmentID,
		ReviewerId:   review.ReviewerID,
		ReviewerRole: review.ReviewerRole,
		RevieweeId:   review.RevieweeID,
		ReviewerName: stringPtr(review.ReviewerName),
		RevieweeName: stringPtr(review.RevieweeName),
		Rating:       review.Rating,
		Comment:      review.Comment,
		CreatedAt:    review.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
