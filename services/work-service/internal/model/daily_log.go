package model

import (
	"strings"
	"time"
)

const (
	DayPortionFull      = "FULL_DAY"
	DayPortionHalf      = "HALF_DAY"
	DayPortionNotWorked = "NOT_WORKED"
	DailyLogSubmitted   = "SUBMITTED"

	HalfUnitsFull    = 2
	HalfUnitsHalf    = 1
	HalfUnitsNone    = 0
	MaxDeployedFloor = 20
)

func ValidDayPortion(portion string) bool {
	switch NormalizeDayPortion(portion) {
	case DayPortionFull, DayPortionHalf, DayPortionNotWorked:
		return true
	default:
		return false
	}
}

func NormalizeDayPortion(portion string) string {
	switch strings.ToUpper(strings.TrimSpace(portion)) {
	case DayPortionFull, "FULL", "FULLDAY":
		return DayPortionFull
	case DayPortionHalf, "HALF", "HALFDAY":
		return DayPortionHalf
	case DayPortionNotWorked, "NOTWORKED", "DIDNT_WORK", "DID_NOT_WORK":
		return DayPortionNotWorked
	default:
		return strings.ToUpper(strings.TrimSpace(portion))
	}
}

func DayPortionUnits(portion string) float64 {
	switch NormalizeDayPortion(portion) {
	case DayPortionFull:
		return 1
	case DayPortionHalf:
		return 0.5
	default:
		return 0
	}
}

func DayPortionHalfUnits(portion string) int {
	switch NormalizeDayPortion(portion) {
	case DayPortionFull:
		return HalfUnitsFull
	case DayPortionHalf:
		return HalfUnitsHalf
	default:
		return HalfUnitsNone
	}
}

