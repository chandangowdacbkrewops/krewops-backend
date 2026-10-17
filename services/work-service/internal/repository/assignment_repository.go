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

const workAssignmentsCollection = "work_assignments"

type agreedPaymentDocument struct {
	RateType         string `bson:"rate_type"`
	RateUnit         string `bson:"rate_unit"`
	RateMinor        int64  `bson:"rate_minor"`
	Currency         string `bson:"currency"`
	FixedAmountMinor *int64 `bson:"fixed_amount_minor,omitempty"`
}

type assignmentDocument struct {
	ID                   string                `bson:"_id"`
	WorkID               string                `bson:"work_id"`
	ApplicationID        string                `bson:"application_id"`
	AssignmentType       string                `bson:"assignment_type"`
	WorkerID             string                `bson:"worker_id"`
	WorkerProfileID      string                `bson:"worker_profile_id,omitempty"`
	ContractorProfileID  string                `bson:"contractor_profile_id,omitempty"`
	Status               string                `bson:"status"`
	Quotation            quotationDocument     `bson:"agreed_quotation"`
	AgreedPayment        agreedPaymentDocument `bson:"agreed_payment"`
	JoinedAt             string                `bson:"joined_at,omitempty"`
	CommittedWorkerCount int                   `bson:"committed_worker_count,omitempty"`
	CompletedQuantity    float64               `bson:"completed_quantity,omitempty"`
	AssignedAt           time.Time             `bson:"assigned_at"`
	StartedAt            *time.Time            `bson:"started_at,omitempty"`
	CompletedAt          *time.Time            `bson:"completed_at,omitempty"`
	PaymentStatus        string                `bson:"payment_status"`
	FullDays             int                   `bson:"full_days"`
	HalfDays             int                   `bson:"half_days"`
	DaysWorked           float64               `bson:"days_worked"`
	PayableAmount        float64               `bson:"payable_amount"`
	PayableCurrency      string                `bson:"payable_currency,omitempty"`
	LogsFrozen           bool                  `bson:"logs_frozen"`
	CreatedAt            time.Time             `bson:"created_at"`
	UpdatedAt            time.Time             `bson:"updated_at"`
}

type AssignmentRepository struct {
	collection *mongo.Collection
}

func NewAssignmentRepository(db *mongo.Database) *AssignmentRepository {
	return &AssignmentRepository{collection: db.Collection(workAssignmentsCollection)}
}

func (r *AssignmentRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "work_id", Value: 1}, {Key: "worker_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "work_id", Value: 1}, {Key: "assignment_type", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{
					"assignment_type": model.AssignmentTypeContractor,
					"status": bson.M{"$in": []string{
						model.AssignmentStatusAssigned,
						model.AssignmentStatusInProgress,
						model.AssignmentStatusSubmitted,
						model.AssignmentStatusApproved,
						"STARTED",
						"COMPLETED",
					}},
				}).
				SetName("uniq_active_contractor_per_work"),
		},
		{Keys: bson.D{{Key: "work_id", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "worker_id", Value: 1}}},
		{Keys: bson.D{{Key: "application_id", Value: 1}}},
	})
	return err
}

func (r *AssignmentRepository) Insert(
	ctx context.Context,
	record *model.WorkAssignment,
) (*model.WorkAssignment, error) {
	now := time.Now().UTC()
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.AssignedAt.IsZero() {
		record.AssignedAt = now
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	if record.PaymentStatus == "" {
		record.PaymentStatus = model.PaymentStatusPending
	}

	if _, err := r.collection.InsertOne(ctx, toAssignmentDocument(record)); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDuplicateAssignment
		}
		return nil, err
	}
	return record, nil
}

