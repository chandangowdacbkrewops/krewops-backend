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
			WorkCategoryID:        req.WorkCategoryId,
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
			Attributes:            req.Attributes,
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

func (s *WorkServer) ListMyWorks(
	ctx context.Context,
	req *workv1.ListMyWorksRequest,
) (*workv1.ListMyWorksResponse, error) {
	requestID := incomingRequestID(ctx)
	log.Printf("list my works received: request_id=%s user_id=%s", requestID, req.UserId)

	result, err := s.workService.ListMyWorks(ctx, req.UserId, req.Status)
	if err != nil {
		log.Printf("list my works failed: request_id=%s user_id=%s error=%v", requestID, req.UserId, err)
		if isValidationError(err) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, "unable to list work postings")
	}

	works := make([]*workv1.WorkPosting, 0, len(result))
	for _, work := range result {
		works = append(works, toProtoWorkPosting(work))
	}

	return &workv1.ListMyWorksResponse{Works: works}, nil
}

func (s *WorkServer) ApplyToWork(
	ctx context.Context,
	req *workv1.ApplyToWorkRequest,
) (*workv1.ApplyToWorkResponse, error) {
	requestID := incomingRequestID(ctx)
	log.Printf("apply to work received: request_id=%s user_id=%s work_id=%s", requestID, req.UserId, req.WorkId)

	result, err := s.workService.ApplyToWork(
		ctx,
		req.UserId,
		req.WorkId,
		model.ApplyToWorkRequest{
			Quotation: quotationFromProto(req.Quotation),
		},
	)
	if err != nil {
		log.Printf("apply to work failed: request_id=%s user_id=%s work_id=%s error=%v", requestID, req.UserId, req.WorkId, err)
		return nil, mapApplicationError(err, "unable to apply to work")
	}

	return &workv1.ApplyToWorkResponse{
		Application: toProtoWorkApplication(result),
	}, nil
}

func (s *WorkServer) ListWorkApplications(
	ctx context.Context,
	req *workv1.ListWorkApplicationsRequest,
) (*workv1.ListWorkApplicationsResponse, error) {
	requestID := incomingRequestID(ctx)
	log.Printf("list work applications received: request_id=%s user_id=%s work_id=%s", requestID, req.UserId, req.WorkId)

	result, err := s.workService.ListWorkApplications(ctx, req.UserId, req.WorkId, req.Status)
	if err != nil {
		log.Printf("list work applications failed: request_id=%s user_id=%s work_id=%s error=%v", requestID, req.UserId, req.WorkId, err)
		return nil, mapApplicationError(err, "unable to list work applications")
	}

	applications := make([]*workv1.WorkApplication, 0, len(result))
	for i := range result {
		applications = append(applications, toProtoWorkApplication(&result[i]))
	}

	return &workv1.ListWorkApplicationsResponse{
		Applications: applications,
	}, nil
}

func (s *WorkServer) ListMyApplications(
	ctx context.Context,
	req *workv1.ListMyApplicationsRequest,
) (*workv1.ListMyApplicationsResponse, error) {
	requestID := incomingRequestID(ctx)
	log.Printf("list my applications received: request_id=%s user_id=%s", requestID, req.UserId)

	result, err := s.workService.ListMyApplications(ctx, req.UserId, req.Status)
	if err != nil {
		log.Printf("list my applications failed: request_id=%s user_id=%s error=%v", requestID, req.UserId, err)
		return nil, mapApplicationError(err, "unable to list applications")
	}

	items := make([]*workv1.MyWorkApplication, 0, len(result))
	for i := range result {
		items = append(items, &workv1.MyWorkApplication{
			Application: toProtoWorkApplication(&result[i].Application),
			Work:        toProtoWorkPosting(result[i].Work),
		})
	}

	return &workv1.ListMyApplicationsResponse{Applications: items}, nil
}

func (s *WorkServer) ShortlistWorkApplication(
	ctx context.Context,
	req *workv1.UpdateWorkApplicationRequest,
) (*workv1.UpdateWorkApplicationResponse, error) {
	return s.updateApplication(ctx, req, s.workService.ShortlistWorkApplication, "unable to shortlist application")
}

