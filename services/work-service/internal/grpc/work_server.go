package grpcserver

import (
	"context"
	"errors"
	"log"

	workv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/work/v1"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type WorkServer struct {
	workv1.UnimplementedWorkServiceServer

	workService *service.WorkService
}

func NewWorkServer(
	workService *service.WorkService,
) *WorkServer {
	return &WorkServer{
		workService: workService,
	}
}

func (s *WorkServer) CreateWork(
	ctx context.Context,
	req *workv1.CreateWorkRequest,
) (*workv1.CreateWorkResponse, error) {
	requestID := incomingRequestID(ctx)
	log.Printf("create work received: request_id=%s user_id=%s status=%s work_type_id=%s", requestID, req.UserId, req.GetStatus(), req.GetWorkTypeId())

	result, err := s.workService.CreateWork(
		ctx,
		req.UserId,
		model.CreateWorkRequest{
			Status:                req.Status,
			Title:                 req.Title,
			WorkTypeID:            req.WorkTypeId,
			Description:           req.Description,
			Address:               req.Address,
			City:                  req.City,
			State:                 req.State,
			Latitude:              req.Latitude,
			Longitude:             req.Longitude,
			WorkersNeeded:         int32PtrToIntPtr(req.WorkersNeeded),
			ExperienceLevel:       req.ExperienceLevel,
			Skills:                req.Skills,
			ToolsProvided:         req.ToolsProvided,
			MaterialsProvided:     req.MaterialsProvided,
			StartDate:             req.StartDate,
			DurationValue:         int32PtrToIntPtr(req.DurationValue),
			DurationUnit:          req.DurationUnit,
			ShiftTiming:           req.ShiftTiming,
			PaymentType:           req.PaymentType,
			BudgetRate:            req.BudgetRate,
			PaymentNotes:          req.PaymentNotes,
			AccommodationProvided: req.AccommodationProvided,
			MealsProvided:         req.MealsProvided,
		},
	)

	if err != nil {
		log.Printf("create work failed: request_id=%s user_id=%s work_type_id=%s error=%v", requestID, req.UserId, req.GetWorkTypeId(), err)
		if errors.Is(err, service.ErrOnboardingNotCompleted) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		if isValidationError(err) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, "unable to create work posting")
	}
	log.Printf("create work succeeded: request_id=%s user_id=%s work_id=%s", requestID, req.UserId, result.ID)

	return &workv1.CreateWorkResponse{
		Work: toProtoWorkPosting(result),
	}, nil
}

func incomingRequestID(ctx context.Context) string {
	values := metadata.ValueFromIncomingContext(ctx, "x-request-id")
	if len(values) == 0 || values[0] == "" {
		return "unknown"
	}
	return values[0]
}

func isValidationError(err error) bool {
	msg := err.Error()
	validationPrefixes := []string{
		"status",
		"title",
		"description",
		"work_type_id",
		"address",
		"city",
		"state",
		"workers_needed",
		"tools_provided",
		"materials_provided",
		"start_date",
		"duration_value",
		"duration_unit",
		"payment_type",
		"budget_rate",
		"payment_notes",
	}
	for _, prefix := range validationPrefixes {
		if len(msg) >= len(prefix) && msg[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func int32PtrToIntPtr(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

func intPtrToInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}

func toProtoWorkPosting(resp *model.WorkResponse) *workv1.WorkPosting {
	if resp == nil {
		return nil
	}

	return &workv1.WorkPosting{
		Id:                    resp.ID,
		UserId:                resp.UserID,
		Status:                resp.Status,
		Title:                 resp.Title,
		WorkTypeId:            resp.WorkTypeID,
		Description:           resp.Description,
		Address:               resp.Address,
		City:                  resp.City,
		State:                 resp.State,
		Latitude:              resp.Latitude,
		Longitude:             resp.Longitude,
		WorkersNeeded:         intPtrToInt32Ptr(resp.WorkersNeeded),
		ExperienceLevel:       resp.ExperienceLevel,
		Skills:                resp.Skills,
		ToolsProvided:         resp.ToolsProvided,
		MaterialsProvided:     resp.MaterialsProvided,
		StartDate:             resp.StartDate,
		DurationValue:         intPtrToInt32Ptr(resp.DurationValue),
		DurationUnit:          resp.DurationUnit,
		ShiftTiming:           resp.ShiftTiming,
		PaymentType:           resp.PaymentType,
		BudgetRate:            resp.BudgetRate,
		PaymentNotes:          resp.PaymentNotes,
		AccommodationProvided: resp.AccommodationProvided,
		MealsProvided:         resp.MealsProvided,
		PublishedAt:           resp.PublishedAt,
		CreatedAt:             resp.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:             resp.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
