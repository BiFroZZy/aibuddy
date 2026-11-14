package repository

import (
	"aibuddy/internal/auth"
	"aibuddy/internal/domain"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var _ auth.Authenticator = (*MockTokenAuth)(nil)

type MockTokenAuth struct {
	DB *sql.DB
}

func (m *MockTokenAuth) Authenticate(ctx context.Context, token string) (domain.Identity, error) {
	if token == "" {
		return domain.Identity{}, domain.ErrUnauthorized
	}
	var (
		id domain.Identity
		role string 
	)
	err := m.DB.QueryRowContext(ctx, `
		SELECT u.id, u.role, COALESCE(u.team_id, '')
		FROM mock_auth_tokens t
		JOIN users u ON u.id = t.user_id
		WHERE t.token = $1`, token).Scan(&id.UserID, &role, &id.Team)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Identity{}, domain.ErrUnauthorized
	}
	if err != nil {
		return domain.Identity{}, fmt.Errorf("authenticate: %w", err)
	}
	id.Role = domain.Role(role)
	return id, nil
}