func (r *AssignmentRepository) FindByID(ctx context.Context, id string) (*model.WorkAssignment, error) {
	var doc assignmentDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *AssignmentRepository) FindByWorkIDAndWorkerID(
	ctx context.Context,
	workID string,
	workerID string,
) (*model.WorkAssignment, error) {
	var doc assignmentDocument
	err := r.collection.FindOne(ctx, bson.M{
		"work_id":   workID,
		"worker_id": workerID,
		"status":    bson.M{"$ne": model.AssignmentStatusCancelled},
	}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *AssignmentRepository) ListByWorkID(ctx context.Context, workID string) ([]model.WorkAssignment, error) {
	return r.find(ctx, bson.M{"work_id": workID}, bson.D{{Key: "assigned_at", Value: 1}})
}

func (r *AssignmentRepository) ListActiveByWorkID(ctx context.Context, workID string) ([]model.WorkAssignment, error) {
	return r.find(ctx, bson.M{
		"work_id": workID,
		"status":  bson.M{"$ne": model.AssignmentStatusCancelled},
	}, bson.D{{Key: "assigned_at", Value: 1}})
}

func (r *AssignmentRepository) FindActiveContractorByWorkID(
	ctx context.Context,
	workID string,
) (*model.WorkAssignment, error) {
	var doc assignmentDocument
	err := r.collection.FindOne(ctx, bson.M{
		"work_id":         workID,
		"assignment_type": model.AssignmentTypeContractor,
		"status":          bson.M{"$ne": model.AssignmentStatusCancelled},
	}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *AssignmentRepository) CountActiveByWorkIDAndType(
	ctx context.Context,
	workID string,
	assignmentType string,
) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{
		"work_id":         workID,
		"assignment_type": assignmentType,
		"status":          bson.M{"$ne": model.AssignmentStatusCancelled},
	})
}

func (r *AssignmentRepository) ListByWorkerID(ctx context.Context, workerID string) ([]model.WorkAssignment, error) {
	return r.find(ctx, bson.M{"worker_id": workerID}, bson.D{{Key: "assigned_at", Value: -1}})
}

func (r *AssignmentRepository) UpdateStatus(
	ctx context.Context,
	id string,
	from []string,
	to string,
	extra bson.M,
) (*model.WorkAssignment, error) {
	now := time.Now().UTC()
	set := bson.M{"status": to, "updated_at": now}
	for key, value := range extra {
		set[key] = value
	}
	var doc assignmentDocument
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

func (r *AssignmentRepository) UpdateLiveSettlement(
	ctx context.Context,
	id string,
	fullDays int,
	halfDays int,
	daysWorked float64,
	payableAmount float64,
	currency string,
	paymentStatus string,
) error {
	set := bson.M{
		"full_days":      fullDays,
		"half_days":      halfDays,
		"days_worked":    daysWorked,
		"payable_amount": payableAmount,
		"updated_at":     time.Now().UTC(),
	}
	if currency != "" {
		set["payable_currency"] = currency
	}
	if paymentStatus != "" {
		set["payment_status"] = paymentStatus
	}
	_, err := r.collection.UpdateByID(ctx, id, bson.M{"$set": set})
	return err
}

func (r *AssignmentRepository) SaveSettlement(
	ctx context.Context,
	id string,
	fullDays int,
	halfDays int,
	daysWorked float64,
	payableAmount float64,
	currency string,
) (*model.WorkAssignment, error) {
	now := time.Now().UTC()
	var doc assignmentDocument
	err := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{
			"full_days":        fullDays,
			"half_days":        halfDays,
			"days_worked":      daysWorked,
			"payable_amount":   payableAmount,
			"payable_currency": currency,
			"payment_status":   model.PaymentStatusPending,
			"logs_frozen":      true,
			"updated_at":       now,
		}},
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

func (r *AssignmentRepository) UpdatePaymentStatus(
	ctx context.Context,
	id string,
	status string,
) error {
	_, err := r.collection.UpdateByID(ctx, id, bson.M{"$set": bson.M{
		"payment_status": status,
		"updated_at":     time.Now().UTC(),
	}})
	return err
}

func (r *AssignmentRepository) find(
	ctx context.Context,
	filter bson.M,
	sort bson.D,
) ([]model.WorkAssignment, error) {
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(sort))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make([]model.WorkAssignment, 0)
	for cursor.Next(ctx) {
		var doc assignmentDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		records = append(records, *doc.toRecord())
	}
	return records, cursor.Err()
}

