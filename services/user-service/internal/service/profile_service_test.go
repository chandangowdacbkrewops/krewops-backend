package service

import (
	"errors"
	"testing"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

func TestNormalizeOnboardingSelectionsIndividual(t *testing.T) {
	t.Parallel()

	got, err := normalizeOnboardingSelections("individual", []model.WorkerCategorySelection{
		{
			WorkCategoryID: "cat-1",
			Segments: []model.WorkerSegmentSelection{
				{
					WorkSegmentID: "seg-1",
					WorkTypes: []model.WorkerTypeSelection{
						{WorkTypeID: "type-1", SkillIDs: []string{"skill-1"}},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected valid onboarding selections, got %v", err)
	}
	if len(got) != 1 || got[0].WorkCategoryID != "cat-1" || len(got[0].Segments) != 1 {
		t.Fatalf("create should keep category and segment, got %+v", got)
	}
	if got[0].Segments[0].WorkSegmentID != "seg-1" || len(got[0].Segments[0].WorkTypes) != 0 {
		t.Fatalf("create should drop work types and skills, got %+v", got)
	}
}

func TestNormalizeOnboardingSelectionsIndividualLimits(t *testing.T) {
	t.Parallel()

	_, err := normalizeOnboardingSelections("individual", []model.WorkerCategorySelection{
		{WorkCategoryID: "cat-1", Segments: []model.WorkerSegmentSelection{{WorkSegmentID: "seg-1"}}},
		{WorkCategoryID: "cat-2", Segments: []model.WorkerSegmentSelection{{WorkSegmentID: "seg-2"}}},
	})
	if !errors.Is(err, ErrInvalidWorkerSelections) {
		t.Fatalf("expected ErrInvalidWorkerSelections, got %v", err)
	}
	if err.Error() != "individual workers can select only one work category" {
		t.Fatalf("got %q", err.Error())
	}

	_, err = normalizeOnboardingSelections("individual", []model.WorkerCategorySelection{
		{
			WorkCategoryID: "cat-1",
			Segments: []model.WorkerSegmentSelection{
				{WorkSegmentID: "seg-1"},
				{WorkSegmentID: "seg-2"},
			},
		},
	})
	if err == nil || err.Error() != "individual workers can select only one work segment" {
		t.Fatalf("expected one segment limit, got %v", err)
	}
}

func TestNormalizeOnboardingSelectionsContractor(t *testing.T) {
	t.Parallel()

	got, err := normalizeOnboardingSelections("contractor", []model.WorkerCategorySelection{
		{WorkCategoryID: "cat-1", Segments: []model.WorkerSegmentSelection{{WorkSegmentID: "seg-1"}}},
		{WorkCategoryID: "cat-2", Segments: []model.WorkerSegmentSelection{{WorkSegmentID: "seg-2"}}},
	})
	if err != nil {
		t.Fatalf("expected valid contractor onboarding, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 categories, got %+v", got)
	}
}

func TestNormalizeOnboardingSelectionsRequired(t *testing.T) {
	t.Parallel()

	_, err := normalizeOnboardingSelections("contractor", nil)
	if err == nil || err.Error() != "selections are required" {
		t.Fatalf("expected selections required, got %v", err)
	}

	_, err = normalizeOnboardingSelections("contractor", []model.WorkerCategorySelection{
		{WorkCategoryID: "cat-1"},
	})
	if err == nil || err.Error() != "at least one work segment is required for each category" {
		t.Fatalf("expected segment required, got %v", err)
	}
}

func TestNormalizeProfileSelectionsIndividual(t *testing.T) {
	t.Parallel()

	got, _, _, skillIDs, err := normalizeProfileSelections("individual", []model.WorkerCategorySelection{
		{
			WorkCategoryID: "cat-1",
			Segments: []model.WorkerSegmentSelection{
				{
					WorkSegmentID: "seg-1",
					WorkTypes: []model.WorkerTypeSelection{
						{WorkTypeID: "type-1", SkillIDs: []string{"skill-1", "skill-2"}},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected valid individual profile selections, got %v", err)
	}
	if len(got) != 1 || len(got[0].Segments) != 1 || len(got[0].Segments[0].WorkTypes) != 1 || len(skillIDs) != 2 {
		t.Fatalf("unexpected normalized individual selections: %+v", got)
	}
}

func TestNormalizeProfileSelectionsIndividualLimits(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		selections []model.WorkerCategorySelection
		want       string
	}{
		{
			name: "multiple work types",
			selections: []model.WorkerCategorySelection{
				{
					WorkCategoryID: "cat-1",
					Segments: []model.WorkerSegmentSelection{
						{
							WorkSegmentID: "seg-1",
							WorkTypes: []model.WorkerTypeSelection{
								{WorkTypeID: "type-1", SkillIDs: []string{"skill-1"}},
								{WorkTypeID: "type-2", SkillIDs: []string{"skill-2"}},
							},
						},
					},
				},
			},
			want: "individual workers can select only one work type",
		},
		{
			name: "more than two skills",
			selections: []model.WorkerCategorySelection{
				{
					WorkCategoryID: "cat-1",
					Segments: []model.WorkerSegmentSelection{
						{
							WorkSegmentID: "seg-1",
							WorkTypes: []model.WorkerTypeSelection{
								{WorkTypeID: "type-1", SkillIDs: []string{"skill-1", "skill-2", "skill-3"}},
							},
						},
					},
				},
			},
			want: "individual workers can select up to 2 skills",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, _, _, err := normalizeProfileSelections("individual", tc.selections)
			if !errors.Is(err, ErrInvalidWorkerSelections) {
				t.Fatalf("expected ErrInvalidWorkerSelections, got %v", err)
			}
			if err.Error() != tc.want {
				t.Fatalf("got %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestNormalizeProfileSelectionsContractor(t *testing.T) {
	t.Parallel()

	got, segmentIDs, workTypeIDs, skillIDs, err := normalizeProfileSelections("contractor", []model.WorkerCategorySelection{
		{
			WorkCategoryID: "cat-1",
			Segments: []model.WorkerSegmentSelection{
				{
					WorkSegmentID: "seg-1",
					WorkTypes: []model.WorkerTypeSelection{
						{WorkTypeID: "type-1", SkillIDs: []string{"skill-1", "skill-2"}},
						{WorkTypeID: "type-2", SkillIDs: []string{"skill-3"}},
					},
				},
			},
		},
		{
			WorkCategoryID: "cat-2",
			Segments: []model.WorkerSegmentSelection{
				{
					WorkSegmentID: "seg-2",
					WorkTypes: []model.WorkerTypeSelection{
						{WorkTypeID: "type-3", SkillIDs: []string{"skill-4", "skill-5", "skill-6"}},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected valid contractor profile selections, got %v", err)
	}
	if len(got) != 2 || len(segmentIDs) != 2 || len(workTypeIDs) != 3 || len(skillIDs) != 6 {
		t.Fatalf("unexpected normalized contractor selections: %+v", got)
	}
}

func TestNormalizeProfileSelectionsRequiredFields(t *testing.T) {
	t.Parallel()

	_, _, _, _, err := normalizeProfileSelections("contractor", nil)
	if err == nil || err.Error() != "selections are required" {
		t.Fatalf("expected selections required, got %v", err)
	}

	_, _, _, _, err = normalizeProfileSelections("contractor", []model.WorkerCategorySelection{
		{WorkCategoryID: "cat-1"},
	})
	if err == nil || err.Error() != "at least one work segment is required for each category" {
		t.Fatalf("expected segment required, got %v", err)
	}

	_, _, _, _, err = normalizeProfileSelections("contractor", []model.WorkerCategorySelection{
		{WorkCategoryID: "cat-1", Segments: []model.WorkerSegmentSelection{{WorkSegmentID: "seg-1"}}},
	})
	if err == nil || err.Error() != "at least one work type is required for each segment" {
		t.Fatalf("expected work type required, got %v", err)
	}

	_, _, _, _, err = normalizeProfileSelections("contractor", []model.WorkerCategorySelection{
		{
			WorkCategoryID: "cat-1",
			Segments: []model.WorkerSegmentSelection{
				{WorkSegmentID: "seg-1", WorkTypes: []model.WorkerTypeSelection{{WorkTypeID: "type-1"}}},
			},
		},
	})
	if err == nil || err.Error() != "at least one skill is required for each work type" {
		t.Fatalf("expected skill required, got %v", err)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

func TestNormalizeExpectedRateRange(t *testing.T) {
	t.Parallel()

	req := model.CreateWorkerProfileRequest{
		ExpectedRateMin: floatPtr(400),
		ExpectedRateMax: floatPtr(800),
	}
	if err := normalizeExpectedRate(&req); err != nil {
		t.Fatalf("expected valid rate range, got %v", err)
	}
	if req.ExpectedRateMin == nil || *req.ExpectedRateMin != 400 {
		t.Fatalf("min = %v, want 400", req.ExpectedRateMin)
	}
	if req.ExpectedRateMax == nil || *req.ExpectedRateMax != 800 {
		t.Fatalf("max = %v, want 800", req.ExpectedRateMax)
	}
	if req.ExpectedRate == nil || *req.ExpectedRate != 400 {
		t.Fatalf("legacy expected_rate should copy min, got %v", req.ExpectedRate)
	}
}

func TestNormalizeExpectedRateLegacySingleValue(t *testing.T) {
	t.Parallel()

	req := model.CreateWorkerProfileRequest{ExpectedRate: floatPtr(500)}
	if err := normalizeExpectedRate(&req); err != nil {
		t.Fatalf("expected legacy rate to expand, got %v", err)
	}
	if req.ExpectedRateMin == nil || *req.ExpectedRateMin != 500 || req.ExpectedRateMax == nil || *req.ExpectedRateMax != 500 {
		t.Fatalf("legacy rate should fill min and max, got min=%v max=%v", req.ExpectedRateMin, req.ExpectedRateMax)
	}
}

func TestNormalizeExpectedRateRejectsInvalidRange(t *testing.T) {
	t.Parallel()

	err := normalizeExpectedRate(&model.CreateWorkerProfileRequest{
		ExpectedRateMin: floatPtr(800),
		ExpectedRateMax: floatPtr(400),
	})
	if !errors.Is(err, ErrInvalidWorkerSelections) {
		t.Fatalf("expected ErrInvalidWorkerSelections, got %v", err)
	}
	if err.Error() != "expected_rate_max must be greater than or equal to expected_rate_min" {
		t.Fatalf("got %q", err.Error())
	}

	err = normalizeExpectedRate(&model.CreateWorkerProfileRequest{
		ExpectedRateMin: floatPtr(400),
	})
	if err == nil || err.Error() != "expected_rate_min and expected_rate_max are required together" {
		t.Fatalf("expected both bounds required, got %v", err)
	}
}

func TestNormalizeRatePair(t *testing.T) {
	t.Parallel()

	basis := "Per_Time"
	unit := "Day"
	basisPtr := &basis
	unitPtr := &unit
	if err := normalizeRatePair(&basisPtr, &unitPtr); err != nil {
		t.Fatalf("expected valid pair, got %v", err)
	}
	if basisPtr == nil || *basisPtr != "per_time" || unitPtr == nil || *unitPtr != "day" {
		t.Fatalf("got basis=%v unit=%v", valueOrEmpty(basisPtr), valueOrEmpty(unitPtr))
	}

	negotiable := "negotiable"
	negPtr := &negotiable
	var empty *string
	if err := normalizeRatePair(&negPtr, &empty); err != nil {
		t.Fatalf("negotiable without unit should be valid, got %v", err)
	}

	day := "day"
	dayPtr := &day
	negPtr = &negotiable
	err := normalizeRatePair(&negPtr, &dayPtr)
	if err == nil || err.Error() != "rate_unit must be empty when rate_basis is negotiable" {
		t.Fatalf("expected negotiable unit error, got %v", err)
	}

	fixed := "fixed"
	fixedPtr := &fixed
	err = normalizeRatePair(&fixedPtr, &empty)
	if err == nil || err.Error() != "rate_unit is required" {
		t.Fatalf("expected unit required, got %v", err)
	}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func TestNormalizeGender(t *testing.T) {
	t.Parallel()

	got, err := normalizeGender(nil)
	if err != nil || got != nil {
		t.Fatalf("nil gender should stay unset, got %v err %v", got, err)
	}

	empty := "  "
	got, err = normalizeGender(&empty)
	if err != nil || got != nil {
		t.Fatalf("blank gender should stay unset, got %v err %v", got, err)
	}

	male := "Male"
	got, err = normalizeGender(&male)
	if err != nil || got == nil || *got != "male" {
		t.Fatalf("got %v err %v", valueOrEmpty(got), err)
	}

	invalid := "unknown"
	_, err = normalizeGender(&invalid)
	if !errors.Is(err, ErrInvalidGender) {
		t.Fatalf("expected ErrInvalidGender, got %v", err)
	}
}
