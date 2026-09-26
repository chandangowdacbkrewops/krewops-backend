package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
)

const workPostingsCollection = "work_postings"

type workDocument struct {
	ID     string `bson:"_id"`
	UserID string `bson:"user_id"`

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

	WorkersNeeded     *int     `bson:"workers_needed,omitempty"`
	ExperienceLevel   *string  `bson:"experience_level,omitempty"`
	Skills            []string `bson:"skills"`
	ToolsProvided     *bool    `bson:"tools_provided,omitempty"`
	MaterialsProvided *bool    `bson:"materials_provided,omitempty"`

	StartDate     *time.Time `bson:"start_date,omitempty"`
	DurationValue *int       `bson:"duration_value,omitempty"`
	DurationUnit  *string    `bson:"duration_unit,omitempty"`
	ShiftTiming   *string    `bson:"shift_timing,omitempty"`

	PaymentType           *string  `bson:"payment_type,omitempty"`
	BudgetRate            *float64 `bson:"budget_rate,omitempty"`
	PaymentNotes          *string  `bson:"payment_notes,omitempty"`
	AccommodationProvided *bool    `bson:"accommodation_provided,omitempty"`
	MealsProvided         *bool    `bson:"meals_provided,omitempty"`

	Attributes map[string]string `bson:"attributes"`

	Status      string     `bson:"status"`
	PublishedAt *time.Time `bson:"published_at,omitempty"`
	CreatedAt   time.Time  `bson:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at"`
}

type WorkRepository struct {
	collection *mongo.Collection
}

func NewWorkRepository(db *mongo.Database) *WorkRepository {
	return &WorkRepository{
		collection: db.Collection(workPostingsCollection),
	}
}

func (r *WorkRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "work_type_id", Value: 1}}},
		{Keys: bson.D{{Key: "work_category_id", Value: 1}}},
		{Keys: bson.D{{Key: "city", Value: 1}}, Options: options.Index().SetCollation(&options.Collation{Locale: "en", Strength: 2})},
		{Keys: bson.D{{Key: "state", Value: 1}}},
		{Keys: bson.D{{Key: "published_at", Value: -1}}},
		{Keys: bson.D{{Key: "budget_rate", Value: 1}}},
	})
	return err
}

func (r *WorkRepository) InsertWork(
	ctx context.Context,
	record *model.WorkRecord,
) (*model.WorkRecord, error) {
	now := time.Now().UTC()
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now

	if record.Skills == nil {
		record.Skills = []string{}
	}
	if record.Attributes == nil {
		record.Attributes = map[string]string{}
	}

	doc := workDocument{
		ID:                    record.ID,
		UserID:                record.UserID,
		Title:                 record.Title,
		WorkTypeID:            record.WorkTypeID,
		WorkTypeName:          record.WorkTypeName,
		WorkCategoryID:        record.WorkCategoryID,
		WorkCategoryName:      record.WorkCategoryName,
		Description:           record.Description,
		Address:               record.Address,
		City:                  record.City,
		State:                 record.State,
		Latitude:              record.Latitude,
		Longitude:             record.Longitude,
		WorkersNeeded:         record.WorkersNeeded,
		ExperienceLevel:       record.ExperienceLevel,
		Skills:                record.Skills,
		ToolsProvided:         record.ToolsProvided,
		MaterialsProvided:     record.MaterialsProvided,
		StartDate:             record.StartDate,
		DurationValue:         record.DurationValue,
		DurationUnit:          record.DurationUnit,
		ShiftTiming:           record.ShiftTiming,
		PaymentType:           record.PaymentType,
		BudgetRate:            record.BudgetRate,
		PaymentNotes:          record.PaymentNotes,
		AccommodationProvided: record.AccommodationProvided,
		MealsProvided:         record.MealsProvided,
		Attributes:            record.Attributes,
		Status:                record.Status,
		PublishedAt:           record.PublishedAt,
		CreatedAt:             record.CreatedAt,
		UpdatedAt:             record.UpdatedAt,
	}

	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		return nil, err
	}

	return record, nil
}

func (r *WorkRepository) FindByID(
	ctx context.Context,
	id string,
) (*model.WorkRecord, error) {
	var doc workDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return doc.toRecord(), nil
}

func (r *WorkRepository) ListByUserID(
	ctx context.Context,
	userID string,
	status *string,
) ([]model.WorkRecord, error) {
	filter := bson.M{"user_id": userID}
	if status != nil && *status != "" {
		filter["status"] = *status
	}

	cursor, err := r.collection.Find(
		ctx,
		filter,
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make([]model.WorkRecord, 0)
	for cursor.Next(ctx) {
		var doc workDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		records = append(records, *doc.toRecord())
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (r *WorkRepository) FindByIDs(
	ctx context.Context,
	ids []string,
) (map[string]model.WorkRecord, error) {
	results := make(map[string]model.WorkRecord, len(ids))
	if len(ids) == 0 {
		return results, nil
	}

	cursor, err := r.collection.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc workDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		results[doc.ID] = *doc.toRecord()
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (d workDocument) toRecord() *model.WorkRecord {
	skills := d.Skills
	if skills == nil {
		skills = []string{}
	}
	attributes := d.Attributes
	if attributes == nil {
		attributes = map[string]string{}
	}

	return &model.WorkRecord{
		ID:                    d.ID,
		UserID:                d.UserID,
		Title:                 d.Title,
		WorkTypeID:            d.WorkTypeID,
		WorkTypeName:          d.WorkTypeName,
		WorkCategoryID:        d.WorkCategoryID,
		WorkCategoryName:      d.WorkCategoryName,
		Description:           d.Description,
		Attributes:            attributes,
		Address:               d.Address,
		City:                  d.City,
		State:                 d.State,
		Latitude:              d.Latitude,
		Longitude:             d.Longitude,
		WorkersNeeded:         d.WorkersNeeded,
		ExperienceLevel:       d.ExperienceLevel,
		Skills:                skills,
		ToolsProvided:         d.ToolsProvided,
		MaterialsProvided:     d.MaterialsProvided,
		StartDate:             d.StartDate,
		DurationValue:         d.DurationValue,
		DurationUnit:          d.DurationUnit,
		ShiftTiming:           d.ShiftTiming,
		PaymentType:           d.PaymentType,
		BudgetRate:            d.BudgetRate,
		PaymentNotes:          d.PaymentNotes,
		AccommodationProvided: d.AccommodationProvided,
		MealsProvided:         d.MealsProvided,
		Status:                d.Status,
		PublishedAt:           d.PublishedAt,
		CreatedAt:             d.CreatedAt,
		UpdatedAt:             d.UpdatedAt,
	}
}
