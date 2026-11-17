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

const paymentObligationsCollection = "payment_obligations"
const paymentTransactionsCollection = "payment_transactions"

type obligationDocument struct {
	ID                 string    `bson:"_id"`
	WorkID             string    `bson:"work_id"`
	AssignmentID       string    `bson:"assignment_id"`
	PayeeType          string    `bson:"payee_type"`
	PayeeID            string    `bson:"payee_id"`
	Currency           string    `bson:"currency"`
	PayableAmountMinor int64     `bson:"payable_amount_minor"`
	PaidAmountMinor    int64     `bson:"paid_amount_minor"`
	Status             string    `bson:"status"`
	CreatedAt          time.Time `bson:"created_at"`
	UpdatedAt          time.Time `bson:"updated_at"`
}

type transactionDocument struct {
	ID                  string     `bson:"_id"`
	PaymentObligationID string     `bson:"payment_obligation_id"`
	WorkID              string     `bson:"work_id"`
	AssignmentID        string     `bson:"assignment_id"`
	AmountMinor         int64      `bson:"amount_minor"`
	PaymentMethod       string     `bson:"payment_method"`
	Reference           *string    `bson:"reference,omitempty"`
	RecordedBy          string     `bson:"recorded_by"`
	PaidAt              time.Time  `bson:"paid_at"`
	ReceiptStatus       string     `bson:"receipt_status,omitempty"`
	DisputeMessage      *string    `bson:"dispute_message,omitempty"`
	ConfirmedAt         *time.Time `bson:"confirmed_at,omitempty"`
	DisputedAt          *time.Time `bson:"disputed_at,omitempty"`
	CreatedAt           time.Time  `bson:"created_at"`
}

type ObligationRepository struct {
	obligations  *mongo.Collection
	transactions *mongo.Collection
}

func NewObligationRepository(db *mongo.Database) *ObligationRepository {
	return &ObligationRepository{
		obligations:  db.Collection(paymentObligationsCollection),
		transactions: db.Collection(paymentTransactionsCollection),
	}
}

func (r *ObligationRepository) EnsureIndexes(ctx context.Context) error {
	if _, err := r.obligations.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "assignment_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "work_id", Value: 1}}},
	}); err != nil {
		return err
	}
	_, err := r.transactions.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "payment_obligation_id", Value: 1}, {Key: "created_at", Value: 1}}},
		{Keys: bson.D{{Key: "work_id", Value: 1}}},
		{Keys: bson.D{{Key: "assignment_id", Value: 1}}},
	})
	return err
}

