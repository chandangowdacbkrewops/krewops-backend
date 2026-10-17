package grpcserver

import (
	"context"
	"errors"
	"log"
	"time"

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
			WorkSegmentID:         req.WorkSegmentId,
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
			RateBasis:             req.RateBasis,
			RateUnit:              req.RateUnit,
			BudgetRate:            req.BudgetRate,
			PaymentNotes:          req.PaymentNotes,
			AccommodationProvided: req.AccommodationProvided,
			MealsProvided:         req.MealsProvided,
			Attributes:            req.Attributes,
			HiringMode:            req.HiringMode,
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
			Assignment:  toProtoAssignment(result[i].Assignment),
			Payments:    toProtoPayments(result[i].Payments),
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

func (s *WorkServer) StartWork(
	ctx context.Context,
	req *workv1.WorkLifecycleRequest,
) (*workv1.WorkLifecycleResponse, error) {
	return s.updateWorkStatus(ctx, req, s.workService.StartWork, "unable to start work")
}

func (s *WorkServer) CompleteWork(
	ctx context.Context,
	req *workv1.WorkLifecycleRequest,
) (*workv1.WorkLifecycleResponse, error) {
	return s.updateWorkStatus(ctx, req, s.workService.CompleteWork, "unable to complete work")
}

func (s *WorkServer) ConfirmWorkCompletion(
	ctx context.Context,
	req *workv1.WorkLifecycleRequest,
) (*workv1.WorkLifecycleResponse, error) {
	return s.updateWorkStatus(ctx, req, s.workService.ConfirmWorkCompletion, "unable to confirm work completion")
}

func (s *WorkServer) CancelWork(
	ctx context.Context,
	req *workv1.WorkLifecycleRequest,
) (*workv1.WorkLifecycleResponse, error) {
	return s.updateWorkStatus(ctx, req, s.workService.CancelWork, "unable to cancel work")
}

func (s *WorkServer) CloseApplications(
	ctx context.Context,
	req *workv1.WorkLifecycleRequest,
) (*workv1.WorkLifecycleResponse, error) {
	return s.updateWorkStatus(ctx, req, s.workService.CloseApplications, "unable to close applications")
}

func (s *WorkServer) CompleteJob(
	ctx context.Context,
	req *workv1.WorkLifecycleRequest,
) (*workv1.WorkLifecycleResponse, error) {
	return s.updateWorkStatus(ctx, req, s.workService.CompleteJob, "unable to complete job")
}

func (s *WorkServer) CloseWork(
	ctx context.Context,
	req *workv1.WorkLifecycleRequest,
) (*workv1.WorkLifecycleResponse, error) {
	return s.updateWorkStatus(ctx, req, s.workService.CloseWork, "unable to close work")
}

func (s *WorkServer) StartAssignment(
	ctx context.Context,
	req *workv1.AssignmentLifecycleRequest,
) (*workv1.AssignmentLifecycleResponse, error) {
	result, err := s.workService.StartAssignment(ctx, req.UserId, req.AssignmentId)
	if err != nil {
		return nil, mapApplicationError(err, "unable to start assignment")
	}
	return &workv1.AssignmentLifecycleResponse{Assignment: toProtoAssignment(result)}, nil
}

func (s *WorkServer) CompleteAssignment(
	ctx context.Context,
	req *workv1.AssignmentLifecycleRequest,
) (*workv1.AssignmentLifecycleResponse, error) {
	result, err := s.workService.CompleteAssignment(ctx, req.UserId, req.AssignmentId)
	if err != nil {
		return nil, mapApplicationError(err, "unable to complete assignment")
	}
	return &workv1.AssignmentLifecycleResponse{Assignment: toProtoAssignment(result)}, nil
}

func (s *WorkServer) UpsertDailyLog(
	ctx context.Context,
	req *workv1.UpsertDailyLogRequest,
) (*workv1.UpsertDailyLogResponse, error) {
	result, err := s.workService.UpsertDailyLog(ctx, req.UserId, req.AssignmentId, req.WorkDate, model.UpsertDailyLogRequest{
		DayPortion: req.DayPortion,
		Note:       req.Note,
	})
	if err != nil {
		return nil, mapApplicationError(err, "unable to save daily log")
	}
	return &workv1.UpsertDailyLogResponse{DailyLog: toProtoDailyLog(result)}, nil
}

func (s *WorkServer) ListAssignmentDailyLogs(
	ctx context.Context,
	req *workv1.ListAssignmentDailyLogsRequest,
) (*workv1.ListDailyLogsResponse, error) {
	result, err := s.workService.ListAssignmentDailyLogs(ctx, req.UserId, req.AssignmentId)
	if err != nil {
		return nil, mapApplicationError(err, "unable to list daily logs")
	}
	return &workv1.ListDailyLogsResponse{DailyLogs: toProtoDailyLogs(result)}, nil
}

func (s *WorkServer) ListWorkDailyLogs(
	ctx context.Context,
	req *workv1.ListWorkDailyLogsRequest,
) (*workv1.ListDailyLogsResponse, error) {
	result, err := s.workService.ListWorkDailyLogs(ctx, req.UserId, req.WorkId)
	if err != nil {
		return nil, mapApplicationError(err, "unable to list daily logs")
	}
	return &workv1.ListDailyLogsResponse{DailyLogs: toProtoDailyLogs(result)}, nil
}

func (s *WorkServer) GetWorkSettlement(
	ctx context.Context,
	req *workv1.GetWorkSettlementRequest,
) (*workv1.GetWorkSettlementResponse, error) {
	result, err := s.workService.GetWorkSettlement(ctx, req.UserId, req.WorkId)
	if err != nil {
		return nil, mapApplicationError(err, "unable to get settlement")
	}
	workers := make([]*workv1.WorkerSettlement, 0, len(result.Workers))
	for i := range result.Workers {
		worker := result.Workers[i]
		workers = append(workers, &workv1.WorkerSettlement{
			Assignment:    toProtoAssignment(&worker.Assignment),
			Logs:          toProtoDailyLogs(worker.Logs),
			Payments:      toProtoPayments(worker.Payments),
			FullDays:      int32(worker.FullDays),
			HalfDays:      int32(worker.HalfDays),
			DaysWorked:    worker.DaysWorked,
			PayableAmount: worker.PayableAmount,
			Currency:      worker.Currency,
			PaymentStatus: worker.PaymentStatus,
		})
	}
	return &workv1.GetWorkSettlementResponse{
		Work:    toProtoWorkPosting(result.Work),
		Workers: workers,
	}, nil
}

func (s *WorkServer) RecordAssignmentPayment(
	ctx context.Context,
	req *workv1.RecordAssignmentPaymentRequest,
) (*workv1.RecordAssignmentPaymentResponse, error) {
	result, err := s.workService.RecordAssignmentPayment(ctx, req.UserId, req.WorkId, req.AssignmentId, model.RecordPaymentRequest{
		Method:    req.Method,
		Amount:    req.GetAmount(),
		Currency:  req.GetCurrency(),
		Reference: req.Reference,
	})
	if err != nil {
		return nil, mapApplicationError(err, "unable to record payment")
	}
	return &workv1.RecordAssignmentPaymentResponse{Payment: toProtoPayment(result)}, nil
}

func (s *WorkServer) ListAssignmentPayments(
	ctx context.Context,
	req *workv1.ListAssignmentPaymentsRequest,
) (*workv1.ListAssignmentPaymentsResponse, error) {
	result, err := s.workService.ListAssignmentPayments(ctx, req.UserId, req.AssignmentId)
	if err != nil {
		return nil, mapApplicationError(err, "unable to list payments")
	}
	return &workv1.ListAssignmentPaymentsResponse{Payments: toProtoPayments(result)}, nil
}

func (s *WorkServer) ConfirmAssignmentPayment(
	ctx context.Context,
	req *workv1.UpdateAssignmentPaymentRequest,
) (*workv1.RecordAssignmentPaymentResponse, error) {
	result, err := s.workService.ConfirmAssignmentPayment(ctx, req.UserId, req.AssignmentId, req.PaymentId)
	if err != nil {
		return nil, mapApplicationError(err, "unable to confirm payment")
	}
	return &workv1.RecordAssignmentPaymentResponse{Payment: toProtoPayment(result)}, nil
}

func (s *WorkServer) DisputeAssignmentPayment(
	ctx context.Context,
	req *workv1.DisputeAssignmentPaymentRequest,
) (*workv1.RecordAssignmentPaymentResponse, error) {
	result, err := s.workService.DisputeAssignmentPayment(ctx, req.UserId, req.AssignmentId, req.PaymentId, req.Message)
	if err != nil {
		return nil, mapApplicationError(err, "unable to dispute payment")
	}
	return &workv1.RecordAssignmentPaymentResponse{Payment: toProtoPayment(result)}, nil
}

func (s *WorkServer) GetWorkDayRecords(
	ctx context.Context,
	req *workv1.GetWorkDayRecordsRequest,
) (*workv1.GetWorkDayRecordsResponse, error) {
	result, err := s.workService.GetWorkDayRecords(ctx, req.UserId, req.WorkId, req.WorkDate)
	if err != nil {
		return nil, mapApplicationError(err, "unable to get work records")
	}
	return toProtoWorkDayRecords(result), nil
}

func (s *WorkServer) SaveWorkDayRecords(
	ctx context.Context,
	req *workv1.SaveWorkDayRecordsRequest,
) (*workv1.GetWorkDayRecordsResponse, error) {
	inputs := make([]model.SaveWorkDayRecordInput, 0, len(req.Records))
	for _, record := range req.Records {
		inputs = append(inputs, model.SaveWorkDayRecordInput{
			AssignmentID: record.AssignmentId,
			DayType:      record.DayType,
		})
	}
	result, err := s.workService.SaveWorkDayRecords(ctx, req.UserId, req.WorkId, model.SaveWorkDayRecordsRequest{
		WorkDate: req.WorkDate,
		Records:  inputs,
	})
	if err != nil {
		return nil, mapApplicationError(err, "unable to save work records")
	}
	return toProtoWorkDayRecords(result), nil
}

func (s *WorkServer) GetWorkerWorkSummary(
	ctx context.Context,
	req *workv1.GetWorkerWorkSummaryRequest,
) (*workv1.GetWorkerWorkSummaryResponse, error) {
	result, err := s.workService.GetWorkerWorkSummary(ctx, req.UserId, req.WorkId)
	if err != nil {
		return nil, mapApplicationError(err, "unable to get worker work summary")
	}
	workers := make([]*workv1.WorkerWorkSummary, 0, len(result.Workers))
	for _, worker := range result.Workers {
		workers = append(workers, &workv1.WorkerWorkSummary{
			AssignmentId:          worker.AssignmentID,
			WorkerProfileId:       worker.WorkerProfileID,
			WorkerName:            worker.WorkerName,
			FullDays:              int32(worker.FullDays),
			HalfDays:              int32(worker.HalfDays),
			NotWorkedDays:         int32(worker.NotWorkedDays),
			UnrecordedDays:        int32(worker.UnrecordedDays),
			PayableDays:           worker.PayableDays,
			AgreedRateMinor:       worker.AgreedRateMinor,
			CalculatedAmountMinor: worker.CalculatedAmountMinor,
			Currency:              worker.Currency,
		})
	}
	return &workv1.GetWorkerWorkSummaryResponse{WorkId: result.WorkID, Workers: workers}, nil
}

func (s *WorkServer) GetContractorWorkRecords(
	ctx context.Context,
	req *workv1.GetContractorWorkRecordsRequest,
) (*workv1.GetContractorWorkRecordsResponse, error) {
	result, err := s.workService.GetContractorWorkRecords(ctx, req.UserId, req.WorkId, req.GetWorkDate())
	if err != nil {
		return nil, mapApplicationError(err, "unable to get contractor work records")
	}
	return toProtoContractorWorkRecords(result), nil
}

func (s *WorkServer) SaveContractorWorkRecord(
	ctx context.Context,
	req *workv1.SaveContractorWorkRecordRequest,
) (*workv1.GetContractorWorkRecordsResponse, error) {
	result, err := s.workService.SaveContractorWorkRecord(ctx, req.UserId, req.WorkId, model.SaveContractorWorkRecordRequest{
		WorkDate:            req.WorkDate,
		DeployedWorkerCount: int(req.DeployedWorkerCount),
	})
	if err != nil {
		return nil, mapApplicationError(err, "unable to save contractor work record")
	}
	return toProtoContractorWorkRecords(result), nil
}

func (s *WorkServer) GetContractorWorkSummary(
	ctx context.Context,
	req *workv1.GetContractorWorkSummaryRequest,
) (*workv1.GetContractorWorkSummaryResponse, error) {
	result, err := s.workService.GetContractorWorkSummary(ctx, req.UserId, req.WorkId)
	if err != nil {
		return nil, mapApplicationError(err, "unable to get contractor work summary")
	}
	return &workv1.GetContractorWorkSummaryResponse{
		WorkId:                  result.WorkID,
		AssignmentId:            result.AssignmentID,
		ContractorProfileId:     result.ContractorProfileID,
		ContractorName:          result.ContractorName,
		CommittedWorkerCount:    int32(result.CommittedWorkerCount),
		TotalRecordedDays:       int32(result.TotalRecordedDays),
		TotalDeployedWorkerDays: int32(result.TotalDeployedWorkerDays),
		AverageDeployedWorkers:  result.AverageDeployedWorkers,
		TotalShortfall:          int32(result.TotalShortfall),
		RateType:                result.RateType,
		RateMinor:               result.RateMinor,
		CompletedQuantity:       result.CompletedQuantity,
		EstimatedAmountMinor:    result.EstimatedAmountMinor,
		Currency:                result.Currency,
	}, nil
}

func (s *WorkServer) GetWorkPayments(
	ctx context.Context,
	req *workv1.GetWorkPaymentsRequest,
) (*workv1.GetWorkPaymentsResponse, error) {
	result, err := s.workService.GetWorkPayments(ctx, req.UserId, req.WorkId)
	if err != nil {
		return nil, mapApplicationError(err, "unable to get work payments")
	}
	payments := make([]*workv1.WorkPaymentItem, 0, len(result.Payments))
	for _, item := range result.Payments {
		payments = append(payments, toProtoWorkPaymentItem(item))
	}
	return &workv1.GetWorkPaymentsResponse{
		WorkId:           result.WorkID,
		HiringMode:       result.HiringMode,
		OutstandingMinor: result.OutstandingMinor,
		Payments:         payments,
	}, nil
}

func (s *WorkServer) RecordPayment(
	ctx context.Context,
	req *workv1.RecordPaymentRequest,
) (*workv1.RecordPaymentResponse, error) {
	result, err := s.workService.RecordObligationPayment(ctx, req.UserId, req.PaymentId, model.RecordObligationPaymentRequest{
		AmountMinor:   req.AmountMinor,
		PaymentMethod: req.PaymentMethod,
		Reference:     req.Reference,
		PaidAt:        req.PaidAt,
	})
	if err != nil {
		return nil, mapApplicationError(err, "unable to record payment")
	}
	return &workv1.RecordPaymentResponse{Transaction: toProtoPaymentTransaction(result)}, nil
}

func toProtoWorkDayRecords(result *model.WorkDayRecordsResponse) *workv1.GetWorkDayRecordsResponse {
	if result == nil {
		return &workv1.GetWorkDayRecordsResponse{}
	}
	workers := make([]*workv1.WorkerDayRecordItem, 0, len(result.Workers))
	for _, worker := range result.Workers {
		workers = append(workers, &workv1.WorkerDayRecordItem{
			AssignmentId:         worker.AssignmentID,
			WorkerProfileId:      worker.WorkerProfileID,
			WorkerName:           worker.WorkerName,
			DayType:              worker.DayType,
			PayableUnits:         worker.PayableUnits,
			AgreedRateMinor:      worker.AgreedRateMinor,
			EstimatedAmountMinor: worker.EstimatedAmountMinor,
		})
	}
	return &workv1.GetWorkDayRecordsResponse{
		WorkId:   result.WorkID,
		WorkDate: result.WorkDate,
		Workers:  workers,
	}
}

func toProtoContractorWorkRecords(result *model.ContractorWorkRecordsResponse) *workv1.GetContractorWorkRecordsResponse {
	if result == nil {
		return &workv1.GetContractorWorkRecordsResponse{}
	}
	records := make([]*workv1.ContractorWorkRecordItem, 0, len(result.Records))
	for _, record := range result.Records {
		records = append(records, &workv1.ContractorWorkRecordItem{
			WorkDate:            record.WorkDate,
			DeployedWorkerCount: int32(record.DeployedWorkerCount),
			Shortfall:           int32(record.Shortfall),
		})
	}
	return &workv1.GetContractorWorkRecordsResponse{
		WorkId: result.WorkID,
		Contractor: &workv1.ContractorInfo{
			AssignmentId:         result.Contractor.AssignmentID,
			ContractorProfileId:  result.Contractor.ContractorProfileID,
			ContractorName:       result.Contractor.ContractorName,
			CommittedWorkerCount: int32(result.Contractor.CommittedWorkerCount),
		},
		Records: records,
	}
}

func toProtoWorkPaymentItem(item model.WorkPaymentItem) *workv1.WorkPaymentItem {
	transactions := make([]*workv1.PaymentTransaction, 0, len(item.Transactions))
	for i := range item.Transactions {
		transactions = append(transactions, toProtoPaymentTransaction(&item.Transactions[i]))
	}
	return &workv1.WorkPaymentItem{
		Obligation: &workv1.PaymentObligation{
			Id:                 item.Obligation.ID,
			WorkId:             item.Obligation.WorkID,
			AssignmentId:       item.Obligation.AssignmentID,
			PayeeType:          item.Obligation.PayeeType,
			PayeeId:            item.Obligation.PayeeID,
			PayeeName:          item.Obligation.PayeeName,
			Currency:           item.Obligation.Currency,
			PayableAmountMinor: item.Obligation.PayableAmountMinor,
			PaidAmountMinor:    item.Obligation.PaidAmountMinor,
			Status:             item.Obligation.Status,
			CreatedAt:          item.Obligation.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:          item.Obligation.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		AssignmentId:         item.AssignmentID,
		PayeeType:            item.PayeeType,
		PayeeId:              item.PayeeID,
		PayeeName:            item.PayeeName,
		RateType:             item.RateType,
		PayableAmountMinor:   item.PayableAmountMinor,
		PaidAmountMinor:      item.PaidAmountMinor,
		RemainingAmountMinor: item.RemainingAmountMinor,
		Status:               item.Status,
		Transactions:         transactions,
	}
}

func toProtoPaymentTransaction(txn *model.PaymentTransaction) *workv1.PaymentTransaction {
	if txn == nil {
		return nil
	}
	return &workv1.PaymentTransaction{
		Id:                  txn.ID,
		PaymentObligationId: txn.PaymentObligationID,
		WorkId:              txn.WorkID,
		AssignmentId:        txn.AssignmentID,
		AmountMinor:         txn.AmountMinor,
		PaymentMethod:       txn.PaymentMethod,
		Reference:           txn.Reference,
		RecordedBy:          txn.RecordedBy,
		PaidAt:              txn.PaidAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt:           txn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (s *WorkServer) updateWorkStatus(
	ctx context.Context,
	req *workv1.WorkLifecycleRequest,
	fn func(ctx context.Context, userID, workID string) (*model.WorkResponse, error),
	fallback string,
) (*workv1.WorkLifecycleResponse, error) {
	requestID := incomingRequestID(ctx)
	result, err := fn(ctx, req.UserId, req.WorkId)
	if err != nil {
		log.Printf("update work status failed: request_id=%s user_id=%s work_id=%s error=%v", requestID, req.UserId, req.WorkId, err)
		return nil, mapApplicationError(err, fallback)
	}

	return &workv1.WorkLifecycleResponse{
		Work: toProtoWorkPosting(result),
	}, nil
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
		"work_segment_id",
		"address",
		"city",
		"state",
		"workers_needed",
		"tools_provided",
		"materials_provided",
		"start_date",
		"duration_value",
		"duration_unit",
		"rate_basis",
		"rate_unit",
		"budget_rate",
		"payment_notes",
		"attributes",
		"quotation",
		"work_id",
		"application_id",
		"day_portion",
		"day_type",
		"work_date",
		"amount",
		"amount_minor",
		"assignment_id",
		"deployed_worker_count",
		"paid_at",
		"hiring_mode",
		"payment_id",
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
		WorkSegmentId:         resp.WorkSegmentID,
		WorkSegmentName:       resp.WorkSegmentName,
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
		RateBasis:             resp.RateBasis,
		RateUnit:              resp.RateUnit,
		BudgetRate:            resp.BudgetRate,
		PaymentNotes:          resp.PaymentNotes,
		AccommodationProvided: resp.AccommodationProvided,
		MealsProvided:         resp.MealsProvided,
		PublishedAt:           resp.PublishedAt,
		CreatedAt:             resp.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:             resp.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Attributes:            resp.Attributes,
		WorkersSelected:       int32Ptr(int32(resp.WorkersSelected)),
		ApplicationsOpen:      boolPtr(resp.ApplicationsOpen),
		HiringMode:            stringPtr(resp.HiringMode),
		StartedAt:             resp.StartedAt,
	}
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func int32Ptr(value int32) *int32 { return &value }

func boolPtr(value bool) *bool { return &value }

func timePtrString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("2006-01-02T15:04:05Z07:00")
	return &formatted
}

func mapApplicationError(err error, fallback string) error {
	switch {
	case errors.Is(err, service.ErrWorkerProfileNotCompleted),
		errors.Is(err, service.ErrCannotApplyToOwnWork),
		errors.Is(err, service.ErrNotWorkOwner),
		errors.Is(err, service.ErrNotApplicationWorker),
		errors.Is(err, service.ErrNotAssignedWorker),
		errors.Is(err, service.ErrContractorRequired):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, service.ErrWorkNotFound),
		errors.Is(err, service.ErrApplicationNotFound),
		errors.Is(err, service.ErrAssignmentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, service.ErrDuplicateApplication),
		errors.Is(err, service.ErrApplicationAlreadyAccepted),
		errors.Is(err, service.ErrNoWorkerSeats):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, service.ErrWorkNotPublished),
		errors.Is(err, service.ErrInvalidApplicationStatus),
		errors.Is(err, service.ErrInvalidWorkStatus),
		errors.Is(err, service.ErrWorkNotReadyToStart),
		errors.Is(err, service.ErrApplicationsClosed),
		errors.Is(err, service.ErrInvalidAssignmentStatus),
		errors.Is(err, service.ErrLogsFrozen),
		errors.Is(err, service.ErrAssignmentsNotCompleted),
		errors.Is(err, service.ErrPaymentsIncomplete),
		errors.Is(err, service.ErrInvalidPaymentMethod),
		errors.Is(err, service.ErrNoActiveAssignment),
		errors.Is(err, service.ErrContractorAssignmentRequired),
		errors.Is(err, service.ErrContractorAlreadyAssigned),
		errors.Is(err, service.ErrHiringModeMismatch),
		errors.Is(err, service.ErrPaymentAdjustmentRequired),
		errors.Is(err, service.ErrOverpayment),
		errors.Is(err, service.ErrIndividualWorkersNotAllowed):
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
		RateBasis:              quotation.RateBasis,
		RateUnit:               quotation.RateUnit,
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
			RateBasis:              application.Quotation.RateBasis,
			RateUnit:               application.Quotation.RateUnit,
			EstimatedDurationHours: application.Quotation.EstimatedDurationHours,
			Message:                application.Quotation.Message,
		},
		AppliedAt: application.AppliedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: application.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toProtoAssignment(assignment *model.WorkAssignment) *workv1.WorkAssignment {
	if assignment == nil {
		return nil
	}
	return &workv1.WorkAssignment{
		Id:            assignment.ID,
		WorkId:        assignment.WorkID,
		ApplicationId: assignment.ApplicationID,
		WorkerId:      assignment.WorkerID,
		Status:        assignment.Status,
		AgreedQuotation: &workv1.Quotation{
			Amount:                 assignment.Quotation.Amount,
			Currency:               assignment.Quotation.Currency,
			RateBasis:              assignment.Quotation.RateBasis,
			RateUnit:               assignment.Quotation.RateUnit,
			EstimatedDurationHours: assignment.Quotation.EstimatedDurationHours,
			Message:                assignment.Quotation.Message,
		},
		AssignedAt:      assignment.AssignedAt.Format("2006-01-02T15:04:05Z07:00"),
		StartedAt:       timePtrString(assignment.StartedAt),
		CompletedAt:     timePtrString(assignment.CompletedAt),
		PaymentStatus:   assignment.PaymentStatus,
		FullDays:        int32(assignment.FullDays),
		HalfDays:        int32(assignment.HalfDays),
		DaysWorked:      assignment.DaysWorked,
		PayableAmount:   assignment.PayableAmount,
		PayableCurrency: assignment.PayableCurrency,
		LogsFrozen:      assignment.LogsFrozen,
		CreatedAt:       assignment.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       assignment.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		AssignmentType:  assignment.AssignmentType,
		AgreedPayment: &workv1.AgreedPayment{
			RateType:         assignment.AgreedPayment.RateType,
			RateUnit:         assignment.AgreedPayment.RateUnit,
			RateMinor:        assignment.AgreedPayment.RateMinor,
			Currency:         assignment.AgreedPayment.Currency,
			FixedAmountMinor: assignment.AgreedPayment.FixedAmountMinor,
		},
		JoinedAt:             assignment.JoinedAt,
		WorkerProfileId:      stringPtr(assignment.WorkerProfileID),
		ContractorProfileId:  stringPtr(assignment.ContractorProfileID),
		CommittedWorkerCount: int32Ptr(int32(assignment.CommittedWorkerCount)),
	}
}

func toProtoDailyLog(log *model.DailyLog) *workv1.DailyLog {
	if log == nil {
		return nil
	}
	return &workv1.DailyLog{
		Id:           log.ID,
		WorkId:       log.WorkID,
		AssignmentId: log.AssignmentID,
		WorkerId:     log.WorkerID,
		WorkDate:     log.WorkDate,
		DayPortion:   log.DayPortion,
		Note:         log.Note,
		Status:       log.Status,
		SubmittedAt:  log.SubmittedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    log.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toProtoDailyLogs(logs []model.DailyLog) []*workv1.DailyLog {
	items := make([]*workv1.DailyLog, 0, len(logs))
	for i := range logs {
		items = append(items, toProtoDailyLog(&logs[i]))
	}
	return items
}

func toProtoPayment(payment *model.WorkPayment) *workv1.WorkPayment {
	if payment == nil {
		return nil
	}
	var receipt *string
	if payment.ReceiptStatus != "" {
		receipt = &payment.ReceiptStatus
	}
	return &workv1.WorkPayment{
		Id:             payment.ID,
		WorkId:         payment.WorkID,
		AssignmentId:   payment.AssignmentID,
		WorkerId:       payment.WorkerID,
		Method:         payment.Method,
		Amount:         payment.Amount,
		Currency:       payment.Currency,
		Reference:      payment.Reference,
		Status:         payment.Status,
		ReceiptStatus:  receipt,
		DisputeMessage: payment.DisputeMessage,
		RecordedAt:     payment.RecordedAt.Format("2006-01-02T15:04:05Z07:00"),
		ConfirmedAt:    timePtrString(payment.ConfirmedAt),
		DisputedAt:     timePtrString(payment.DisputedAt),
	}
}

func toProtoPayments(payments []model.WorkPayment) []*workv1.WorkPayment {
	items := make([]*workv1.WorkPayment, 0, len(payments))
	for i := range payments {
		items = append(items, toProtoPayment(&payments[i]))
	}
	return items
}
