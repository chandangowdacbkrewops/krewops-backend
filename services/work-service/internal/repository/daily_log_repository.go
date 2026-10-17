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

const workDayRecordsCollection = "work_day_records"

type dailyLogDocument struct {
	ID           string    `bson:"_id"`
	WorkID       string    `bson:"work_id"`
	AssignmentID string    `bson:"assignment_id"`
	WorkerID     string    `bson:"worker_profile_id"`
	WorkDate     string    `bson:"work_date"`
	DayType      string    `bson:"day_type"`
	DayPortion   string    `bson:"day_portion,omitempty"`
	PayableUnits float64   `bson:"payable_units"`
	RecordedBy   string    `bson:"recorded_by,omitempty"`
	Note         *string   `bson:"note,omitempty"`
	Status       string    `bson:"status"`
	SubmittedAt  time.Time `bson:"submitted_at"`
	CreatedAt    time.Time `bson:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at"`
}

type DailyLogRepository struct {
	collection *mongo.Collection
}

func NewDailyLogRepository(db *mongo.Database) *DailyLogRepository {
	return &DailyLogRepository{collection: db.Collection(workDayRecordsCollection)}
}

func (r *DailyLogRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "assignment_id", Value: 1}, {Key: "work_date", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "work_id", Value: 1}, {Key: "work_date", Value: 1}}},
		{Keys: bson.D{{Key: "worker_profile_id", Value: 1}, {Key: "work_date", Value: 1}}},
	})
	return err
}

func (r *DailyLogRepository) Upsert(ctx context.Context, record *model.DailyLog) (*model.DailyLog, error) {
	now := time.Now().UTC()
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.SubmittedAt.IsZero() {
		record.SubmittedAt = now
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	if record.Status == "" {
		record.Status = model.DailyLogSubmitted
	}
	dayType := model.NormalizeDayPortion(firstNonEmpty(record.DayType, record.DayPortion))
	record.DayType = dayType
	record.DayPortion = dayType
	record.PayableUnits = model.DayPortionUnits(dayType)

	filter := bson.M{
		"assignment_id": record.AssignmentID,
		"work_date":     record.WorkDate,
	}
	update := bson.M{
		"$set": bson.M{
			"work_id":           record.WorkID,
			"assignment_id":     record.AssignmentID,
			"work_date":         record.WorkDate,
			"worker_profile_id": record.WorkerID,
			"day_type":          dayType,
			"day_portion":       dayType,
			"payable_units":     record.PayableUnits,
			"recorded_by":       record.RecordedBy,
			"note":              record.Note,
			"status":            record.Status,
			"updated_at":        now,
		},
		"$setOnInsert": bson.M{
			"_id":          record.ID,
			"submitted_at": record.SubmittedAt,
			"created_at":   record.CreatedAt,
		},
	}

	var doc dailyLogDocument
	err := r.collection.FindOneAndUpdate(
		ctx,
		filter,
		update,
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *DailyLogRepository) ListByAssignmentID(ctx context.Context, assignmentID string) ([]model.DailyLog, error) {
	return r.find(ctx, bson.M{"assignment_id": assignmentID})
}

func (r *DailyLogRepository) ListByWorkID(ctx context.Context, workID string) ([]model.DailyLog, error) {
	return r.find(ctx, bson.M{"work_id": workID})
}

func (r *DailyLogRepository) ListByWorkIDAndDate(ctx context.Context, workID string, workDate string) ([]model.DailyLog, error) {
	return r.find(ctx, bson.M{"work_id": workID, "work_date": workDate})
}

func (r *DailyLogRepository) LatestWorkDate(ctx context.Context, workID string) (string, error) {
	var doc dailyLogDocument
	err := r.collection.FindOne(
		ctx,
		bson.M{"work_id": workID},
		options.FindOne().SetSort(bson.D{{Key: "work_date", Value: -1}, {Key: "updated_at", Value: -1}}),
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", nil
		}
		return "", err
	}
	return doc.WorkDate, nil
}

func (r *DailyLogRepository) find(ctx context.Context, filter bson.M) ([]model.DailyLog, error) {
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "work_date", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make([]model.DailyLog, 0)
	for cursor.Next(ctx) {
		var doc dailyLogDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		records = append(records, *doc.toRecord())
	}
	return records, cursor.Err()
}

func (d dailyLogDocument) toRecord() *model.DailyLog {
	dayType := model.NormalizeDayPortion(firstNonEmpty(d.DayType, d.DayPortion))
	createdAt := d.CreatedAt
	if createdAt.IsZero() {
		createdAt = d.SubmittedAt
	}
	return &model.DailyLog{
		ID:           d.ID,
		WorkID:       d.WorkID,
		AssignmentID: d.AssignmentID,
		WorkerID:     d.WorkerID,
		WorkDate:     d.WorkDate,
		DayPortion:   dayType,
		DayType:      dayType,
		PayableUnits: model.DayPortionUnits(dayType),
		RecordedBy:   d.RecordedBy,
		Note:         d.Note,
		Status:       d.Status,
		SubmittedAt:  d.SubmittedAt,
		CreatedAt:    createdAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
