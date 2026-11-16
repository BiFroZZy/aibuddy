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
	TeamID uuid.UUID
}

type StepStatus string

const (
	StepNotStarted StepStatus = "not_started"
	StepInProgress StepStatus = "in_progress"
	StepWaiting    StepStatus = "waiting"
	StepDone       StepStatus = "done"
)

func (s StepStatus) Valid() bool {
	switch s {
	case StepNotStarted, StepInProgress, StepWaiting, StepDone:
		return true
	}
	return false
}

type ConfirmBy string

const (
	ConfirmSelf   ConfirmBy = "self"
	ConfirmMentor ConfirmBy = "mentor"
)

type Step struct {
	ID            uuid.UUID  `json:"id"`
	ScenarioCode  string     `json:"scenario_code"`
	ScenarioTitle string     `json:"scenario_title"`
	Position      int        `json:"position"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	ConfirmBy     ConfirmBy  `json:"confirm_by"`
	Status        StepStatus `json:"status"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type Confirmation struct {
	InternID   uuid.UUID `json:"intern_id"`
	InternName string    `json:"intern_name"`
	StepID     uuid.UUID `json:"step_id"`
	StepTitle  string    `json:"step_title"`
	Since      time.Time `json:"since"`
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
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"-"`
	Tool           string          `json:"tool"`
	Args           json.RawMessage `json:"args"`
	Result         json.RawMessage `json:"result,omitempty"`
	Status         ActionStatus    `json:"status"`
	IdempotencyKey string          `json:"-"`
	ExternalID     *uuid.UUID      `json:"external_id,omitempty"`
	Error          string          `json:"error,omitempty"`
	Summary        string          `json:"summary,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

const (
	KindAbsence = "absence"
	KindAccess  = "access"

	RequestPending  = "pending_approval"
	RequestApproved = "approved"
	RequestRejected = "rejected"
)

type Request struct {
	ID              uuid.UUID       `json:"id"`
	Kind            string          `json:"kind"`
	UserID          uuid.UUID       `json:"user_id"`
	UserName        string          `json:"user_name,omitempty"`
	Payload         json.RawMessage `json:"payload"`
	Status          string          `json:"status"`
	ApproverRole    string          `json:"approver_role"`
	DecisionComment string          `json:"decision_comment,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	DecidedAt       *time.Time      `json:"decided_at,omitempty"`
}

type KBHit struct {
	Title   string `json:"title"`
	Source  string `json:"source"`
	Snippet string `json:"snippet"`
}

type Person struct {
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	Note     string `json:"note"`
}

type ChatMessage struct {
	ID        uuid.UUID `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Metrics struct {
	Since       time.Time      `json:"since"`
	Events      map[string]int `json:"events"`
	Tools       map[string]int `json:"tools"`
	ActiveUsers int            `json:"active_users"`
}

var (
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidArgs    = errors.New("invalid arguments")
	ErrNotFound       = errors.New("not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrConflict       = errors.New("conflict")
	ErrLLMUnavailable = errors.New("llm unavailable")
	ErrUnavailable    = errors.New("corporate service is temporarily unavailable")
)