func (s *WorkServer) AcceptWorkApplication(
	ctx context.Context,
	req *workv1.UpdateWorkApplicationRequest,
) (*workv1.UpdateWorkApplicationResponse, error) {
	return s.updateApplication(ctx, req, s.workService.AcceptWorkApplication, "unable to accept application")
}

func (s *WorkServer) RejectWorkApplication(
	ctx context.Context,
	req *workv1.UpdateWorkApplicationRequest,
) (*workv1.UpdateWorkApplicationResponse, error) {
	return s.updateApplication(ctx, req, s.workService.RejectWorkApplication, "unable to reject application")
}

func (s *WorkServer) WithdrawWorkApplication(
	ctx context.Context,
	req *workv1.UpdateWorkApplicationRequest,
) (*workv1.UpdateWorkApplicationResponse, error) {
	return s.updateApplication(ctx, req, s.workService.WithdrawWorkApplication, "unable to withdraw application")
}

func (s *WorkServer) CancelWorkApplication(
	ctx context.Context,
	req *workv1.UpdateWorkApplicationRequest,
) (*workv1.UpdateWorkApplicationResponse, error) {
	return s.updateApplication(ctx, req, s.workService.CancelWorkApplication, "unable to cancel application")
}

func (s *WorkServer) updateApplication(
	ctx context.Context,
	req *workv1.UpdateWorkApplicationRequest,
	fn func(ctx context.Context, userID, workID, applicationID string) (*model.WorkApplication, error),
	fallback string,
) (*workv1.UpdateWorkApplicationResponse, error) {
	requestID := incomingRequestID(ctx)
	result, err := fn(ctx, req.UserId, req.WorkId, req.ApplicationId)
	if err != nil {
		log.Printf("update work application failed: request_id=%s user_id=%s work_id=%s application_id=%s error=%v", requestID, req.UserId, req.WorkId, req.ApplicationId, err)
		return nil, mapApplicationError(err, fallback)
	}

	return &workv1.UpdateWorkApplicationResponse{
		Application: toProtoWorkApplication(result),
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
		"work_category_id",
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
		"attributes",
		"quotation",
		"work_id",
		"application_id",
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
		WorkTypeName:          resp.WorkTypeName,
		WorkCategoryId:        resp.WorkCategoryID,
		WorkCategoryName:      resp.WorkCategoryName,
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
		Attributes:            resp.Attributes,
	}
}

func mapApplicationError(err error, fallback string) error {
	switch {
	case errors.Is(err, service.ErrWorkerProfileNotCompleted),
		errors.Is(err, service.ErrCannotApplyToOwnWork),
		errors.Is(err, service.ErrNotWorkOwner),
		errors.Is(err, service.ErrNotApplicationWorker):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, service.ErrWorkNotFound),
		errors.Is(err, service.ErrApplicationNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, service.ErrDuplicateApplication),
		errors.Is(err, service.ErrApplicationAlreadyAccepted):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, service.ErrWorkNotPublished),
		errors.Is(err, service.ErrInvalidApplicationStatus):
		return status.Error(codes.FailedPrecondition, err.Error())
	case isValidationError(err):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, fallback)
	}
}

func quotationFromProto(quotation *workv1.Quotation) model.Quotation {
	if quotation == nil {
		return model.Quotation{}
	}

	return model.Quotation{
		Amount:                 quotation.Amount,
		Currency:               quotation.Currency,
		PriceType:              quotation.PriceType,
		EstimatedDurationHours: quotation.EstimatedDurationHours,
		Message:                quotation.Message,
	}
}

func toProtoWorkApplication(application *model.WorkApplication) *workv1.WorkApplication {
	if application == nil {
		return nil
	}

	return &workv1.WorkApplication{
		Id:       application.ID,
		WorkId:   application.WorkID,
		WorkerId: application.WorkerID,
		Status:   application.Status,
		Quotation: &workv1.Quotation{
			Amount:                 application.Quotation.Amount,
			Currency:               application.Quotation.Currency,
			PriceType:              application.Quotation.PriceType,
			EstimatedDurationHours: application.Quotation.EstimatedDurationHours,
			Message:                application.Quotation.Message,
		},
		AppliedAt: application.AppliedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: application.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