func (r *ObligationRepository) UpsertForAssignment(
	ctx context.Context,
	record *model.PaymentObligation,
) (*model.PaymentObligation, error) {
	now := time.Now().UTC()
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	if record.Status == "" {
		record.Status = model.ObligationStatus(record.PayableAmountMinor, record.PaidAmountMinor)
	}

	filter := bson.M{"assignment_id": record.AssignmentID}
	update := bson.M{
		"$set": bson.M{
			"work_id":              record.WorkID,
			"payee_type":           record.PayeeType,
			"payee_id":             record.PayeeID,
			"currency":             record.Currency,
			"payable_amount_minor": record.PayableAmountMinor,
			"status":               record.Status,
			"updated_at":           now,
		},
		"$setOnInsert": bson.M{
			"_id":               record.ID,
			"paid_amount_minor": int64(0),
			"created_at":        record.CreatedAt,
		},
	}

	var doc obligationDocument
	err := r.obligations.FindOneAndUpdate(
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

func (r *ObligationRepository) FindByID(ctx context.Context, id string) (*model.PaymentObligation, error) {
	var doc obligationDocument
	err := r.obligations.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *ObligationRepository) FindByAssignmentID(ctx context.Context, assignmentID string) (*model.PaymentObligation, error) {
	var doc obligationDocument
	err := r.obligations.FindOne(ctx, bson.M{"assignment_id": assignmentID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *ObligationRepository) ListByWorkID(ctx context.Context, workID string) ([]model.PaymentObligation, error) {
	cursor, err := r.obligations.Find(ctx, bson.M{"work_id": workID}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make([]model.PaymentObligation, 0)
	for cursor.Next(ctx) {
		var doc obligationDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		records = append(records, *doc.toRecord())
	}
	return records, cursor.Err()
}

func (r *ObligationRepository) AddPayment(
	ctx context.Context,
	obligationID string,
	amountMinor int64,
	status string,
) (*model.PaymentObligation, error) {
	now := time.Now().UTC()
	var doc obligationDocument
	err := r.obligations.FindOneAndUpdate(
		ctx,
		bson.M{"_id": obligationID},
		bson.M{
			"$inc": bson.M{"paid_amount_minor": amountMinor},
			"$set": bson.M{"status": status, "updated_at": now},
		},
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

func (r *ObligationRepository) InsertTransaction(
	ctx context.Context,
	record *model.PaymentTransaction,
) (*model.PaymentTransaction, error) {
	now := time.Now().UTC()
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	if record.PaidAt.IsZero() {
		record.PaidAt = now
	}
	if _, err := r.transactions.InsertOne(ctx, transactionDocument{
		ID:                  record.ID,
		PaymentObligationID: record.PaymentObligationID,
		WorkID:              record.WorkID,
		AssignmentID:        record.AssignmentID,
		AmountMinor:         record.AmountMinor,
		PaymentMethod:       record.PaymentMethod,
		Reference:           record.Reference,
		RecordedBy:          record.RecordedBy,
		PaidAt:              record.PaidAt,
		ReceiptStatus:       record.ReceiptStatus,
		DisputeMessage:      record.DisputeMessage,
		ConfirmedAt:         record.ConfirmedAt,
		DisputedAt:          record.DisputedAt,
		CreatedAt:           record.CreatedAt,
	}); err != nil {
		return nil, err
	}
	return record, nil
}

func (r *ObligationRepository) FindTransactionByID(
	ctx context.Context,
	id string,
) (*model.PaymentTransaction, error) {
	var doc transactionDocument
	err := r.transactions.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *ObligationRepository) LatestTransactionByAssignmentID(
	ctx context.Context,
	assignmentID string,
) (*model.PaymentTransaction, error) {
	var doc transactionDocument
	err := r.transactions.FindOne(
		ctx,
		bson.M{"assignment_id": assignmentID},
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *ObligationRepository) LatestTransactionByObligationID(
	ctx context.Context,
	obligationID string,
) (*model.PaymentTransaction, error) {
	var doc transactionDocument
	err := r.transactions.FindOne(
		ctx,
		bson.M{"payment_obligation_id": obligationID},
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *ObligationRepository) UpdateTransactionReceipt(
	ctx context.Context,
	id string,
	receiptStatus string,
	message *string,
) (*model.PaymentTransaction, error) {
	now := time.Now().UTC()
	set := bson.M{"receipt_status": receiptStatus}
	if receiptStatus == model.PaymentReceiptConfirmed {
		set["confirmed_at"] = now
	}
	if receiptStatus == model.PaymentReceiptDisputed {
		set["disputed_at"] = now
		set["dispute_message"] = message
	}

	var doc transactionDocument
	err := r.transactions.FindOneAndUpdate(
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

func (r *ObligationRepository) ListTransactionsByObligationID(
	ctx context.Context,
	obligationID string,
) ([]model.PaymentTransaction, error) {
	return r.findTransactions(ctx, bson.M{"payment_obligation_id": obligationID})
}

func (r *ObligationRepository) ListTransactionsByWorkID(
	ctx context.Context,
	workID string,
) ([]model.PaymentTransaction, error) {
	return r.findTransactions(ctx, bson.M{"work_id": workID})
}

func (r *ObligationRepository) FindRecentDuplicate(
	ctx context.Context,
	obligationID string,
	amountMinor int64,
	method string,
	paidAt time.Time,
) (*model.PaymentTransaction, error) {
	filter := bson.M{
		"payment_obligation_id": obligationID,
		"amount_minor":          amountMinor,
		"payment_method":        method,
	}
	if !paidAt.IsZero() {
		filter["paid_at"] = paidAt
	}
	var doc transactionDocument
	err := r.transactions.FindOne(
		ctx,
		filter,
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *ObligationRepository) findTransactions(ctx context.Context, filter bson.M) ([]model.PaymentTransaction, error) {
	cursor, err := r.transactions.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make([]model.PaymentTransaction, 0)
	for cursor.Next(ctx) {
		var doc transactionDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		records = append(records, *doc.toRecord())
	}
	return records, cursor.Err()
}

func (d obligationDocument) toRecord() *model.PaymentObligation {
	return &model.PaymentObligation{
		ID:                 d.ID,
		WorkID:             d.WorkID,
		AssignmentID:       d.AssignmentID,
		PayeeType:          d.PayeeType,
		PayeeID:            d.PayeeID,
		Currency:           d.Currency,
		PayableAmountMinor: d.PayableAmountMinor,
		PaidAmountMinor:    d.PaidAmountMinor,
		Status:             d.Status,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

func (d transactionDocument) toRecord() *model.PaymentTransaction {
	return &model.PaymentTransaction{
		ID:                  d.ID,
		PaymentObligationID: d.PaymentObligationID,
		WorkID:              d.WorkID,
		AssignmentID:        d.AssignmentID,
		AmountMinor:         d.AmountMinor,
		PaymentMethod:       d.PaymentMethod,
		Reference:           d.Reference,
		RecordedBy:          d.RecordedBy,
		PaidAt:              d.PaidAt,
		ReceiptStatus:       d.ReceiptStatus,
		DisputeMessage:      d.DisputeMessage,
		ConfirmedAt:         d.ConfirmedAt,
		DisputedAt:          d.DisputedAt,
		CreatedAt:           d.CreatedAt,
	}
}
