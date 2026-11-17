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

const workInvitationsCollection = "work_invitations"

var ErrDuplicatePendingInvitation = errors.New("pending invitation already exists")

type offeredPaymentDocument struct {
	Amount    float64 `bson:"amount"`
	Currency  string  `bson:"currency"`
	RateBasis string  `bson:"rate_basis"`
	RateUnit  *string `bson:"rate_unit,omitempty"`
	Notes     *string `bson:"notes,omitempty"`
}

type invitationDocument struct {
	ID             string                 `bson:"_id"`
	WorkID         string                 `bson:"work_id"`
	OwnerID        string                 `bson:"owner_id"`
	WorkerID       string                 `bson:"worker_id"`
	Status         string                 `bson:"status"`
	ApplicantType  string                 `bson:"applicant_type"`
	SeatCount      int                    `bson:"seat_count"`
	OfferedPayment offeredPaymentDocument `bson:"offered_payment"`
	CreatedAt      time.Time              `bson:"created_at"`
	UpdatedAt      time.Time              `bson:"updated_at"`
	ExpiresAt      time.Time              `bson:"expires_at"`
	RespondedAt    *time.Time             `bson:"responded_at,omitempty"`
	AssignmentID   string                 `bson:"assignment_id,omitempty"`
}

type InvitationRepository struct {
	collection *mongo.Collection
}

func NewInvitationRepository(db *mongo.Database) *InvitationRepository {
	return &InvitationRepository{collection: db.Collection(workInvitationsCollection)}
}

func (r *InvitationRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "work_id", Value: 1}, {Key: "worker_id", Value: 1}},
			Options: options.Index().
				SetName("uniq_pending_invite_per_worker").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"status": model.InvitationStatusPending}),
		},
		{Keys: bson.D{{Key: "work_id", Value: 1}, {Key: "status", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "worker_id", Value: 1}, {Key: "status", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "expires_at", Value: 1}}},
	})
	return err
}

func (r *InvitationRepository) Insert(
	ctx context.Context,
	record *model.WorkInvitation,
) (*model.WorkInvitation, error) {
	now := time.Now().UTC()
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	if record.ExpiresAt.IsZero() {
		record.ExpiresAt = now.Add(model.InvitationTTL)
	}
	if record.Status == "" {
		record.Status = model.InvitationStatusPending
	}
	if record.SeatCount < 1 {
		record.SeatCount = 1
	}

	if _, err := r.collection.InsertOne(ctx, toInvitationDocument(record)); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDuplicatePendingInvitation
		}
		return nil, err
	}
	return record, nil
}

func (r *InvitationRepository) FindByID(ctx context.Context, id string) (*model.WorkInvitation, error) {
	var doc invitationDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *InvitationRepository) FindPending(
	ctx context.Context,
	workID string,
	workerID string,
) (*model.WorkInvitation, error) {
	var doc invitationDocument
	err := r.collection.FindOne(ctx, bson.M{
		"work_id":   workID,
		"worker_id": workerID,
		"status":    model.InvitationStatusPending,
	}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *InvitationRepository) ListByWorkID(ctx context.Context, workID string) ([]model.WorkInvitation, error) {
	return r.list(ctx, bson.M{"work_id": workID})
}

func (r *InvitationRepository) ListByWorkerID(
	ctx context.Context,
	workerID string,
	status string,
) ([]model.WorkInvitation, error) {
	filter := bson.M{"worker_id": workerID}
	if status != "" {
		filter["status"] = model.NormalizeInvitationStatus(status)
	}
	return r.list(ctx, filter)
}

func (r *InvitationRepository) HasPendingContractor(
	ctx context.Context,
	workID string,
	exceptWorkerID string,
) (bool, error) {
	filter := bson.M{
		"work_id":        workID,
		"status":         model.InvitationStatusPending,
		"applicant_type": model.ApplicantTypeContractor,
	}
	if exceptWorkerID != "" {
		filter["worker_id"] = bson.M{"$ne": exceptWorkerID}
	}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *InvitationRepository) UpdateStatus(
	ctx context.Context,
	id string,
	from []string,
	to string,
	extra bson.M,
) (*model.WorkInvitation, error) {
	now := time.Now().UTC()
	set := bson.M{
		"status":     to,
		"updated_at": now,
	}
	for key, value := range extra {
		set[key] = value
	}
	var doc invitationDocument
	err := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": id, "status": bson.M{"$in": from}},
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

func (r *InvitationRepository) CancelPendingByWorkID(ctx context.Context, workID string) error {
	now := time.Now().UTC()
	_, err := r.collection.UpdateMany(
		ctx,
		bson.M{"work_id": workID, "status": model.InvitationStatusPending},
		bson.M{"$set": bson.M{
			"status":       model.InvitationStatusCancelled,
			"updated_at":   now,
			"responded_at": now,
		}},
	)
	return err
}

func (r *InvitationRepository) list(ctx context.Context, filter bson.M) ([]model.WorkInvitation, error) {
	cursor, err := r.collection.Find(
		ctx,
		filter,
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make([]model.WorkInvitation, 0)
	for cursor.Next(ctx) {
		var doc invitationDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		records = append(records, *doc.toRecord())
	}
	return records, cursor.Err()
}

func toInvitationDocument(record *model.WorkInvitation) invitationDocument {
	return invitationDocument{
		ID:            record.ID,
		WorkID:        record.WorkID,
		OwnerID:       record.OwnerID,
		WorkerID:      record.WorkerID,
		Status:        model.NormalizeInvitationStatus(record.Status),
		ApplicantType: model.NormalizeApplicantType(record.ApplicantType),
		SeatCount:     record.SeatCount,
		OfferedPayment: offeredPaymentDocument{
			Amount:    record.OfferedPayment.Amount,
			Currency:  record.OfferedPayment.Currency,
			RateBasis: record.OfferedPayment.RateBasis,
			RateUnit:  record.OfferedPayment.RateUnit,
			Notes:     record.OfferedPayment.Notes,
		},
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
		ExpiresAt:    record.ExpiresAt,
		RespondedAt:  record.RespondedAt,
		AssignmentID: record.AssignmentID,
	}
}

func (d invitationDocument) toRecord() *model.WorkInvitation {
	return &model.WorkInvitation{
		ID:            d.ID,
		WorkID:        d.WorkID,
		OwnerID:       d.OwnerID,
		WorkerID:      d.WorkerID,
		Status:        model.NormalizeInvitationStatus(d.Status),
		ApplicantType: model.NormalizeApplicantType(d.ApplicantType),
		SeatCount:     d.SeatCount,
		OfferedPayment: model.OfferedPayment{
			Amount:    d.OfferedPayment.Amount,
			Currency:  d.OfferedPayment.Currency,
			RateBasis: d.OfferedPayment.RateBasis,
			RateUnit:  d.OfferedPayment.RateUnit,
			Notes:     d.OfferedPayment.Notes,
		},
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
		ExpiresAt:    d.ExpiresAt,
		RespondedAt:  d.RespondedAt,
		AssignmentID: d.AssignmentID,
	}
}
