package repository

import (
	"testing"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

func TestFlattenWorkerSelectionsOmitsUnselectedSector(t *testing.T) {
	t.Parallel()

	remaining := []model.WorkerCategorySelection{
		{
			WorkCategoryID: "agriculture",
			Segments: []model.WorkerSegmentSelection{
				{
					WorkSegmentID: "coffee",
					WorkTypes: []model.WorkerTypeSelection{
						{WorkTypeID: "harvesting", SkillIDs: []string{"selective_picking", "cherry_collection"}},
					},
				},
			},
		},
	}

	categories, skillIDs := flattenWorkerSelections(remaining)

	gotIDs := make([]string, 0, len(categories))
	var primaryCount int
	for _, category := range categories {
		gotIDs = append(gotIDs, category.CategoryID)
		if category.IsPrimary {
			primaryCount++
		}
	}

	wantIDs := []string{"agriculture", "coffee", "harvesting"}
	if len(gotIDs) != len(wantIDs) {
		t.Fatalf("category ids = %v, want %v", gotIDs, wantIDs)
	}
	for i, id := range wantIDs {
		if gotIDs[i] != id {
			t.Fatalf("category ids = %v, want %v", gotIDs, wantIDs)
		}
	}
	if containsID(gotIDs, "construction") || containsID(gotIDs, "masonry") {
		t.Fatalf("omitted sector leaked into persist rows: %v", gotIDs)
	}
	if primaryCount != 1 {
		t.Fatalf("primary work types = %d, want 1", primaryCount)
	}
	if len(skillIDs) != 2 || skillIDs[0] != "selective_picking" || skillIDs[1] != "cherry_collection" {
		t.Fatalf("skill ids = %v, want [selective_picking cherry_collection]", skillIDs)
	}
}

func TestFlattenWorkerSelectionsMarksFirstWorkTypePrimary(t *testing.T) {
	t.Parallel()

	categories, _ := flattenWorkerSelections([]model.WorkerCategorySelection{
		{
			WorkCategoryID: "agriculture",
			Segments: []model.WorkerSegmentSelection{
				{
					WorkSegmentID: "coffee",
					WorkTypes: []model.WorkerTypeSelection{
						{WorkTypeID: "harvesting", SkillIDs: []string{"skill-1"}},
						{WorkTypeID: "pruning", SkillIDs: []string{"skill-2"}},
					},
				},
			},
		},
	})

	var primaryIDs []string
	for _, category := range categories {
		if category.IsPrimary {
			primaryIDs = append(primaryIDs, category.CategoryID)
		}
	}
	if len(primaryIDs) != 1 || primaryIDs[0] != "harvesting" {
		t.Fatalf("primary ids = %v, want [harvesting]", primaryIDs)
	}
}

func containsID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