type DailyLog struct {
	ID           string    `json:"id"`
	WorkID       string    `json:"work_id"`
	AssignmentID string    `json:"assignment_id"`
	WorkerID     string    `json:"worker_id"`
	WorkDate     string    `json:"work_date"`
	DayPortion   string    `json:"day_portion"`
	DayType      string    `json:"day_type"`
	PayableUnits float64   `json:"payable_units"`
	RecordedBy   string    `json:"recorded_by,omitempty"`
	Note         *string   `json:"note,omitempty"`
	Status       string    `json:"status"`
	SubmittedAt  time.Time `json:"submitted_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type WorkDayRecord = DailyLog

type SaveWorkDayRecordInput struct {
	AssignmentID string `json:"assignment_id"`
	DayType      string `json:"day_type"`
}

type SaveWorkDayRecordsRequest struct {
	WorkDate string                   `json:"work_date"`
	Records  []SaveWorkDayRecordInput `json:"records"`
}

type WorkerDayRecordItem struct {
	AssignmentID         string   `json:"assignment_id"`
	WorkerProfileID      string   `json:"worker_profile_id"`
	WorkerName           string   `json:"worker_name"`
	DayType              *string  `json:"day_type"`
	PayableUnits         *float64 `json:"payable_units"`
	AgreedRateMinor      int64    `json:"agreed_rate_minor"`
	EstimatedAmountMinor int64    `json:"estimated_amount_minor"`
}

type WorkDayRecordsResponse struct {
	WorkID   string                `json:"work_id"`
	WorkDate string                `json:"work_date"`
	Workers  []WorkerDayRecordItem `json:"workers"`
}

type WorkerWorkSummary struct {
	AssignmentID          string  `json:"assignment_id"`
	WorkerProfileID       string  `json:"worker_profile_id"`
	WorkerName            string  `json:"worker_name"`
	FullDays              int     `json:"full_days"`
	HalfDays              int     `json:"half_days"`
	NotWorkedDays         int     `json:"not_worked_days"`
	UnrecordedDays        int     `json:"unrecorded_days"`
	PayableDays           float64 `json:"payable_days"`
	AgreedRateMinor       int64   `json:"agreed_rate_minor"`
	CalculatedAmountMinor int64   `json:"calculated_amount_minor"`
	Currency              string  `json:"currency"`
}

type WorkerWorkSummaryResponse struct {
	WorkID  string              `json:"work_id"`
	Workers []WorkerWorkSummary `json:"workers"`
}

func SumDaysWorked(logs []DailyLog) (fullDays int, halfDays int, daysWorked float64) {
	notWorked := 0
	fullDays, halfDays, notWorked, daysWorked = SumDayTypes(logs)
	_ = notWorked
	return fullDays, halfDays, daysWorked
}

func SumDayTypes(logs []DailyLog) (fullDays int, halfDays int, notWorkedDays int, payableDays float64) {
	for _, log := range logs {
		switch NormalizeDayPortion(firstNonEmptyDay(log.DayType, log.DayPortion)) {
		case DayPortionFull:
			fullDays++
		case DayPortionHalf:
			halfDays++
		case DayPortionNotWorked:
			notWorkedDays++
		}
	}
	payableDays = float64(fullDays) + 0.5*float64(halfDays)
	return fullDays, halfDays, notWorkedDays, payableDays
}

func firstNonEmptyDay(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func PayableAmount(daysWorked float64, rate float64) float64 {
	if daysWorked <= 0 || rate <= 0 {
		return 0
	}
	return daysWorked * rate
}

func WorkDateInWindow(workDate time.Time, startDate *time.Time, durationValue *int, durationUnit *string, now time.Time) error {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	date := time.Date(workDate.Year(), workDate.Month(), workDate.Day(), 0, 0, 0, 0, time.UTC)
	if date.After(today) {
		return errWorkDate("work_date must not be in the future")
	}
	if startDate != nil {
		start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
		if date.Before(start) {
			return errWorkDate("work_date must not be before the work start date")
		}
		if end := workWindowEnd(start, durationValue, durationUnit); end != nil && date.After(*end) {
			return errWorkDate("work_date is outside the work duration")
		}
	}
	return nil
}

func DateOnOrAfterJoined(workDate time.Time, joinedAt string) error {
	joinedAt = strings.TrimSpace(joinedAt)
	if joinedAt == "" {
		return nil
	}
	joined, err := time.Parse("2006-01-02", joinedAt)
	if err != nil {
		return nil
	}
	date := time.Date(workDate.Year(), workDate.Month(), workDate.Day(), 0, 0, 0, 0, time.UTC)
	start := time.Date(joined.Year(), joined.Month(), joined.Day(), 0, 0, 0, 0, time.UTC)
	if date.Before(start) {
		return errWorkDate("work_date must not be before the worker joined_at")
	}
	return nil
}

func ExpectedWorkDates(joinedAt string, startDate *time.Time, durationValue *int, durationUnit *string, now time.Time) []string {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := today
	if startDate != nil {
		start = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	}
	if joinedAt = strings.TrimSpace(joinedAt); joinedAt != "" {
		if joined, err := time.Parse("2006-01-02", joinedAt); err == nil {
			joinedDay := time.Date(joined.Year(), joined.Month(), joined.Day(), 0, 0, 0, 0, time.UTC)
			if joinedDay.After(start) {
				start = joinedDay
			}
		}
	}
	end := today
	if startDate != nil {
		if windowEnd := workWindowEnd(
			time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC),
			durationValue,
			durationUnit,
		); windowEnd != nil && windowEnd.Before(end) {
			end = *windowEnd
		}
	}
	if start.After(end) {
		return nil
	}
	dates := make([]string, 0)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		dates = append(dates, day.Format("2006-01-02"))
	}
	return dates
}

func CountUnrecordedDays(expected []string, logs []DailyLog) int {
	recorded := make(map[string]struct{}, len(logs))
	for _, log := range logs {
		if strings.TrimSpace(log.WorkDate) != "" {
			recorded[log.WorkDate] = struct{}{}
		}
	}
	count := 0
	for _, date := range expected {
		if _, ok := recorded[date]; !ok {
			count++
		}
	}
	return count
}

func MaxDeployedWorkerCount(committed int) int {
	if committed < 1 {
		committed = 1
	}
	limit := committed * 3
	if limit < MaxDeployedFloor {
		return MaxDeployedFloor
	}
	return limit
}

func workWindowEnd(start time.Time, durationValue *int, durationUnit *string) *time.Time {
	if durationValue == nil || *durationValue < 1 || durationUnit == nil {
		return nil
	}
	var end time.Time
	switch strings.ToLower(strings.TrimSpace(*durationUnit)) {
	case DurationUnitDays:
		end = start.AddDate(0, 0, *durationValue-1)
	case DurationUnitWeeks:
		end = start.AddDate(0, 0, (*durationValue)*7-1)
	case DurationUnitMonths:
		end = start.AddDate(0, *durationValue, -1)
	default:
		return nil
	}
	return &end
}

type workDateError string

func (e workDateError) Error() string { return string(e) }

func errWorkDate(message string) error { return workDateError(message) }
