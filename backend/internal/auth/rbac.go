package auth

import "aibuddy/internal/domain"

const (
	PermProgress       = "get_my_progress"
	PermSearchKB       = "search_kb"
	PermWhoToAsk       = "get_who_to_ask"
	PermAbsenceCreate  = "create_absence_request"
	PermAbsenceStatus  = "get_absence_status"
	PermAccessRequest  = "request_access"
	PermConfirmStep    = "confirm_step"
	PermDecideRequest  = "decide_request"
	PermViewMetrics    = "view_metrics"
	PermManageOwnSteps = "manage_own_steps"
)

type Policy interface {
	Allowed(role domain.Role, tool string) bool
}

type StaticPolicy map[domain.Role]map[string]struct{}

func (p StaticPolicy) Allowed(role domain.Role, tool string) bool {
	_, ok := p[role][tool]
	return ok
}

func set(names ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}
	return m
}

func DefaultPolicy() StaticPolicy {
	return StaticPolicy{
		domain.RoleIntern: set(
			PermProgress, PermSearchKB, PermWhoToAsk,
			PermAbsenceCreate, PermAbsenceStatus, PermAccessRequest,
			PermManageOwnSteps,
		),
		domain.RoleMentor:  set(PermProgress, PermSearchKB, PermWhoToAsk, PermConfirmStep),
		domain.RoleManager: set(PermProgress, PermSearchKB, PermWhoToAsk, PermDecideRequest, PermViewMetrics),
		domain.RoleHRBP:    set(PermSearchKB, PermWhoToAsk, PermViewMetrics),
	}
}
