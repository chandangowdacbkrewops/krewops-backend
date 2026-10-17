package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
)

const contractorWorkRecordsCollection = "contractor_work_records"

type contractorWorkRecordDocument struct {
	ID                     string    `bson:"_id"`
	WorkID                 string    `bson:"work_id"`
	ContractorAssignmentID string    `bson:"contractor_assignment_id"`
	WorkDate               string    `bson:"work_date"`
	DeployedWorkerCount    int       `bson:"deployed_worker_count"`
	RecordedBy             string    `bson:"recorded_by"`
	CreatedAt              time.Time `bson:"created_at"`
	UpdatedAt              time.Time `bson:"updated_at"`
}

type ContractorRecordRepository struct {
	collection *mongo.Collection
}

func NewContractorRecordRepository(db *mongo.Database) *ContractorRecordRepository {
	return &ContractorRecordRepository{collection: db.Collection(contractorWorkRecordsCollection)}
}

func (r *ContractorRecordRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "contractor_assignment_id", Value: 1},
				{Key: "work_date", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "work_id", Value: 1}, {Key: "work_date", Value: 1}}},
	})
	return err
}

func (r *ContractorRecordRepository) Upsert(
	ctx context.Context,
	record *model.ContractorWorkRecord,
) (*model.ContractorWorkRecord, error) {
	now := time.Now().UTC()
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now

	filter := bson.M{
		"contractor_assignment_id": record.ContractorAssignmentID,
		"work_date":                record.WorkDate,
	}
	update := bson.M{
		"$set": bson.M{
			"work_id":               record.WorkID,
			"deployed_worker_count": record.DeployedWorkerCount,
			"recorded_by":           record.RecordedBy,
			"updated_at":            now,
		},
		"$setOnInsert": bson.M{
			"_id":        record.ID,
			"created_at": record.CreatedAt,
		},
	}

	var doc contractorWorkRecordDocument
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

func (r *ContractorRecordRepository) ListByWorkID(ctx context.Context, workID string) ([]model.ContractorWorkRecord, error) {
	return r.find(ctx, bson.M{"work_id": workID})
}

func (r *ContractorRecordRepository) ListByAssignmentID(ctx context.Context, assignmentID string) ([]model.ContractorWorkRecord, error) {
	return r.find(ctx, bson.M{"contractor_assignment_id": assignmentID})
}

func (r *ContractorRecordRepository) ListByWorkIDAndDate(
	ctx context.Context,
	workID string,
	workDate string,
) ([]model.ContractorWorkRecord, error) {
	return r.find(ctx, bson.M{"work_id": workID, "work_date": workDate})
}

func (r *ContractorRecordRepository) find(ctx context.Context, filter bson.M) ([]model.ContractorWorkRecord, error) {
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "work_date", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make([]model.ContractorWorkRecord, 0)
	for cursor.Next(ctx) {
		var doc contractorWorkRecordDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		records = append(records, *doc.toRecord())
	}
	return records, cursor.Err()
}

func (d contractorWorkRecordDocument) toRecord() *model.ContractorWorkRecord {
	return &model.ContractorWorkRecord{
		ID:                     d.ID,
		WorkID:                 d.WorkID,
		ContractorAssignmentID: d.ContractorAssignmentID,
		WorkDate:               d.WorkDate,
		DeployedWorkerCount:    d.DeployedWorkerCount,
		RecordedBy:             d.RecordedBy,
		CreatedAt:              d.CreatedAt,
		UpdatedAt:              d.UpdatedAt,
	}
}
