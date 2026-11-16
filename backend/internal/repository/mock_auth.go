package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"aibuddy/internal/auth"
	"aibuddy/internal/domain"
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
		id   domain.Identity
		role string
	)
	err := m.DB.QueryRowContext(ctx, `
		SELECT u.id, u.role, COALESCE(t.name, ''),
		       COALESCE(u.team_id, '00000000-0000-0000-0000-000000000000'::uuid)
		FROM mock_auth_tokens tk
		JOIN users u ON u.id = tk.user_id
		LEFT JOIN teams t ON t.id = u.team_id
		WHERE tk.token = $1`, token).Scan(&id.UserID, &role, &id.Team, &id.TeamID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Identity{}, domain.ErrUnauthorized
	}
	if err != nil {
		return domain.Identity{}, fmt.Errorf("authenticate: %w", err)
	}
	id.Role = domain.Role(role)
	return id, nil
}
