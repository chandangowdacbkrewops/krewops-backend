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

const workReviewsCollection = "work_reviews"

var ErrDuplicateReview = errors.New("review already submitted for this assignment")

type reviewDocument struct {
	ID           string `bson:"_id"`
	WorkID       string `bson:"work_id"`
	AssignmentID string `bson:"assignment_id"`

	ReviewerID   string `bson:"reviewer_id"`
	ReviewerRole string `bson:"reviewer_role"`
	RevieweeID   string `bson:"reviewee_id"`

	Rating  int32   `bson:"rating"`
	Comment *string `bson:"comment,omitempty"`

	CreatedAt time.Time `bson:"created_at"`
}

type ReviewRepository struct {
	collection *mongo.Collection
}

func NewReviewRepository(db *mongo.Database) *ReviewRepository {
	return &ReviewRepository{collection: db.Collection(workReviewsCollection)}
}

func (r *ReviewRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "assignment_id", Value: 1}, {Key: "reviewer_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "reviewee_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "work_id", Value: 1}}},
	})
	return err
}

func (r *ReviewRepository) Insert(
	ctx context.Context,
	record *model.WorkReview,
) (*model.WorkReview, error) {
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}

	if _, err := r.collection.InsertOne(ctx, toReviewDocument(record)); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDuplicateReview
		}
		return nil, err
	}
	return record, nil
}

func (r *ReviewRepository) FindByAssignmentAndReviewer(
	ctx context.Context,
	assignmentID string,
	reviewerID string,
) (*model.WorkReview, error) {
	var doc reviewDocument
	err := r.collection.FindOne(ctx, bson.M{
		"assignment_id": assignmentID,
		"reviewer_id":   reviewerID,
	}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.toRecord(), nil
}

func (r *ReviewRepository) ListByWorkID(ctx context.Context, workID string) ([]model.WorkReview, error) {
	return r.list(ctx, bson.M{"work_id": workID})
}

func (r *ReviewRepository) ListByRevieweeID(ctx context.Context, revieweeID string) ([]model.WorkReview, error) {
	return r.list(ctx, bson.M{"reviewee_id": revieweeID})
}

// AggregateRatingForUser computes the average rating and review count for
// revieweeID on read, avoiding a denormalized counter that could drift out
// of sync with concurrent writes.
func (r *ReviewRepository) AggregateRatingForUser(
	ctx context.Context,
	revieweeID string,
) (*model.UserRatingSummary, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"reviewee_id": revieweeID}}},
		{{Key: "$group", Value: bson.M{
			"_id":     nil,
			"average": bson.M{"$avg": "$rating"},
			"count":   bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	summary := &model.UserRatingSummary{UserID: revieweeID}
	if !cursor.Next(ctx) {
		return summary, cursor.Err()
	}

	var result struct {
		Average float64 `bson:"average"`
		Count   int32   `bson:"count"`
	}
	if err := cursor.Decode(&result); err != nil {
		return nil, err
	}
	summary.AverageRating = result.Average
	summary.ReviewCount = result.Count
	return summary, nil
}

func (r *ReviewRepository) list(ctx context.Context, filter bson.M) ([]model.WorkReview, error) {
	cursor, err := r.collection.Find(
		ctx,
		filter,
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make([]model.WorkReview, 0)
	for cursor.Next(ctx) {
		var doc reviewDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		records = append(records, *doc.toRecord())
	}
	return records, cursor.Err()
}

func toReviewDocument(record *model.WorkReview) reviewDocument {
	return reviewDocument{
		ID:           record.ID,
		WorkID:       record.WorkID,
		AssignmentID: record.AssignmentID,
		ReviewerID:   record.ReviewerID,
		ReviewerRole: record.ReviewerRole,
		RevieweeID:   record.RevieweeID,
		Rating:       record.Rating,
		Comment:      record.Comment,
		CreatedAt:    record.CreatedAt,
	}
}

func (d reviewDocument) toRecord() *model.WorkReview {
	return &model.WorkReview{
		ID:           d.ID,
		WorkID:       d.WorkID,
		AssignmentID: d.AssignmentID,
		ReviewerID:   d.ReviewerID,
		ReviewerRole: d.ReviewerRole,
		RevieweeID:   d.RevieweeID,
		Rating:       d.Rating,
		Comment:      d.Comment,
		CreatedAt:    d.CreatedAt,
	}
}
