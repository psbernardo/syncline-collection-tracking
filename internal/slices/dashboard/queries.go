package dashboard

import (
	"context"
	"time"
)

type Service struct {
	repository QueryRepository
	now        func() time.Time
}

func NewService(repository QueryRepository) *Service {
	return &Service{repository: repository, now: func() time.Time { return time.Now().UTC() }}
}

func (service *Service) Totals(ctx context.Context) (DashboardViewModel, error) {
	totals, err := service.repository.GetTotals(ctx, Query{Now: service.now()})
	if err != nil {
		return DashboardViewModel{}, err
	}
	return totals.ViewModel(), nil
}
