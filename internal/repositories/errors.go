package repositories

import (
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/andrespalacio/finapp-backend/pkg/apperror"
)

// mapNotFoundErr maps pgx.ErrNoRows to apperror.ErrNotFound, wrapping any
// other error as apperror.ErrInternal. Shared by every single-row lookup
// across the repositories in this package.
func mapNotFoundErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.ErrNotFound
	}
	return apperror.Wrap(apperror.ErrInternal, err)
}
