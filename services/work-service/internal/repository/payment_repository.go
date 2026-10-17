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

const workPaymentsCollection = "work_payments"

type paymentDocument struct {
	ID             string     `bson:"_id"`
	WorkID         string     `bson:"work_id"`
	AssignmentID   string     `bson:"assignment_id"`
	WorkerID       string     `bson:"worker_id"`
	Method         string     `bson:"method"`
	Amount         float64    `bson:"amount"`
	Currency       string     `bson:"currency"`
	Reference      *string    `bson:"reference,omitempty"`
	Status         string     `bson:"status"`
	ReceiptStatus  string     `bson:"receipt_status,omitempty"`
	DisputeMessage *string    `bson:"dispute_message,omitempty"`
	RecordedAt     time.Time  `bson:"recorded_at"`
	ConfirmedAt    *time.Time `bson:"confirmed_at,omitempty"`
	DisputedAt     *time.Time `bson:"disputed_at,omitempty"`
}

type PaymentRepository struct {
	collection *mongo.Collection
}

func NewPaymentRepository(db *mongo.Database) *PaymentRepository {
	return &PaymentRepository{collection: db.Collection(workPaymentsCollection)}
}

func (r *PaymentRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "assignment_id", Value: 1}, {Key: "recorded_at", Value: -1}}},
		{Keys: bson.D{{Key: "work_id", Value: 1}}},
		{Keys: bson.D{{Key: "worker_id", Value: 1}}},
	})
	return err
}

func (r *PaymentRepository) Insert(ctx context.Context, record *model.WorkPayment) (*model.WorkPayment, error) {
	now := time.Now().UTC()
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.RecordedAt.IsZero() {
		record.RecordedAt = now
	}
	if record.Status == "" {
		record.Status = model.PaymentStatusPaid
	}
	if _, err := r.collection.InsertOne(ctx, toPaymentDocument(record)); err != nil {
		return nil, err
	}
	return record, nil
}

func (r *PaymentRepository) FindByID(ctx context.Context, id string) (*model.WorkPayment, error) {
	var doc paymentDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *PaymentRepository) LatestPaidByAssignmentID(ctx context.Context, assignmentID string) (*model.WorkPayment, error) {
	var doc paymentDocument
	err := r.collection.FindOne(
		ctx,
		bson.M{"assignment_id": assignmentID, "status": model.PaymentStatusPaid},
		options.FindOne().SetSort(bson.D{{Key: "recorded_at", Value: -1}}),
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *PaymentRepository) ListByAssignmentID(ctx context.Context, assignmentID string) ([]model.WorkPayment, error) {
	return r.find(ctx, bson.M{"assignment_id": assignmentID})
}

func (r *PaymentRepository) ListByWorkID(ctx context.Context, workID string) ([]model.WorkPayment, error) {
	return r.find(ctx, bson.M{"work_id": workID})
}

func (r *PaymentRepository) UpdateReceipt(
	ctx context.Context,
	id string,
	receiptStatus string,
	message *string,
) (*model.WorkPayment, error) {
	now := time.Now().UTC()
	set := bson.M{"receipt_status": receiptStatus}
	if receiptStatus == model.PaymentReceiptConfirmed {
		set["confirmed_at"] = now
	}
	if receiptStatus == model.PaymentReceiptDisputed {
		set["disputed_at"] = now
		set["dispute_message"] = message
	}

	var doc paymentDocument
	err := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
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

func (r *PaymentRepository) find(ctx context.Context, filter bson.M) ([]model.WorkPayment, error) {
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "recorded_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make([]model.WorkPayment, 0)
	for cursor.Next(ctx) {
		var doc paymentDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		records = append(records, *doc.toRecord())
	}
	return records, cursor.Err()
}

func toPaymentDocument(record *model.WorkPayment) paymentDocument {
	return paymentDocument{
		ID:             record.ID,
		WorkID:         record.WorkID,
		AssignmentID:   record.AssignmentID,
		WorkerID:       record.WorkerID,
		Method:         record.Method,
		Amount:         record.Amount,
		Currency:       record.Currency,
		Reference:      record.Reference,
		Status:         record.Status,
		ReceiptStatus:  record.ReceiptStatus,
		DisputeMessage: record.DisputeMessage,
		RecordedAt:     record.RecordedAt,
		ConfirmedAt:    record.ConfirmedAt,
		DisputedAt:     record.DisputedAt,
	}
}

func (d paymentDocument) toRecord() *model.WorkPayment {
	return &model.WorkPayment{
		ID:             d.ID,
		WorkID:         d.WorkID,
		AssignmentID:   d.AssignmentID,
		WorkerID:       d.WorkerID,
		Method:         d.Method,
		Amount:         d.Amount,
		Currency:       d.Currency,
		Reference:      d.Reference,
		Status:         d.Status,
		ReceiptStatus:  d.ReceiptStatus,
		DisputeMessage: d.DisputeMessage,
		RecordedAt:     d.RecordedAt,
		ConfirmedAt:    d.ConfirmedAt,
		DisputedAt:     d.DisputedAt,
	}
}
