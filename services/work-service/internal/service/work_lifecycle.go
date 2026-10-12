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

func (s *WorkService) StartWork(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkResponse, error) {
	log.Printf("start work started: user_id=%s work_id=%s", userID, workID)

	work, err := s.requireAssignedWorker(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if work.Status == model.StatusInProgress {
		return model.BuildWorkResponse(work), nil
	}
	if !model.CanTransition(work.Status, model.StatusInProgress) {
		log.Printf("start work rejected: user_id=%s work_id=%s current_status=%s", userID, workID, work.Status)
		return nil, ErrInvalidWorkStatus
	}
	if !startAllowed(work.StartDate, time.Now().UTC()) {
		return nil, ErrWorkNotReadyToStart
	}

	updated, err := s.transitionWorkStatus(
		ctx,
		workID,
		work.Status,
		model.StatusInProgress,
		model.StatusPublished,
		model.StatusAssigned,
	)
	if err != nil {
		log.Printf("start work failed: user_id=%s work_id=%s current_status=%s error=%v", userID, workID, work.Status, err)
		return nil, err
	}

	return model.BuildWorkResponse(updated), nil
}

func (s *WorkService) CompleteWork(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkResponse, error) {
	log.Printf("complete work started: user_id=%s work_id=%s", userID, workID)

	work, err := s.requireAssignedWorker(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if work.Status == model.StatusWorkCompleted || work.Status == model.StatusClosed {
		return model.BuildWorkResponse(work), nil
	}
	if !model.CanTransition(work.Status, model.StatusWorkCompleted) {
		return nil, ErrInvalidWorkStatus
	}

	updated, err := s.transitionWorkStatus(ctx, workID, work.Status, model.StatusWorkCompleted)
	if err != nil {
		log.Printf("complete work failed: user_id=%s work_id=%s error=%v", userID, workID, err)
		return nil, err
	}

	return model.BuildWorkResponse(updated), nil
}

func (s *WorkService) ConfirmWorkCompletion(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkResponse, error) {
	log.Printf("confirm work completion started: user_id=%s work_id=%s", userID, workID)

	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if work.Status == model.StatusClosed {
		return model.BuildWorkResponse(work), nil
	}
	if !model.CanTransition(work.Status, model.StatusClosed) {
		return nil, ErrInvalidWorkStatus
	}

	updated, err := s.transitionWorkStatus(ctx, workID, work.Status, model.StatusClosed)
	if err != nil {
		log.Printf("confirm work completion failed: user_id=%s work_id=%s error=%v", userID, workID, err)
		return nil, err
	}

	return model.BuildWorkResponse(updated), nil
}

func (s *WorkService) CancelWork(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkResponse, error) {
	log.Printf("cancel work started: user_id=%s work_id=%s", userID, workID)

	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if work.Status == model.StatusCancelled {
		return model.BuildWorkResponse(work), nil
	}
	if !model.CanTransition(work.Status, model.StatusCancelled) {
		return nil, ErrInvalidWorkStatus
	}

	updated, err := s.transitionWorkStatus(ctx, workID, work.Status, model.StatusCancelled, work.Status)
	if err != nil {
		log.Printf("cancel work failed: user_id=%s work_id=%s error=%v", userID, workID, err)
		return nil, err
	}

	return model.BuildWorkResponse(updated), nil
}

func (s *WorkService) markWorkAssigned(ctx context.Context, workID string) error {
	_, err := s.transitionWorkStatus(ctx, workID, model.StatusPublished, model.StatusAssigned, model.StatusPublished)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrInvalidWorkStatus) {
		return err
	}

	current, err := s.workRepository.FindByID(ctx, workID)
	if err != nil {
		return err
	}
	if current == nil {
		return ErrWorkNotFound
	}
	switch current.Status {
	case model.StatusAssigned, model.StatusInProgress, model.StatusWorkCompleted, model.StatusClosed:
		return nil
	default:
		return ErrInvalidWorkStatus
	}
}

func (s *WorkService) transitionWorkStatus(
	ctx context.Context,
	workID string,
	current string,
	to string,
	from ...string,
) (*model.WorkRecord, error) {
	if !model.CanTransition(current, to) {
		return nil, ErrInvalidWorkStatus
	}

	matchFrom := from
	if len(matchFrom) == 0 {
		matchFrom = []string{current}
	}

	updated, err := s.workRepository.UpdateStatus(ctx, workID, matchFrom, to)
	if err != nil {
		return nil, err
	}
	if updated != nil {
		return updated, nil
	}

	record, err := s.workRepository.FindByID(ctx, workID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrWorkNotFound
	}
	if model.NormalizeWorkStatus(record.Status) == model.NormalizeWorkStatus(to) {
		return record, nil
	}

	return nil, ErrInvalidWorkStatus
}

func (s *WorkService) requireAssignedWorker(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkRecord, error) {
	if strings.TrimSpace(workID) == "" {
		return nil, fmt.Errorf("work_id is required")
	}

	work, err := s.workRepository.FindByID(ctx, workID)
	if err != nil {
		return nil, err
	}
	if work == nil {
		return nil, ErrWorkNotFound
	}

	application, err := s.applicationRepository.FindByWorkIDAndStatus(ctx, workID, model.ApplicationStatusAccepted)
	if err != nil {
		return nil, err
	}
	if application == nil || application.WorkerID != userID {
		return nil, ErrNotAssignedWorker
	}

	work.Status = model.NormalizeWorkStatus(work.Status)
	return work, nil
}

func startAllowed(startDate *time.Time, now time.Time) bool {
	if startDate == nil {
		return true
	}

	scheduled := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return !scheduled.After(today)
}
