package repository

import (
	"context"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/model"
)

const workPostingsCollection = "work_postings"

type workSearchDocument struct {
	ID     string `bson:"_id"`
	UserID string `bson:"user_id"`
	Status string `bson:"status"`

	Title            *string `bson:"title,omitempty"`
	WorkTypeID       *string `bson:"work_type_id,omitempty"`
	WorkTypeName     *string `bson:"work_type_name,omitempty"`
	WorkCategoryID   *string `bson:"work_category_id,omitempty"`
	WorkCategoryName *string `bson:"work_category_name,omitempty"`
	Description      *string `bson:"description,omitempty"`

	Address   *string  `bson:"address,omitempty"`
	City      *string  `bson:"city,omitempty"`
	State     *string  `bson:"state,omitempty"`
	Latitude  *float64 `bson:"latitude,omitempty"`
	Longitude *float64 `bson:"longitude,omitempty"`

	WorkersNeeded   *int     `bson:"workers_needed,omitempty"`
	ExperienceLevel *string  `bson:"experience_level,omitempty"`
	Skills          []string `bson:"skills"`

	StartDate     *time.Time `bson:"start_date,omitempty"`
	DurationValue *int       `bson:"duration_value,omitempty"`
	DurationUnit  *string    `bson:"duration_unit,omitempty"`
	ShiftTiming   *string    `bson:"shift_timing,omitempty"`

	PaymentType *string  `bson:"payment_type,omitempty"`
	BudgetRate  *float64 `bson:"budget_rate,omitempty"`

	PublishedAt *time.Time `bson:"published_at,omitempty"`
	CreatedAt   time.Time  `bson:"created_at"`
}

// WorkSearchRepository issues read-only search queries against the
// work_postings Mongo collection owned by work-service.
type WorkSearchRepository struct {
	collection *mongo.Collection
}

func NewWorkSearchRepository(db *mongo.Database) *WorkSearchRepository {
	return &WorkSearchRepository{
		collection: db.Collection(workPostingsCollection),
	}
}

func (r *WorkSearchRepository) SearchWork(
	ctx context.Context,
	filters model.WorkSearchFilters,
) (*model.WorkSearchResponse, error) {
	filter := bson.M{"status": "published"}

	if v := trimmed(filters.WorkTypeID); v != "" {
		filter["work_type_id"] = v
	}
	if v := trimmed(filters.WorkCategoryID); v != "" {
		filter["work_category_id"] = v
	}
	if v := trimmed(filters.City); v != "" {
		filter["city"] = caseInsensitiveExact(v)
	}
	if v := trimmed(filters.State); v != "" {
		filter["state"] = caseInsensitiveExact(v)
	}
	if v := trimmed(filters.ExperienceLevel); v != "" {
		filter["experience_level"] = v
	}
	if v := trimmed(filters.PaymentType); v != "" {
		filter["payment_type"] = v
	}

	budget := bson.M{}
	if filters.MinBudgetRate != nil {
		budget["$gte"] = *filters.MinBudgetRate
	}
	if filters.MaxBudgetRate != nil {
		budget["$lte"] = *filters.MaxBudgetRate
	}
	if len(budget) > 0 {
		filter["budget_rate"] = budget
	}

	if v := trimmed(filters.Keyword); v != "" {
		pattern := regexp.QuoteMeta(v)
		filter["$or"] = []bson.M{
			{"title": bson.M{"$regex": pattern, "$options": "i"}},
			{"description": bson.M{"$regex": pattern, "$options": "i"}},
		}
	}

	page, pageSize := model.NormalizePagination(filters.Page, filters.PageSize)
	offset := int64((page - 1) * pageSize)

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "published_at", Value: -1}, {Key: "created_at", Value: -1}}).
		SetSkip(offset).
		SetLimit(int64(pageSize))

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	response := &model.WorkSearchResponse{
		Results:  []model.WorkSearchResult{},
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
	}

	for cursor.Next(ctx) {
		var doc workSearchDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}

		item := model.WorkSearchResult{
			ID:               doc.ID,
			UserID:           doc.UserID,
			Status:           doc.Status,
			Title:            doc.Title,
			WorkTypeID:       doc.WorkTypeID,
			WorkTypeName:     doc.WorkTypeName,
			WorkCategoryID:   doc.WorkCategoryID,
			WorkCategoryName: doc.WorkCategoryName,
			Description:      doc.Description,
			Address:          doc.Address,
			City:             doc.City,
			State:            doc.State,
			Latitude:         doc.Latitude,
			Longitude:        doc.Longitude,
			WorkersNeeded:    doc.WorkersNeeded,
			ExperienceLevel:  doc.ExperienceLevel,
			Skills:           doc.Skills,
			StartDate:        doc.StartDate,
			DurationValue:    doc.DurationValue,
			DurationUnit:     doc.DurationUnit,
			ShiftTiming:      doc.ShiftTiming,
			PaymentType:      doc.PaymentType,
			BudgetRate:       doc.BudgetRate,
			PublishedAt:      doc.PublishedAt,
			CreatedAt:        doc.CreatedAt,
		}
		if item.Skills == nil {
			item.Skills = []string{}
		}
		response.Results = append(response.Results, item)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return response, nil
}

func caseInsensitiveExact(value string) bson.M {
	return bson.M{
		"$regex":   "^" + regexp.QuoteMeta(value) + "$",
		"$options": "i",
	}
}

func trimmed(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