func toAssignmentDocument(record *model.WorkAssignment) assignmentDocument {
	assignmentType := model.NormalizeAssignmentType(record.AssignmentType)
	return assignmentDocument{
		ID:                  record.ID,
		WorkID:              record.WorkID,
		ApplicationID:       record.ApplicationID,
		AssignmentType:      assignmentType,
		WorkerID:            record.WorkerID,
		WorkerProfileID:     record.WorkerProfileID,
		ContractorProfileID: record.ContractorProfileID,
		Status:              record.Status,
		Quotation: quotationDocument{
			Amount:                 record.Quotation.Amount,
			Currency:               record.Quotation.Currency,
			RateBasis:              record.Quotation.RateBasis,
			RateUnit:               record.Quotation.RateUnit,
			EstimatedDurationHours: record.Quotation.EstimatedDurationHours,
			Message:                record.Quotation.Message,
		},
		AgreedPayment: agreedPaymentDocument{
			RateType:         record.AgreedPayment.RateType,
			RateUnit:         record.AgreedPayment.RateUnit,
			RateMinor:        record.AgreedPayment.RateMinor,
			Currency:         record.AgreedPayment.Currency,
			FixedAmountMinor: record.AgreedPayment.FixedAmountMinor,
		},
		JoinedAt:             record.JoinedAt,
		CommittedWorkerCount: record.CommittedWorkerCount,
		CompletedQuantity:    record.CompletedQuantity,
		AssignedAt:           record.AssignedAt,
		StartedAt:            record.StartedAt,
		CompletedAt:          record.CompletedAt,
		PaymentStatus:        record.PaymentStatus,
		FullDays:             record.FullDays,
		HalfDays:             record.HalfDays,
		DaysWorked:           record.DaysWorked,
		PayableAmount:        record.PayableAmount,
		PayableCurrency:      record.PayableCurrency,
		LogsFrozen:           record.LogsFrozen,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
	}
}

func (d assignmentDocument) toRecord() *model.WorkAssignment {
	assignmentType := model.NormalizeAssignmentType(d.AssignmentType)
	quotation := model.Quotation{
		Amount:                 d.Quotation.Amount,
		Currency:               d.Quotation.Currency,
		RateBasis:              quotationRateBasis(d.Quotation),
		RateUnit:               quotationRateUnit(d.Quotation),
		EstimatedDurationHours: d.Quotation.EstimatedDurationHours,
		Message:                d.Quotation.Message,
	}
	payment := model.AgreedPayment{
		RateType:         d.AgreedPayment.RateType,
		RateUnit:         d.AgreedPayment.RateUnit,
		RateMinor:        d.AgreedPayment.RateMinor,
		Currency:         d.AgreedPayment.Currency,
		FixedAmountMinor: d.AgreedPayment.FixedAmountMinor,
	}
	if payment.RateMinor == 0 && quotation.Amount > 0 {
		payment = model.AgreedPaymentFromQuotation(quotation, assignmentType)
	}
	joinedAt := d.JoinedAt
	if joinedAt == "" && !d.AssignedAt.IsZero() {
		joinedAt = d.AssignedAt.Format("2006-01-02")
	}
	workerProfileID := d.WorkerProfileID
	if workerProfileID == "" && assignmentType == model.AssignmentTypeIndividual {
		workerProfileID = d.WorkerID
	}
	contractorProfileID := d.ContractorProfileID
	if contractorProfileID == "" && assignmentType == model.AssignmentTypeContractor {
		contractorProfileID = d.WorkerID
	}
	return &model.WorkAssignment{
		ID:                   d.ID,
		WorkID:               d.WorkID,
		ApplicationID:        d.ApplicationID,
		AssignmentType:       assignmentType,
		WorkerID:             d.WorkerID,
		WorkerProfileID:      workerProfileID,
		ContractorProfileID:  contractorProfileID,
		Status:               model.NormalizeAssignmentStatus(d.Status),
		Quotation:            quotation,
		AgreedPayment:        payment,
		JoinedAt:             joinedAt,
		CommittedWorkerCount: d.CommittedWorkerCount,
		CompletedQuantity:    d.CompletedQuantity,
		AssignedAt:           d.AssignedAt,
		StartedAt:            d.StartedAt,
		CompletedAt:          d.CompletedAt,
		PaymentStatus:        d.PaymentStatus,
		FullDays:             d.FullDays,
		HalfDays:             d.HalfDays,
		DaysWorked:           d.DaysWorked,
		PayableAmount:        d.PayableAmount,
		PayableCurrency:      d.PayableCurrency,
		LogsFrozen:           d.LogsFrozen,
		CreatedAt:            d.CreatedAt,
		UpdatedAt:            d.UpdatedAt,
	}
}
