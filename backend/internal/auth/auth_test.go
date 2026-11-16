package auth

import (
	"aibuddy/internal/domain"
	"context"
	"github.com/google/uuid"
	"testing"
)

func TestIdentityContext(t *testing.T) {
	want := domain.Identity{UserID: uuid.New(), Role: domain.RoleIntern, Team: "backend"}
	got, ok := IdentityFrom(WithIdentity(context.Background(), want))
	if !ok || got != want {
		t.Fatalf("got=%#v ok=%v", got, ok)
	}
	if _, ok := IdentityFrom(context.Background()); ok {
		t.Fatal("identity unexpectedly found")
	}
}
func TestDefaultPolicy(t *testing.T) {
	p := DefaultPolicy()
	cases := []struct {
		r    domain.Role
		tool string
		want bool
	}{{domain.RoleIntern, "create_absence_request", true}, {domain.RoleMentor, "create_absence_request", false}, {domain.RoleManager, "search_kb", true}, {domain.RoleHRBP, "get_my_progress", false}, {"unknown", "search_kb", false}}
	for _, c := range cases {
		if got := p.Allowed(c.r, c.tool); got != c.want {
			t.Errorf("Allowed(%s,%s)=%v", c.r, c.tool, got)
		}
	}
	if (StaticPolicy{}).Allowed(domain.RoleIntern, "x") {
		t.Fatal("empty policy allowed tool")
	}
}
