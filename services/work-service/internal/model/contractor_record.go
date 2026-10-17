package model

import "time"

type ContractorWorkRecord struct {
	ID                     string    `json:"id"`
	WorkID                 string    `json:"work_id"`
	ContractorAssignmentID string    `json:"contractor_assignment_id"`
	WorkDate               string    `json:"work_date"`
	DeployedWorkerCount    int       `json:"deployed_worker_count"`
	RecordedBy             string    `json:"recorded_by"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type SaveContractorWorkRecordRequest struct {
	WorkDate            string `json:"work_date"`
	DeployedWorkerCount int    `json:"deployed_worker_count"`
}

type ContractorWorkRecordItem struct {
	WorkDate            string `json:"work_date"`
	DeployedWorkerCount int    `json:"deployed_worker_count"`
	Shortfall           int    `json:"shortfall"`
}

type ContractorInfo struct {
	AssignmentID         string `json:"assignment_id"`
	ContractorProfileID  string `json:"contractor_profile_id"`
	ContractorName       string `json:"contractor_name"`
	CommittedWorkerCount int    `json:"committed_worker_count"`
}

type ContractorWorkRecordsResponse struct {
	WorkID     string                     `json:"work_id"`
	Contractor ContractorInfo             `json:"contractor"`
	Records    []ContractorWorkRecordItem `json:"records"`
}

type ContractorWorkSummary struct {
	WorkID                  string  `json:"work_id"`
	AssignmentID            string  `json:"assignment_id"`
	ContractorProfileID     string  `json:"contractor_profile_id"`
	ContractorName          string  `json:"contractor_name"`
	CommittedWorkerCount    int     `json:"committed_worker_count"`
	TotalRecordedDays       int     `json:"total_recorded_days"`
	TotalDeployedWorkerDays int     `json:"total_deployed_worker_days"`
	AverageDeployedWorkers  float64 `json:"average_deployed_workers"`
	TotalShortfall          int     `json:"total_shortfall"`
	RateType                string  `json:"rate_type"`
	RateMinor               int64   `json:"rate_minor"`
	CompletedQuantity       float64 `json:"completed_quantity,omitempty"`
	EstimatedAmountMinor    int64   `json:"estimated_amount_minor"`
	Currency                string  `json:"currency"`
}

func ContractorShortfall(committed int, deployed int) int {
	if committed <= deployed {
		return 0
	}
	return committed - deployed
}

func SumDeployedWorkerDays(records []ContractorWorkRecord) int {
	total := 0
	for _, record := range records {
		if record.DeployedWorkerCount > 0 {
			total += record.DeployedWorkerCount
		}
	}
	return total
}
