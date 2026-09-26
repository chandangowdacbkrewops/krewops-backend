package service

import (
	"context"

	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/repository"
)

// SearchService is the single point of entry for the "Search" domain: work
// postings and worker profiles are queried independently, but both are
// exposed through this one service so the gateway/mobile app has a single
// place to go for search. SearchWork reads MongoDB job documents;
// SearchWorkers still queries PostgreSQL. Either backend can change later
// without changing this API.
type SearchService struct {
	workSearchRepository   *repository.WorkSearchRepository
	workerSearchRepository *repository.WorkerSearchRepository
}

func NewSearchService(
	workSearchRepository *repository.WorkSearchRepository,
	workerSearchRepository *repository.WorkerSearchRepository,
) *SearchService {
	return &SearchService{
		workSearchRepository:   workSearchRepository,
		workerSearchRepository: workerSearchRepository,
	}
}

func (s *SearchService) SearchWork(
	ctx context.Context,
	filters model.WorkSearchFilters,
) (*model.WorkSearchResponse, error) {
	return s.workSearchRepository.SearchWork(ctx, filters)
}

func (s *SearchService) SearchWorkers(
	ctx context.Context,
	filters model.WorkerSearchFilters,
) (*model.WorkerSearchResponse, error) {
	return s.workerSearchRepository.SearchWorkers(ctx, filters)
}
