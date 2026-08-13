package dashboard

import (
	"context"
	"time"
)

type Service struct {
	repository QueryRepository
	now        func() time.Time
}

type CompanyTotalsRepository interface {
	GetCompanyTotals(ctx context.Context, companyAccountID int64, query Query) (CompanyTotals, error)
}

type CompanyTotalsService struct {
	repository CompanyTotalsRepository
	now        func() time.Time
}

func NewCompanyTotalsService(repository CompanyTotalsRepository) *CompanyTotalsService {
	return &CompanyTotalsService{repository: repository, now: func() time.Time { return time.Now().UTC() }}
}

func (service *CompanyTotalsService) Totals(ctx context.Context, companyAccountID int64) (CompanyTotalsViewModel, error) {
	totals, err := service.repository.GetCompanyTotals(ctx, companyAccountID, Query{Now: service.now()})
	if err != nil {
		return CompanyTotalsViewModel{}, err
	}
	return totals.ViewModel(), nil
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
