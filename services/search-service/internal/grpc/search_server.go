package grpcserver

import (
	"context"
	"time"

	searchv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/search/v1"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SearchServer struct {
	searchv1.UnimplementedSearchServiceServer

	searchService *service.SearchService
}

func NewSearchServer(searchService *service.SearchService) *SearchServer {
	return &SearchServer{searchService: searchService}
}

func (s *SearchServer) SearchWork(
	ctx context.Context,
	req *searchv1.SearchWorkRequest,
) (*searchv1.SearchWorkResponse, error) {
	result, err := s.searchService.SearchWork(ctx, model.WorkSearchFilters{
		Keyword:         req.Keyword,
		WorkTypeID:      req.WorkTypeId,
		City:            req.City,
		State:           req.State,
		ExperienceLevel: req.ExperienceLevel,
		PaymentType:     req.PaymentType,
		MinBudgetRate:   req.MinBudgetRate,
		MaxBudgetRate:   req.MaxBudgetRate,
		Page:            int(req.GetPage()),
		PageSize:        int(req.GetPageSize()),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to search work postings")
	}

	resp := &searchv1.SearchWorkResponse{
		Results:  make([]*searchv1.WorkSummary, 0, len(result.Results)),
		Total:    int32(result.Total),
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
	}
	for _, item := range result.Results {
		resp.Results = append(resp.Results, toWorkSummary(item))
	}

	return resp, nil
}

func (s *SearchServer) SearchWorkers(
	ctx context.Context,
	req *searchv1.SearchWorkersRequest,
) (*searchv1.SearchWorkersResponse, error) {
	result, err := s.searchService.SearchWorkers(ctx, model.WorkerSearchFilters{
		Keyword:            req.Keyword,
		WorkTypeID:         req.WorkTypeId,
		City:               req.City,
		State:              req.State,
		WorkerType:         req.WorkerType,
		AvailabilityStatus: req.AvailabilityStatus,
		MinExperienceYears: req.MinExperienceYears,
		Page:               int(req.GetPage()),
		PageSize:           int(req.GetPageSize()),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to search workers")
	}

	resp := &searchv1.SearchWorkersResponse{
		Results:  make([]*searchv1.WorkerSummary, 0, len(result.Results)),
		Total:    int32(result.Total),
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
	}
	for _, item := range result.Results {
		resp.Results = append(resp.Results, toWorkerSummary(item))
	}

	return resp, nil
}

func toWorkSummary(item model.WorkSearchResult) *searchv1.WorkSummary {
	summary := &searchv1.WorkSummary{
		Id:              item.ID,
		UserId:          item.UserID,
		Status:          item.Status,
		Title:           item.Title,
		WorkTypeId:      item.WorkTypeID,
		WorkTypeName:    item.WorkTypeName,
		Description:     item.Description,
		Address:         item.Address,
		City:            item.City,
		State:           item.State,
		Latitude:        item.Latitude,
		Longitude:       item.Longitude,
		WorkersNeeded:   intPtrToInt32Ptr(item.WorkersNeeded),
		ExperienceLevel: item.ExperienceLevel,
		Skills:          item.Skills,
		DurationValue:   intPtrToInt32Ptr(item.DurationValue),
		DurationUnit:    item.DurationUnit,
		ShiftTiming:     item.ShiftTiming,
		PaymentType:     item.PaymentType,
		BudgetRate:      item.BudgetRate,
		CreatedAt:       item.CreatedAt.Format(time.RFC3339),
	}

	if item.StartDate != nil {
		formatted := item.StartDate.Format("2006-01-02")
		summary.StartDate = &formatted
	}
	if item.PublishedAt != nil {
		formatted := item.PublishedAt.Format(time.RFC3339)
		summary.PublishedAt = &formatted
	}
	if summary.Skills == nil {
		summary.Skills = []string{}
	}

	return summary
}

func toWorkerSummary(item model.WorkerSearchResult) *searchv1.WorkerSummary {
	summary := &searchv1.WorkerSummary{
		Id:                 item.ID,
		UserId:             item.UserID,
		FirstName:          item.FirstName,
		LastName:           item.LastName,
		City:               item.City,
		State:              item.State,
		WorkerType:         item.WorkerType,
		CrewName:           item.CrewName,
		CrewSize:           int32(item.CrewSize),
		ExperienceYears:    item.ExperienceYears,
		ExpectedRate:       item.ExpectedRate,
		RateType:           item.RateType,
		AvailabilityStatus: item.AvailabilityStatus,
		VerificationStatus: item.VerificationStatus,
		Bio:                item.Bio,
		Skills:             make([]*searchv1.WorkerSkillSummary, 0, len(item.Skills)),
	}

	for _, skill := range item.Skills {
		summary.Skills = append(summary.Skills, &searchv1.WorkerSkillSummary{
			WorkTypeId:      skill.WorkTypeID,
			WorkTypeName:    skill.WorkTypeName,
			ExperienceYears: skill.ExperienceYears,
		})
	}

	return summary
}

func intPtrToInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}
