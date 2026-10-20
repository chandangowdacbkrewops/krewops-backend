package model

import "testing"

func TestRemainingWorkers(t *testing.T) {
	if RemainingWorkers(10, 0) != 10 {
		t.Fatal("posted work should have all seats remaining")
	}
	if RemainingWorkers(10, 6) != 4 {
		t.Fatal("accepting 6 of 10 should leave 4")
	}
	if RemainingWorkers(10, 10) != 0 {
		t.Fatal("full crew should leave 0")
	}
	if RemainingWorkers(10, 12) != 0 {
		t.Fatal("over-assigned count should clamp to 0")
	}
}

func TestTotalAssignedWorkersCountsHeads(t *testing.T) {
	assignments := []WorkAssignment{
		{Status: AssignmentStatusAssigned, AssignmentType: AssignmentTypeContractor, AssignedWorkerCount: 6},
		{Status: AssignmentStatusAssigned, AssignmentType: AssignmentTypeIndividual},
		{Status: AssignmentStatusAssigned, AssignmentType: AssignmentTypeIndividual},
		{Status: AssignmentStatusCancelled, AssignmentType: AssignmentTypeIndividual},
	}
	if got := TotalAssignedWorkers(assignments); got != 8 {
		t.Fatalf("got %d assigned workers, want 8", got)
	}
}

func TestAssignmentHeadcountFallsBackToCommitted(t *testing.T) {
	assignment := WorkAssignment{
		Status:               AssignmentStatusAssigned,
		AssignmentType:       AssignmentTypeContractor,
		CommittedWorkerCount: 6,
	}
	if got := AssignmentHeadcount(assignment); got != 6 {
		t.Fatalf("got %d, want 6", got)
	}
}

func TestApplicationSeatCount(t *testing.T) {
	contractor := WorkApplication{ApplicantType: ApplicantTypeContractor, OfferedWorkerCount: 6}
	if ApplicationSeatCount(contractor) != 6 {
		t.Fatal("contractor offer should be the seat count")
	}
	individual := WorkApplication{ApplicantType: ApplicantTypeIndividual, OfferedWorkerCount: 4}
	if ApplicationSeatCount(individual) != 1 {
		t.Fatal("individual applications always claim 1 seat")
	}
}

func TestIndividualRecordAssignmentsKeepsMixedCrew(t *testing.T) {
	assignments := []WorkAssignment{
		{
			ID:                  "contractor-1",
			Status:              AssignmentStatusAssigned,
			AssignmentType:      AssignmentTypeContractor,
			AssignedWorkerCount: 6,
		},
		{
			ID:             "individual-1",
			Status:         AssignmentStatusAssigned,
			AssignmentType: AssignmentTypeIndividual,
		},
	}
	got := IndividualRecordAssignments(assignments)
	if len(got) != 1 || got[0].ID != "individual-1" {
		t.Fatalf("mixed crew should list the individual worker, got %#v", got)
	}
	contractor := PickContractorAssignment(assignments)
	if contractor == nil || contractor.ID != "contractor-1" {
		t.Fatalf("mixed crew should keep the contractor crew, got %#v", contractor)
	}
}

func TestIndividualRecordAssignmentsIncludesMislabeledWorker(t *testing.T) {
	assignments := []WorkAssignment{
		{
			ID:                  "contractor-1",
			Status:              AssignmentStatusAssigned,
			AssignmentType:      AssignmentTypeContractor,
			AssignedWorkerCount: 6,
		},
		{
			ID:                  "individual-1",
			Status:              AssignmentStatusAssigned,
			AssignmentType:      AssignmentTypeContractor,
			AssignedWorkerCount: 1,
		},
	}
	got := IndividualRecordAssignments(assignments)
	if len(got) != 1 || got[0].ID != "individual-1" {
		t.Fatalf("1-person contractor-typed assignment should still appear as an individual, got %#v", got)
	}
}

func TestBuildWorkResponseRemainingWorkers(t *testing.T) {
	needed := 10
	resp := BuildWorkResponse(&WorkRecord{WorkersNeeded: &needed, WorkersSelected: 6})
	if resp.RemainingWorkers != 4 {
		t.Fatalf("remaining = %d", resp.RemainingWorkers)
	}
}
