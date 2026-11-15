package domain

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleIntern  Role = "intern"
	RoleMentor  Role = "mentor"
	RoleManager Role = "manager"
	RoleHRBP    Role = "hr_bp"
)

type Identity struct {
	UserID uuid.UUID
	Role   Role
	Team   string
}

type ActionStatus string

const (
	ActionPending   ActionStatus = "pending_confirmation"
	ActionRunning   ActionStatus = "running"
	ActionDone      ActionStatus = "done"
	ActionFailed    ActionStatus = "failed"
	ActionCancelled ActionStatus = "cancelled"
	ActionDenied    ActionStatus = "denied"
)

type Action struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Tool           string
	Args           json.RawMessage
	Result         json.RawMessage
	Status         ActionStatus
	IdempotencyKey string
	ExternalID     string
	Error          string
	CreatedAt      time.Time
}

var (
	ErrUnknownTool    = errors.New("unknown tool")
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidArgs    = errors.New("invalid arguments")
	ErrNotFound       = errors.New("not found")
	ErrAlreadyHandled = errors.New("action already handled")
	ErrUnauthorized   = errors.New("unauthorized")
)
