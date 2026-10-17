package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
)

func (s *WorkService) GetWorkDayRecords(
	ctx context.Context,
	userID string,
	workID string,
	workDate string,
) (*model.WorkDayRecordsResponse, error) {
	work, viewerAssignment, err := s.requireWorkViewer(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if model.NormalizeHiringMode(work.HiringMode) != model.HiringModeIndividual {
		return nil, ErrHiringModeMismatch
	}
	if strings.TrimSpace(workDate) == "" {
		latest, err := s.dailyLogRepository.LatestWorkDate(ctx, workID)
		if err != nil {
			return nil, err
		}
		if latest != "" {
			workDate = latest
		} else {
			workDate = time.Now().UTC().Format("2006-01-02")
		}
	}
	parsed, err := parseWorkDate(workDate)
	if err != nil {
		return nil, err
	}

	assignments, err := s.assignmentRepository.ListActiveByWorkID(ctx, workID)
	if err != nil {
		return nil, err
	}
	logs, err := s.dailyLogRepository.ListByWorkIDAndDate(ctx, workID, parsed.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	byAssignment := make(map[string]model.DailyLog, len(logs))
	for _, log := range logs {
		byAssignment[log.AssignmentID] = log
	}
	names, err := s.displayNames(ctx, assignmentPayeeIDs(assignments))
	if err != nil {
		return nil, err
	}

	items := make([]model.WorkerDayRecordItem, 0, len(assignments))
	for _, assignment := range assignments {
		if model.NormalizeAssignmentType(assignment.AssignmentType) == model.AssignmentTypeContractor {
			continue
		}
		if viewerAssignment != nil && viewerAssignment.ID != assignment.ID {
			continue
		}
		item := model.WorkerDayRecordItem{
			AssignmentID:    assignment.ID,
			WorkerProfileID: assignment.PayeeID(),
			WorkerName:      names[assignment.PayeeID()],
			AgreedRateMinor: assignment.AgreedPayment.RateMinor,
		}
		if log, ok := byAssignment[assignment.ID]; ok {
			dayType := model.NormalizeDayPortion(log.DayType)
			units := model.DayPortionUnits(dayType)
			item.DayType = &dayType
			item.PayableUnits = &units
			item.EstimatedAmountMinor = model.IndividualPayableMinor(
				boolToInt(dayType == model.DayPortionFull),
				boolToInt(dayType == model.DayPortionHalf),
				assignment.AgreedPayment.RateMinor,
			)
		} else {
			item.EstimatedAmountMinor = 0
		}
		items = append(items, item)
	}

	return &model.WorkDayRecordsResponse{
		WorkID:   workID,
		WorkDate: parsed.Format("2006-01-02"),
		Workers:  items,
	}, nil
}

func (s *WorkService) SaveWorkDayRecords(
	ctx context.Context,
	userID string,
	workID string,
	req model.SaveWorkDayRecordsRequest,
) (*model.WorkDayRecordsResponse, error) {
	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if model.WorkRecordsLocked(work.Status) {
		return nil, ErrInvalidWorkStatus
	}
	if model.NormalizeHiringMode(work.HiringMode) != model.HiringModeIndividual {
		return nil, ErrHiringModeMismatch
	}
	parsed, err := parseWorkDate(req.WorkDate)
	if err != nil {
		return nil, err
	}
	if err := model.WorkDateInWindow(parsed, work.StartDate, work.DurationValue, work.DurationUnit, time.Now().UTC()); err != nil {
		return nil, err
	}

	assignments, err := s.assignmentRepository.ListActiveByWorkID(ctx, workID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]model.WorkAssignment, len(assignments))
	for _, assignment := range assignments {
		byID[assignment.ID] = assignment
	}

	for _, input := range req.Records {
		assignment, ok := byID[strings.TrimSpace(input.AssignmentID)]
		if !ok || !model.IsActiveAssignment(assignment.Status) {
			return nil, fmt.Errorf("assignment_id is invalid")
		}
		if model.NormalizeAssignmentType(assignment.AssignmentType) != model.AssignmentTypeIndividual {
			return nil, ErrHiringModeMismatch
		}
		if err := model.DateOnOrAfterJoined(parsed, assignment.JoinedAt); err != nil {
			return nil, err
		}
		dayType := model.NormalizeDayPortion(input.DayType)
		if !model.ValidDayPortion(dayType) {
			return nil, fmt.Errorf("day_type must be FULL_DAY, HALF_DAY, or NOT_WORKED")
		}
		if err := s.rejectSettledRecordChange(ctx, assignment.ID); err != nil {
			return nil, err
		}
		if _, err := s.dailyLogRepository.Upsert(ctx, &model.DailyLog{
			WorkID:       workID,
			AssignmentID: assignment.ID,
			WorkerID:     assignment.PayeeID(),
			WorkDate:     parsed.Format("2006-01-02"),
			DayType:      dayType,
			DayPortion:   dayType,
			RecordedBy:   userID,
			Status:       model.DailyLogSubmitted,
		}); err != nil {
			return nil, err
		}
		if err := s.syncAssignmentFromRecords(ctx, work, assignment); err != nil {
			return nil, err
		}
	}

	if work.Status == model.StatusPaymentPending || work.Status == model.StatusCompleted {
		if _, err := s.ensurePaymentObligations(ctx, work); err != nil && !errors.Is(err, ErrPaymentAdjustmentRequired) {
			return nil, err
		}
	}

	return s.GetWorkDayRecords(ctx, userID, workID, parsed.Format("2006-01-02"))
}

func (s *WorkService) GetWorkerWorkSummary(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkerWorkSummaryResponse, error) {
	work, viewerAssignment, err := s.requireWorkViewer(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if model.NormalizeHiringMode(work.HiringMode) != model.HiringModeIndividual {
		return nil, ErrHiringModeMismatch
	}
	assignments, err := s.assignmentRepository.ListActiveByWorkID(ctx, workID)
	if err != nil {
		return nil, err
	}
	names, err := s.displayNames(ctx, assignmentPayeeIDs(assignments))
	if err != nil {
		return nil, err
	}

	summaries := make([]model.WorkerWorkSummary, 0, len(assignments))
	for _, assignment := range assignments {
		if model.NormalizeAssignmentType(assignment.AssignmentType) == model.AssignmentTypeContractor {
			continue
		}
		if viewerAssignment != nil && viewerAssignment.ID != assignment.ID {
			continue
		}
		logs, err := s.dailyLogRepository.ListByAssignmentID(ctx, assignment.ID)
		if err != nil {
			return nil, err
		}
		fullDays, halfDays, notWorked, payableDays := model.SumDayTypes(logs)
		expected := model.ExpectedWorkDates(assignment.JoinedAt, work.StartDate, work.DurationValue, work.DurationUnit, time.Now().UTC())
		currency := assignment.AgreedPayment.Currency
		if currency == "" {
			currency = "INR"
		}
		summaries = append(summaries, model.WorkerWorkSummary{
			AssignmentID:          assignment.ID,
			WorkerProfileID:       assignment.PayeeID(),
			WorkerName:            names[assignment.PayeeID()],
			FullDays:              fullDays,
			HalfDays:              halfDays,
			NotWorkedDays:         notWorked,
			UnrecordedDays:        model.CountUnrecordedDays(expected, logs),
			PayableDays:           payableDays,
			AgreedRateMinor:       assignment.AgreedPayment.RateMinor,
			CalculatedAmountMinor: model.IndividualPayableMinor(fullDays, halfDays, assignment.AgreedPayment.RateMinor),
			Currency:              currency,
		})
	}
	return &model.WorkerWorkSummaryResponse{WorkID: workID, Workers: summaries}, nil
}

func (s *WorkService) GetContractorWorkRecords(
	ctx context.Context,
	userID string,
	workID string,
	workDate string,
) (*model.ContractorWorkRecordsResponse, error) {
	work, assignment, err := s.requireContractorWork(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	_ = work

	var records []model.ContractorWorkRecord
	if strings.TrimSpace(workDate) != "" {
		parsed, err := parseWorkDate(workDate)
		if err != nil {
			return nil, err
		}
		records, err = s.contractorRecordRepository.ListByWorkIDAndDate(ctx, workID, parsed.Format("2006-01-02"))
		if err != nil {
			return nil, err
		}
	} else {
		records, err = s.contractorRecordRepository.ListByAssignmentID(ctx, assignment.ID)
		if err != nil {
			return nil, err
		}
	}

	names, err := s.displayNames(ctx, []string{assignment.PayeeID()})
	if err != nil {
		return nil, err
	}
	items := make([]model.ContractorWorkRecordItem, 0, len(records))
	for _, record := range records {
		items = append(items, model.ContractorWorkRecordItem{
			WorkDate:            record.WorkDate,
			DeployedWorkerCount: record.DeployedWorkerCount,
			Shortfall:           model.ContractorShortfall(assignment.CommittedWorkerCount, record.DeployedWorkerCount),
		})
	}
	return &model.ContractorWorkRecordsResponse{
		WorkID: workID,
		Contractor: model.ContractorInfo{
			AssignmentID:         assignment.ID,
			ContractorProfileID:  assignment.PayeeID(),
			ContractorName:       names[assignment.PayeeID()],
			CommittedWorkerCount: assignment.CommittedWorkerCount,
		},
		Records: items,
	}, nil
}

func (s *WorkService) SaveContractorWorkRecord(
	ctx context.Context,
	userID string,
	workID string,
	req model.SaveContractorWorkRecordRequest,
) (*model.ContractorWorkRecordsResponse, error) {
	work, assignment, err := s.requireContractorWork(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if model.WorkRecordsLocked(work.Status) {
		return nil, ErrInvalidWorkStatus
	}
	if !model.IsActiveAssignment(assignment.Status) {
		return nil, ErrInvalidAssignmentStatus
	}
	parsed, err := parseWorkDate(req.WorkDate)
	if err != nil {
		return nil, err
	}
	if err := model.WorkDateInWindow(parsed, work.StartDate, work.DurationValue, work.DurationUnit, time.Now().UTC()); err != nil {
		return nil, err
	}
	if req.DeployedWorkerCount < 0 {
		return nil, fmt.Errorf("deployed_worker_count must be 0 or greater")
	}
	if req.DeployedWorkerCount > model.MaxDeployedWorkerCount(assignment.CommittedWorkerCount) {
		return nil, fmt.Errorf("deployed_worker_count exceeds the allowed limit")
	}
	if err := s.rejectSettledRecordChange(ctx, assignment.ID); err != nil {
		return nil, err
	}

	if _, err := s.contractorRecordRepository.Upsert(ctx, &model.ContractorWorkRecord{
		WorkID:                 workID,
		ContractorAssignmentID: assignment.ID,
		WorkDate:               parsed.Format("2006-01-02"),
		DeployedWorkerCount:    req.DeployedWorkerCount,
		RecordedBy:             userID,
	}); err != nil {
		return nil, err
	}
	if err := s.syncAssignmentFromRecords(ctx, work, *assignment); err != nil {
		return nil, err
	}
	if work.Status == model.StatusPaymentPending || work.Status == model.StatusCompleted {
		if _, err := s.ensurePaymentObligations(ctx, work); err != nil && !errors.Is(err, ErrPaymentAdjustmentRequired) {
			return nil, err
		}
	}
	return s.GetContractorWorkRecords(ctx, userID, workID, "")
}

func (s *WorkService) GetContractorWorkSummary(
	ctx context.Context,
	userID string,
	workID string,
) (*model.ContractorWorkSummary, error) {
	work, assignment, err := s.requireContractorWork(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	records, err := s.contractorRecordRepository.ListByAssignmentID(ctx, assignment.ID)
	if err != nil {
		return nil, err
	}
	names, err := s.displayNames(ctx, []string{assignment.PayeeID()})
	if err != nil {
		return nil, err
	}
	workerDays := model.SumDeployedWorkerDays(records)
	shortfall := 0
	for _, record := range records {
		shortfall += model.ContractorShortfall(assignment.CommittedWorkerCount, record.DeployedWorkerCount)
	}
	avg := 0.0
	if len(records) > 0 {
		avg = float64(workerDays) / float64(len(records))
	}
	quantity := assignment.CompletedQuantity
	if quantity <= 0 {
		quantity = model.CompletedQuantityFromAttributes(work.Attributes)
	}
	currency := assignment.AgreedPayment.Currency
	if currency == "" {
		currency = "INR"
	}
	return &model.ContractorWorkSummary{
		WorkID:                  workID,
		AssignmentID:            assignment.ID,
		ContractorProfileID:     assignment.PayeeID(),
		ContractorName:          names[assignment.PayeeID()],
		CommittedWorkerCount:    assignment.CommittedWorkerCount,
		TotalRecordedDays:       len(records),
		TotalDeployedWorkerDays: workerDays,
		AverageDeployedWorkers:  avg,
		TotalShortfall:          shortfall,
		RateType:                assignment.AgreedPayment.RateType,
		RateMinor:               assignment.AgreedPayment.RateMinor,
		CompletedQuantity:       quantity,
		EstimatedAmountMinor:    model.AssignmentPayableMinor(*assignment, 0, 0, workerDays, quantity),
		Currency:                currency,
	}, nil
}

func (s *WorkService) GetWorkPayments(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkPaymentsResponse, error) {
	work, viewerAssignment, err := s.requireWorkViewer(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	obligations, err := s.ensurePaymentObligations(ctx, work)
	if err != nil {
		return nil, err
	}
	assignments, err := s.assignmentRepository.ListActiveByWorkID(ctx, workID)
	if err != nil {
		return nil, err
	}
	assignmentByID := make(map[string]model.WorkAssignment, len(assignments))
	for _, assignment := range assignments {
		assignmentByID[assignment.ID] = assignment
	}
	names, err := s.displayNames(ctx, assignmentPayeeIDs(assignments))
	if err != nil {
		return nil, err
	}

	items := make([]model.WorkPaymentItem, 0, len(obligations))
	var outstanding int64
	for _, obligation := range obligations {
		if viewerAssignment != nil && viewerAssignment.ID != obligation.AssignmentID {
			continue
		}
		transactions, err := s.obligationRepository.ListTransactionsByObligationID(ctx, obligation.ID)
		if err != nil {
			return nil, err
		}
		remaining := obligation.PayableAmountMinor - obligation.PaidAmountMinor
		if remaining < 0 {
			remaining = 0
		}
		if !model.PaymentObligationResolved(obligation) {
			outstanding += remaining
		}
		payeeName := names[obligation.PayeeID]
		rateType := ""
		if assignment, ok := assignmentByID[obligation.AssignmentID]; ok {
			rateType = assignment.AgreedPayment.RateType
			if payeeName == "" {
				payeeName = names[assignment.PayeeID()]
			}
		}
		obligation.PayeeName = payeeName
		items = append(items, model.WorkPaymentItem{
			Obligation:           obligation,
			AssignmentID:         obligation.AssignmentID,
			PayeeType:            obligation.PayeeType,
			PayeeID:              obligation.PayeeID,
			PayeeName:            payeeName,
			RateType:             rateType,
			PayableAmountMinor:   obligation.PayableAmountMinor,
			PaidAmountMinor:      obligation.PaidAmountMinor,
			RemainingAmountMinor: remaining,
			Status:               obligation.Status,
			Transactions:         transactions,
		})
	}

	if _, err := s.maybeCloseWorkIfSettled(ctx, workID); err != nil {
		log.Printf("close settled work failed: work_id=%s error=%v", workID, err)
	}

	return &model.WorkPaymentsResponse{
		WorkID:           workID,
		HiringMode:       model.NormalizeHiringMode(work.HiringMode),
		OutstandingMinor: outstanding,
		Payments:         items,
	}, nil
}

func (s *WorkService) RecordObligationPayment(
	ctx context.Context,
	userID string,
	obligationID string,
	req model.RecordObligationPaymentRequest,
) (*model.PaymentTransaction, error) {
	if strings.TrimSpace(obligationID) == "" {
		return nil, fmt.Errorf("payment_id is required")
	}
	obligation, err := s.obligationRepository.FindByID(ctx, obligationID)
	if err != nil {
		return nil, err
	}
	if obligation == nil {
		return nil, ErrAssignmentNotFound
	}
	work, err := s.requireWorkOwner(ctx, userID, obligation.WorkID)
	if err != nil {
		return nil, err
	}
	if work.Status != model.StatusPaymentPending && work.Status != model.StatusCompleted {
		return nil, ErrInvalidWorkStatus
	}

	method := model.NormalizePaymentMethod(req.PaymentMethod)
	if !model.ValidPaymentMethod(method) {
		return nil, ErrInvalidPaymentMethod
	}
	if req.AmountMinor <= 0 {
		return nil, fmt.Errorf("amount_minor must be greater than 0")
	}
	remaining := obligation.PayableAmountMinor - obligation.PaidAmountMinor
	if req.AmountMinor > remaining {
		return nil, ErrOverpayment
	}

	paidAt := time.Now().UTC()
	if req.PaidAt != nil && strings.TrimSpace(*req.PaidAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.PaidAt))
		if err != nil {
			return nil, fmt.Errorf("paid_at must be an RFC3339 timestamp")
		}
		paidAt = parsed.UTC()
	}

	if existing, err := s.obligationRepository.FindRecentDuplicate(ctx, obligation.ID, req.AmountMinor, method, paidAt); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	var reference *string
	if req.Reference != nil {
		trimmed := strings.TrimSpace(*req.Reference)
		if trimmed != "" {
			reference = &trimmed
		}
	}

	saved, err := s.obligationRepository.InsertTransaction(ctx, &model.PaymentTransaction{
		PaymentObligationID: obligation.ID,
		WorkID:              obligation.WorkID,
		AssignmentID:        obligation.AssignmentID,
		AmountMinor:         req.AmountMinor,
		PaymentMethod:       method,
		Reference:           reference,
		RecordedBy:          userID,
		PaidAt:              paidAt,
	})
	if err != nil {
		return nil, err
	}

	nextPaid := obligation.PaidAmountMinor + req.AmountMinor
	status := model.ObligationStatus(obligation.PayableAmountMinor, nextPaid)
	if _, err := s.obligationRepository.AddPayment(ctx, obligation.ID, req.AmountMinor, status); err != nil {
		return nil, err
	}
	if err := s.assignmentRepository.UpdatePaymentStatus(ctx, obligation.AssignmentID, status); err != nil {
		log.Printf("update assignment payment status failed: assignment_id=%s error=%v", obligation.AssignmentID, err)
	}
	if _, err := s.maybeCloseWorkIfSettled(ctx, obligation.WorkID); err != nil {
		log.Printf("close settled work failed: work_id=%s error=%v", obligation.WorkID, err)
	}
	return saved, nil
}

func (s *WorkService) UpsertDailyLog(
	ctx context.Context,
	userID string,
	assignmentID string,
	workDate string,
	req model.UpsertDailyLogRequest,
) (*model.DailyLog, error) {
	assignment, work, err := s.requireAssignmentOwnerOrWorker(ctx, userID, assignmentID)
	if err != nil {
		return nil, err
	}
	if model.WorkRecordsLocked(work.Status) {
		return nil, ErrInvalidWorkStatus
	}
	if err := s.rejectSettledRecordChange(ctx, assignment.ID); err != nil {
		return nil, err
	}

	parsed, err := parseWorkDate(workDate)
	if err != nil {
		return nil, err
	}
	if err := model.WorkDateInWindow(parsed, work.StartDate, work.DurationValue, work.DurationUnit, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := model.DateOnOrAfterJoined(parsed, assignment.JoinedAt); err != nil {
		return nil, err
	}

	portion := model.NormalizeDayPortion(req.DayPortion)
	if !model.ValidDayPortion(portion) {
		return nil, fmt.Errorf("day_portion must be FULL_DAY, HALF_DAY, or NOT_WORKED")
	}

	var note *string
	if req.Note != nil {
		trimmed := strings.TrimSpace(*req.Note)
		if trimmed != "" {
			note = &trimmed
		}
	}

	return s.dailyLogRepository.Upsert(ctx, &model.DailyLog{
		WorkID:       assignment.WorkID,
		AssignmentID: assignment.ID,
		WorkerID:     assignment.PayeeID(),
		WorkDate:     parsed.Format("2006-01-02"),
		DayPortion:   portion,
		DayType:      portion,
		RecordedBy:   userID,
		Note:         note,
		Status:       model.DailyLogSubmitted,
	})
}

func (s *WorkService) ListAssignmentDailyLogs(
	ctx context.Context,
	userID string,
	assignmentID string,
) ([]model.DailyLog, error) {
	assignment, _, err := s.requireAssignmentVisible(ctx, userID, assignmentID)
	if err != nil {
		return nil, err
	}
	return s.dailyLogRepository.ListByAssignmentID(ctx, assignment.ID)
}

func (s *WorkService) ListWorkDailyLogs(
	ctx context.Context,
	userID string,
	workID string,
) ([]model.DailyLog, error) {
	if _, err := s.requireWorkOwner(ctx, userID, workID); err != nil {
		return nil, err
	}
	return s.dailyLogRepository.ListByWorkID(ctx, workID)
}

func (s *WorkService) GetWorkSettlement(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkSettlement, error) {
	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if _, err := s.ensurePaymentObligations(ctx, work); err != nil && !errors.Is(err, ErrPaymentAdjustmentRequired) && !errors.Is(err, ErrNoActiveAssignment) {
		return nil, err
	}

	assignments, err := s.assignmentRepository.ListActiveByWorkID(ctx, workID)
	if err != nil {
		return nil, err
	}

	workers := make([]model.WorkerSettlement, 0, len(assignments))
	for _, assignment := range assignments {
		logs, err := s.dailyLogRepository.ListByAssignmentID(ctx, assignment.ID)
		if err != nil {
			return nil, err
		}
		payments, err := s.legacyPaymentsForAssignment(ctx, assignment.ID)
		if err != nil {
			return nil, err
		}
		fullDays, halfDays, _, daysWorked := model.SumDayTypes(logs)
		payableMinor, currency, _, err := s.calculateAssignmentPayable(ctx, work, assignment)
		if err != nil {
			return nil, err
		}
		paymentStatus := assignment.PaymentStatus
		if obligation, err := s.obligationRepository.FindByAssignmentID(ctx, assignment.ID); err == nil && obligation != nil {
			paymentStatus = obligation.Status
			if obligation.PaidAmountMinor > 0 {
				payableMinor = obligation.PayableAmountMinor
			}
		}
		assignment.FullDays = fullDays
		assignment.HalfDays = halfDays
		assignment.DaysWorked = daysWorked
		assignment.PayableAmount = model.MinorToRupees(payableMinor)
		assignment.PayableCurrency = currency
		assignment.PaymentStatus = paymentStatus
		workers = append(workers, model.WorkerSettlement{
			Assignment:    assignment,
			Logs:          logs,
			Payments:      payments,
			FullDays:      fullDays,
			HalfDays:      halfDays,
			DaysWorked:    daysWorked,
			PayableAmount: model.MinorToRupees(payableMinor),
			Currency:      currency,
			PaymentStatus: paymentStatus,
		})
	}

	if updated, err := s.maybeCloseWorkIfSettled(ctx, workID); err != nil {
		log.Printf("close settled work failed: work_id=%s error=%v", workID, err)
	} else if updated != nil {
		work = updated
	}

	return &model.WorkSettlement{
		Work:    model.BuildWorkResponse(work),
		Workers: workers,
	}, nil
}

func (s *WorkService) RecordAssignmentPayment(
	ctx context.Context,
	userID string,
	workID string,
	assignmentID string,
	req model.RecordPaymentRequest,
) (*model.WorkPayment, error) {
	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	assignment, err := s.assignmentRepository.FindByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if assignment == nil || assignment.WorkID != workID || assignment.Status == model.AssignmentStatusCancelled {
		return nil, ErrAssignmentNotFound
	}
	obligations, err := s.ensurePaymentObligations(ctx, work)
	if err != nil {
		return nil, err
	}
	var obligation *model.PaymentObligation
	for i := range obligations {
		if obligations[i].AssignmentID == assignmentID {
			obligation = &obligations[i]
			break
		}
	}
	if obligation == nil {
		return nil, ErrAssignmentNotFound
	}

	amountMinor := model.RupeesToMinor(req.Amount)
	if amountMinor <= 0 {
		amountMinor = obligation.PayableAmountMinor - obligation.PaidAmountMinor
	}
	saved, err := s.RecordObligationPayment(ctx, userID, obligation.ID, model.RecordObligationPaymentRequest{
		AmountMinor:   amountMinor,
		PaymentMethod: req.Method,
		Reference:     req.Reference,
	})
	if err != nil {
		return nil, err
	}
	return transactionToLegacyPayment(saved, assignment.PayeeID()), nil
}

func (s *WorkService) ListAssignmentPayments(
	ctx context.Context,
	userID string,
	assignmentID string,
) ([]model.WorkPayment, error) {
	if _, _, err := s.requireAssignmentVisible(ctx, userID, assignmentID); err != nil {
		return nil, err
	}
	return s.legacyPaymentsForAssignment(ctx, assignmentID)
}

func (s *WorkService) ConfirmAssignmentPayment(
	ctx context.Context,
	userID string,
	assignmentID string,
	paymentID string,
) (*model.WorkPayment, error) {
	return s.updatePaymentReceipt(ctx, userID, assignmentID, paymentID, model.PaymentReceiptConfirmed, nil)
}

func (s *WorkService) DisputeAssignmentPayment(
	ctx context.Context,
	userID string,
	assignmentID string,
	paymentID string,
	message *string,
) (*model.WorkPayment, error) {
	updated, err := s.updatePaymentReceipt(ctx, userID, assignmentID, paymentID, model.PaymentReceiptDisputed, message)
	if err != nil {
		return nil, err
	}
	if err := s.assignmentRepository.UpdatePaymentStatus(ctx, updated.AssignmentID, model.PaymentStatusDisputed); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *WorkService) updatePaymentReceipt(
	ctx context.Context,
	userID string,
	assignmentID string,
	paymentID string,
	receiptStatus string,
	message *string,
) (*model.WorkPayment, error) {
	assignment, txn, legacy, err := s.resolveWorkerReceiptTarget(ctx, userID, assignmentID, paymentID)
	if err != nil {
		return nil, err
	}

	var dispute *string
	if message != nil {
		trimmed := strings.TrimSpace(*message)
		if trimmed != "" {
			dispute = &trimmed
		}
	}

	var receipt *model.WorkPayment
	if txn != nil {
		if txn.ReceiptStatus == receiptStatus {
			receipt = transactionToLegacyPayment(txn, assignment.PayeeID())
		} else {
			updated, err := s.obligationRepository.UpdateTransactionReceipt(ctx, txn.ID, receiptStatus, dispute)
			if err != nil {
				return nil, err
			}
			if updated == nil {
				return nil, ErrAssignmentNotFound
			}
			receipt = transactionToLegacyPayment(updated, assignment.PayeeID())
		}
	} else {
		if legacy == nil {
			return nil, ErrAssignmentNotFound
		}
		if legacy.ReceiptStatus == receiptStatus {
			receipt = legacy
		} else {
			updated, err := s.paymentRepository.UpdateReceipt(ctx, legacy.ID, receiptStatus, dispute)
			if err != nil {
				return nil, err
			}
			if updated == nil {
				return nil, ErrAssignmentNotFound
			}
			receipt = updated
		}
	}

	if receiptStatus == model.PaymentReceiptConfirmed {
		s.syncSettledAssignment(ctx, assignment)
	}
	return receipt, nil
}

func (s *WorkService) syncSettledAssignment(ctx context.Context, assignment *model.WorkAssignment) {
	if assignment == nil {
		return
	}
	if obligation, err := s.obligationRepository.FindByAssignmentID(ctx, assignment.ID); err == nil && obligation != nil {
		if err := s.assignmentRepository.UpdatePaymentStatus(ctx, assignment.ID, obligation.Status); err != nil {
			log.Printf("update assignment payment status failed: assignment_id=%s error=%v", assignment.ID, err)
		}
	}
	if _, err := s.maybeCloseWorkIfSettled(ctx, assignment.WorkID); err != nil {
		log.Printf("close settled work failed: work_id=%s error=%v", assignment.WorkID, err)
	}
}

func (s *WorkService) resolveWorkerReceiptTarget(
	ctx context.Context,
	userID string,
	assignmentID string,
	paymentID string,
) (*model.WorkAssignment, *model.PaymentTransaction, *model.WorkPayment, error) {
	assignment, err := s.assignmentRepository.FindByID(ctx, assignmentID)
	if err != nil {
		return nil, nil, nil, err
	}

	txn, err := s.findReceiptTransaction(ctx, assignmentID, paymentID)
	if err != nil {
		return nil, nil, nil, err
	}
	if assignment == nil && txn != nil {
		assignment, err = s.assignmentRepository.FindByID(ctx, txn.AssignmentID)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	if assignment == nil && s.obligationRepository != nil {
		if obligation, err := s.obligationRepository.FindByID(ctx, firstNonEmptyID(assignmentID, paymentID)); err != nil {
			return nil, nil, nil, err
		} else if obligation != nil {
			assignment, err = s.assignmentRepository.FindByID(ctx, obligation.AssignmentID)
			if err != nil {
				return nil, nil, nil, err
			}
			if txn == nil {
				txn, err = s.obligationRepository.LatestTransactionByObligationID(ctx, obligation.ID)
				if err != nil {
					return nil, nil, nil, err
				}
			}
		}
	}

	if assignment == nil || assignment.Status == model.AssignmentStatusCancelled {
		return nil, nil, nil, ErrAssignmentNotFound
	}
	if assignment.WorkerID != userID && assignment.PayeeID() != userID {
		return nil, nil, nil, ErrNotAssignedWorker
	}

	if txn != nil {
		if txn.AssignmentID != "" && txn.AssignmentID != assignment.ID {
			return nil, nil, nil, ErrAssignmentNotFound
		}
		return assignment, txn, nil, nil
	}

	legacy, err := s.paymentRepository.FindByID(ctx, paymentID)
	if err != nil {
		return nil, nil, nil, err
	}
	if legacy == nil || legacy.AssignmentID != assignment.ID {
		return nil, nil, nil, ErrAssignmentNotFound
	}
	return assignment, nil, legacy, nil
}

func (s *WorkService) findReceiptTransaction(
	ctx context.Context,
	assignmentID string,
	paymentID string,
) (*model.PaymentTransaction, error) {
	if s.obligationRepository == nil {
		return nil, nil
	}
	if strings.TrimSpace(paymentID) != "" {
		txn, err := s.obligationRepository.FindTransactionByID(ctx, paymentID)
		if err != nil {
			return nil, err
		}
		if txn != nil {
			return txn, nil
		}
		obligation, err := s.obligationRepository.FindByID(ctx, paymentID)
		if err != nil {
			return nil, err
		}
		if obligation != nil {
			return s.obligationRepository.LatestTransactionByObligationID(ctx, obligation.ID)
		}
	}
	if strings.TrimSpace(assignmentID) == "" {
		return nil, nil
	}
	obligation, err := s.obligationRepository.FindByAssignmentID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if obligation != nil {
		return s.obligationRepository.LatestTransactionByObligationID(ctx, obligation.ID)
	}
	return s.obligationRepository.LatestTransactionByAssignmentID(ctx, assignmentID)
}

func firstNonEmptyID(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *WorkService) requireAssignmentVisible(
	ctx context.Context,
	userID string,
	assignmentID string,
) (*model.WorkAssignment, *model.WorkRecord, error) {
	return s.requireAssignmentOwnerOrWorker(ctx, userID, assignmentID)
}

func (s *WorkService) requireAssignmentOwnerOrWorker(
	ctx context.Context,
	userID string,
	assignmentID string,
) (*model.WorkAssignment, *model.WorkRecord, error) {
	assignment, err := s.assignmentRepository.FindByID(ctx, assignmentID)
	if err != nil {
		return nil, nil, err
	}
	if assignment == nil {
		return nil, nil, ErrAssignmentNotFound
	}
	work, err := s.workRepository.FindByID(ctx, assignment.WorkID)
	if err != nil {
		return nil, nil, err
	}
	if work == nil {
		return nil, nil, ErrWorkNotFound
	}
	if assignment.WorkerID != userID && assignment.PayeeID() != userID && work.UserID != userID {
		return nil, nil, ErrNotAssignedWorker
	}
	work.Status = model.NormalizeWorkStatus(work.Status)
	return assignment, work, nil
}

func (s *WorkService) requireWorkViewer(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkRecord, *model.WorkAssignment, error) {
	if strings.TrimSpace(workID) == "" {
		return nil, nil, fmt.Errorf("work_id is required")
	}
	work, err := s.workRepository.FindByID(ctx, workID)
	if err != nil {
		return nil, nil, err
	}
	if work == nil {
		return nil, nil, ErrWorkNotFound
	}
	work.Status = model.NormalizeWorkStatus(work.Status)
	if work.UserID == userID {
		return work, nil, nil
	}
	assignment, err := s.assignmentRepository.FindByWorkIDAndWorkerID(ctx, workID, userID)
	if err != nil {
		return nil, nil, err
	}
	if assignment == nil || !model.IsActiveAssignment(assignment.Status) {
		return nil, nil, ErrNotWorkOwner
	}
	return work, assignment, nil
}

func (s *WorkService) syncAssignmentFromRecords(
	ctx context.Context,
	work *model.WorkRecord,
	assignment model.WorkAssignment,
) error {
	payable, currency, _, err := s.calculateAssignmentPayable(ctx, work, assignment)
	if err != nil {
		return err
	}
	fullDays, halfDays := 0, 0
	daysWorked := 0.0
	if model.NormalizeAssignmentType(assignment.AssignmentType) == model.AssignmentTypeContractor {
		records, err := s.contractorRecordRepository.ListByAssignmentID(ctx, assignment.ID)
		if err != nil {
			return err
		}
		daysWorked = float64(model.SumDeployedWorkerDays(records))
	} else {
		logs, err := s.dailyLogRepository.ListByAssignmentID(ctx, assignment.ID)
		if err != nil {
			return err
		}
		fullDays, halfDays, _, daysWorked = model.SumDayTypes(logs)
	}
	paymentStatus := assignment.PaymentStatus
	if obligation, err := s.obligationRepository.FindByAssignmentID(ctx, assignment.ID); err != nil {
		return err
	} else if obligation != nil {
		paymentStatus = obligation.Status
	}
	return s.assignmentRepository.UpdateLiveSettlement(
		ctx,
		assignment.ID,
		fullDays,
		halfDays,
		daysWorked,
		model.MinorToRupees(payable),
		currency,
		paymentStatus,
	)
}

func (s *WorkService) hydrateAssignmentForRead(
	ctx context.Context,
	assignment *model.WorkAssignment,
) error {
	if assignment == nil {
		return nil
	}
	work, err := s.workRepository.FindByID(ctx, assignment.WorkID)
	if err != nil || work == nil {
		return err
	}
	payable, currency, _, err := s.calculateAssignmentPayable(ctx, work, *assignment)
	if err != nil {
		return err
	}
	if model.NormalizeAssignmentType(assignment.AssignmentType) != model.AssignmentTypeContractor {
		logs, err := s.dailyLogRepository.ListByAssignmentID(ctx, assignment.ID)
		if err != nil {
			return err
		}
		fullDays, halfDays, _, daysWorked := model.SumDayTypes(logs)
		assignment.FullDays = fullDays
		assignment.HalfDays = halfDays
		assignment.DaysWorked = daysWorked
	} else {
		records, err := s.contractorRecordRepository.ListByAssignmentID(ctx, assignment.ID)
		if err != nil {
			return err
		}
		assignment.DaysWorked = float64(model.SumDeployedWorkerDays(records))
	}
	assignment.PayableAmount = model.MinorToRupees(payable)
	if currency != "" {
		assignment.PayableCurrency = currency
	}
	if obligation, err := s.obligationRepository.FindByAssignmentID(ctx, assignment.ID); err != nil {
		return err
	} else if obligation != nil {
		assignment.PaymentStatus = obligation.Status
	}
	return nil
}

func (s *WorkService) requireContractorWork(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkRecord, *model.WorkAssignment, error) {
	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, nil, err
	}
	if model.NormalizeHiringMode(work.HiringMode) != model.HiringModeContractor {
		return nil, nil, ErrHiringModeMismatch
	}
	assignment, err := s.assignmentRepository.FindActiveContractorByWorkID(ctx, workID)
	if err != nil {
		return nil, nil, err
	}
	if assignment == nil {
		return nil, nil, ErrContractorAssignmentRequired
	}
	return work, assignment, nil
}

func (s *WorkService) ensurePaymentObligations(
	ctx context.Context,
	work *model.WorkRecord,
) ([]model.PaymentObligation, error) {
	assignments, err := s.assignmentRepository.ListActiveByWorkID(ctx, work.ID)
	if err != nil {
		return nil, err
	}
	if len(assignments) == 0 {
		return nil, ErrNoActiveAssignment
	}

	saved := make([]model.PaymentObligation, 0, len(assignments))
	for _, assignment := range assignments {
		payable, currency, payeeType, err := s.calculateAssignmentPayable(ctx, work, assignment)
		if err != nil {
			return nil, err
		}
		existing, err := s.obligationRepository.FindByAssignmentID(ctx, assignment.ID)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.PaidAmountMinor > 0 && existing.PayableAmountMinor != payable {
			return nil, ErrPaymentAdjustmentRequired
		}
		status := model.ObligationStatus(payable, 0)
		if existing != nil {
			status = model.ObligationStatus(payable, existing.PaidAmountMinor)
		}
		record, err := s.obligationRepository.UpsertForAssignment(ctx, &model.PaymentObligation{
			WorkID:             work.ID,
			AssignmentID:       assignment.ID,
			PayeeType:          payeeType,
			PayeeID:            assignment.PayeeID(),
			Currency:           currency,
			PayableAmountMinor: payable,
			Status:             status,
		})
		if err != nil {
			return nil, err
		}
		saved = append(saved, *record)
		if err := s.syncAssignmentFromRecords(ctx, work, assignment); err != nil {
			return nil, err
		}
	}
	return saved, nil
}

func (s *WorkService) calculateAssignmentPayable(
	ctx context.Context,
	work *model.WorkRecord,
	assignment model.WorkAssignment,
) (int64, string, string, error) {
	currency := assignment.AgreedPayment.Currency
	if currency == "" {
		currency = "INR"
	}
	if model.NormalizeAssignmentType(assignment.AssignmentType) == model.AssignmentTypeContractor {
		records, err := s.contractorRecordRepository.ListByAssignmentID(ctx, assignment.ID)
		if err != nil {
			return 0, "", "", err
		}
		quantity := assignment.CompletedQuantity
		if quantity <= 0 {
			quantity = model.CompletedQuantityFromAttributes(work.Attributes)
		}
		payable := model.AssignmentPayableMinor(assignment, 0, 0, model.SumDeployedWorkerDays(records), quantity)
		return payable, currency, model.PayeeTypeContractor, nil
	}
	logs, err := s.dailyLogRepository.ListByAssignmentID(ctx, assignment.ID)
	if err != nil {
		return 0, "", "", err
	}
	fullDays, halfDays, _, _ := model.SumDayTypes(logs)
	payable := model.AssignmentPayableMinor(assignment, fullDays, halfDays, 0, 0)
	return payable, currency, model.PayeeTypeWorker, nil
}

func (s *WorkService) rejectSettledRecordChange(ctx context.Context, assignmentID string) error {
	obligation, err := s.obligationRepository.FindByAssignmentID(ctx, assignmentID)
	if err != nil {
		return err
	}
	if obligation != nil && obligation.PaidAmountMinor > 0 {
		return ErrPaymentAdjustmentRequired
	}
	if obligation != nil && model.ObligationResolved(obligation.Status) && obligation.PayableAmountMinor > 0 {
		return ErrPaymentAdjustmentRequired
	}
	return nil
}

func (s *WorkService) displayNames(ctx context.Context, userIDs []string) (map[string]string, error) {
	if s.workOwnerRepository == nil {
		return map[string]string{}, nil
	}
	return s.workOwnerRepository.FindDisplayNames(ctx, userIDs)
}

func (s *WorkService) legacyPaymentsForAssignment(ctx context.Context, assignmentID string) ([]model.WorkPayment, error) {
	if s.obligationRepository == nil {
		return s.paymentRepository.ListByAssignmentID(ctx, assignmentID)
	}
	obligation, err := s.obligationRepository.FindByAssignmentID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if obligation == nil {
		return []model.WorkPayment{}, nil
	}
	transactions, err := s.obligationRepository.ListTransactionsByObligationID(ctx, obligation.ID)
	if err != nil {
		return nil, err
	}
	items := make([]model.WorkPayment, 0, len(transactions))
	for _, txn := range transactions {
		items = append(items, *transactionToLegacyPayment(&txn, obligation.PayeeID))
	}
	return items, nil
}

func assignmentPayeeIDs(assignments []model.WorkAssignment) []string {
	ids := make([]string, 0, len(assignments))
	seen := make(map[string]struct{}, len(assignments))
	for _, assignment := range assignments {
		id := assignment.PayeeID()
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func parseWorkDate(raw string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, fmt.Errorf("work_date must be in YYYY-MM-DD format")
	}
	return parsed, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func transactionToLegacyPayment(txn *model.PaymentTransaction, payeeID string) *model.WorkPayment {
	if txn == nil {
		return nil
	}
	return &model.WorkPayment{
		ID:             txn.ID,
		WorkID:         txn.WorkID,
		AssignmentID:   txn.AssignmentID,
		WorkerID:       payeeID,
		Method:         txn.PaymentMethod,
		Amount:         model.MinorToRupees(txn.AmountMinor),
		AmountMinor:    txn.AmountMinor,
		Currency:       "INR",
		Reference:      txn.Reference,
		Status:         model.PaymentStatusPaid,
		ReceiptStatus:  txn.ReceiptStatus,
		DisputeMessage: txn.DisputeMessage,
		RecordedAt:     txn.PaidAt,
		ConfirmedAt:    txn.ConfirmedAt,
		DisputedAt:     txn.DisputedAt,
	}
}
