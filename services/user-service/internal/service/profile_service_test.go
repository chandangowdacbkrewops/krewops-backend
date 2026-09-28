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
			WorkTypes: []model.WorkerTypeSelection{
				{WorkTypeID: "type-1", SkillIDs: []string{"skill-1"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected valid onboarding selections, got %v", err)
	}
	if len(got) != 1 || got[0].WorkCategoryID != "cat-1" || len(got[0].WorkTypes) != 0 {
		t.Fatalf("create should keep only the category, got %+v", got)
	}
}

func TestNormalizeOnboardingSelectionsIndividualLimits(t *testing.T) {
	t.Parallel()

	_, err := normalizeOnboardingSelections("individual", []model.WorkerCategorySelection{
		{WorkCategoryID: "cat-1"},
		{WorkCategoryID: "cat-2"},
	})
	if !errors.Is(err, ErrInvalidWorkerSelections) {
		t.Fatalf("expected ErrInvalidWorkerSelections, got %v", err)
	}
	if err.Error() != "individual workers can select only one work category" {
		t.Fatalf("got %q", err.Error())
	}
}

func TestNormalizeOnboardingSelectionsContractor(t *testing.T) {
	t.Parallel()

	got, err := normalizeOnboardingSelections("contractor", []model.WorkerCategorySelection{
		{WorkCategoryID: "cat-1"},
		{WorkCategoryID: "cat-2"},
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
}

func TestNormalizeProfileSelectionsIndividual(t *testing.T) {
	t.Parallel()

	got, _, skillIDs, err := normalizeProfileSelections("individual", []model.WorkerCategorySelection{
		{
			WorkCategoryID: "cat-1",
			WorkTypes: []model.WorkerTypeSelection{
				{WorkTypeID: "type-1", SkillIDs: []string{"skill-1", "skill-2"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected valid individual profile selections, got %v", err)
	}
	if len(got) != 1 || len(got[0].WorkTypes) != 1 || len(skillIDs) != 2 {
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
					WorkTypes: []model.WorkerTypeSelection{
						{WorkTypeID: "type-1", SkillIDs: []string{"skill-1"}},
						{WorkTypeID: "type-2", SkillIDs: []string{"skill-2"}},
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
					WorkTypes: []model.WorkerTypeSelection{
						{WorkTypeID: "type-1", SkillIDs: []string{"skill-1", "skill-2", "skill-3"}},
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
			_, _, _, err := normalizeProfileSelections("individual", tc.selections)
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

	got, workTypeIDs, skillIDs, err := normalizeProfileSelections("contractor", []model.WorkerCategorySelection{
		{
			WorkCategoryID: "cat-1",
			WorkTypes: []model.WorkerTypeSelection{
				{WorkTypeID: "type-1", SkillIDs: []string{"skill-1", "skill-2"}},
				{WorkTypeID: "type-2", SkillIDs: []string{"skill-3"}},
			},
		},
		{
			WorkCategoryID: "cat-2",
			WorkTypes: []model.WorkerTypeSelection{
				{WorkTypeID: "type-3", SkillIDs: []string{"skill-4", "skill-5", "skill-6"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected valid contractor profile selections, got %v", err)
	}
	if len(got) != 2 || len(workTypeIDs) != 3 || len(skillIDs) != 6 {
		t.Fatalf("unexpected normalized contractor selections: %+v", got)
	}
}

func TestNormalizeProfileSelectionsRequiredFields(t *testing.T) {
	t.Parallel()

	_, _, _, err := normalizeProfileSelections("contractor", nil)
	if err == nil || err.Error() != "selections are required" {
		t.Fatalf("expected selections required, got %v", err)
	}

	_, _, _, err = normalizeProfileSelections("contractor", []model.WorkerCategorySelection{
		{WorkCategoryID: "cat-1"},
	})
	if err == nil || err.Error() != "at least one work type is required for each category" {
		t.Fatalf("expected work type required, got %v", err)
	}

	_, _, _, err = normalizeProfileSelections("contractor", []model.WorkerCategorySelection{
		{WorkCategoryID: "cat-1", WorkTypes: []model.WorkerTypeSelection{{WorkTypeID: "type-1"}}},
	})
	if err == nil || err.Error() != "at least one skill is required for each work type" {
		t.Fatalf("expected skill required, got %v", err)
	}
}
