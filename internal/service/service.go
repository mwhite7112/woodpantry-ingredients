package service

import (
	"database/sql"

	"github.com/mwhite7112/woodpantry-ingredients/internal/db"
)

// Service holds all dependencies for the ingredient service layer.
type Service struct {
	q         db.Querier
	sqlDB     *sql.DB
	threshold float64
}

// New creates a new Service.
func New(q db.Querier, sqlDB *sql.DB, threshold float64) *Service {
	return &Service{q: q, sqlDB: sqlDB, threshold: threshold}
}

// Queries exposes the underlying db.Querier for direct use by handlers that
// don't require service-layer logic.
func (s *Service) Queries() db.Querier {
	return s.q
}
