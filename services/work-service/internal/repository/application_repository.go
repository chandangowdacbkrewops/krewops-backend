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

const workApplicationsCollection = "work_applications"

type quotationDocument struct {
	Amount                 float64  `bson:"amount"`
	Currency               string   `bson:"currency"`
	PriceType              string   `bson:"price_type"`
	EstimatedDurationHours *float64 `bson:"estimated_duration_hours,omitempty"`
	Message                *string  `bson:"message,omitempty"`
}

type applicationDocument struct {
	ID        string             `bson:"_id"`
	WorkID    string             `bson:"work_id"`
	WorkerID  string             `bson:"worker_id"`
	Status    string             `bson:"status"`
	Quotation quotationDocument  `bson:"quotation"`
	AppliedAt time.Time          `bson:"applied_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

type ApplicationRepository struct {
	collection *mongo.Collection
}

func NewApplicationRepository(db *mongo.Database) *ApplicationRepository {
	return &ApplicationRepository{
		collection: db.Collection(workApplicationsCollection),
	}
}

func (r *ApplicationRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "work_id", Value: 1}, {Key: "worker_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "work_id", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "worker_id", Value: 1}}},
		{
			Keys: bson.D{{Key: "work_id", Value: 1}},
			Options: options.Index().
				SetName("uniq_accepted_per_work").
				SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "status", Value: model.ApplicationStatusAccepted}}),
		},
	})
	return err
}

func (r *ApplicationRepository) Insert(
	ctx context.Context,
	record *model.WorkApplication,
) (*model.WorkApplication, error) {
	now := time.Now().UTC()
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.AppliedAt.IsZero() {
		record.AppliedAt = now
	}
	record.UpdatedAt = now

	doc := toApplicationDocument(record)
	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDuplicateApplication
		}
		return nil, err
	}

	return record, nil
}

func (r *ApplicationRepository) FindByID(
	ctx context.Context,
	id string,
) (*model.WorkApplication, error) {
	var doc applicationDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return doc.toRecord(), nil
}

func (r *ApplicationRepository) ListByWorkID(
	ctx context.Context,
	workID string,
	status *string,
) ([]model.WorkApplication, error) {
	filter := bson.M{"work_id": workID}
	if status != nil && *status != "" {
		filter["status"] = *status
	}

	cursor, err := r.collection.Find(
		ctx,
		filter,
		options.Find().SetSort(bson.D{{Key: "applied_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make([]model.WorkApplication, 0)
	for cursor.Next(ctx) {
		var doc applicationDocument
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

func (r *ApplicationRepository) ListByWorkerID(
	ctx context.Context,
	workerID string,
	status *string,
) ([]model.WorkApplication, error) {
	filter := bson.M{"worker_id": workerID}
	if status != nil && *status != "" {
		filter["status"] = *status
	}

	cursor, err := r.collection.Find(
		ctx,
		filter,
		options.Find().SetSort(bson.D{{Key: "applied_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make([]model.WorkApplication, 0)
	for cursor.Next(ctx) {
		var doc applicationDocument
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

func (r *ApplicationRepository) UpdateStatus(
	ctx context.Context,
	workID string,
	applicationID string,
	from []string,
	to string,
) (*model.WorkApplication, error) {
	now := time.Now().UTC()
	filter := bson.M{
		"_id":     applicationID,
		"work_id": workID,
		"status":  bson.M{"$in": from},
	}

	var doc applicationDocument
	err := r.collection.FindOneAndUpdate(
		ctx,
		filter,
		bson.M{"$set": bson.M{"status": to, "updated_at": now}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return doc.toRecord(), nil
}

func (r *ApplicationRepository) AcceptAndRejectOthers(
	ctx context.Context,
	workID string,
	applicationID string,
) (*model.WorkApplication, error) {
	now := time.Now().UTC()

	var doc applicationDocument
	err := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{
			"_id":     applicationID,
			"work_id": workID,
			"status": bson.M{"$in": []string{
				model.ApplicationStatusApplied,
				model.ApplicationStatusShortlisted,
			}},
		},
		bson.M{"$set": bson.M{
			"status":     model.ApplicationStatusAccepted,
			"updated_at": now,
		}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if mongo.IsDuplicateKeyError(err) {
		return nil, ErrApplicationAlreadyAccepted
	}
	if err != nil {
		return nil, err
	}

	_, err = r.collection.UpdateMany(
		ctx,
		bson.M{
			"work_id": workID,
			"_id":     bson.M{"$ne": applicationID},
			"status": bson.M{"$in": []string{
				model.ApplicationStatusApplied,
				model.ApplicationStatusShortlisted,
			}},
		},
		bson.M{"$set": bson.M{
			"status":     model.ApplicationStatusRejected,
			"updated_at": now,
		}},
	)
	if err != nil {
		return nil, err
	}

	return doc.toRecord(), nil
}

func toApplicationDocument(record *model.WorkApplication) applicationDocument {
	return applicationDocument{
		ID:       record.ID,
		WorkID:   record.WorkID,
		WorkerID: record.WorkerID,
		Status:   record.Status,
		Quotation: quotationDocument{
			Amount:                 record.Quotation.Amount,
			Currency:               record.Quotation.Currency,
			PriceType:              record.Quotation.PriceType,
			EstimatedDurationHours: record.Quotation.EstimatedDurationHours,
			Message:                record.Quotation.Message,
		},
		AppliedAt: record.AppliedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func (d applicationDocument) toRecord() *model.WorkApplication {
	return &model.WorkApplication{
		ID:       d.ID,
		WorkID:   d.WorkID,
		WorkerID: d.WorkerID,
		Status:   d.Status,
		Quotation: model.Quotation{
			Amount:                 d.Quotation.Amount,
			Currency:               d.Quotation.Currency,
			PriceType:              d.Quotation.PriceType,
			EstimatedDurationHours: d.Quotation.EstimatedDurationHours,
			Message:                d.Quotation.Message,
		},
		AppliedAt: d.AppliedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
