package plan_test

import (
	"testing"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/AmirAbaris/weeto-backend/internal/plan"
)

func TestMaxInterviewTypes(t *testing.T) {
	freeLimit := plan.MaxInterviewTypes(db.PlanTypeFree)
	if freeLimit == nil || *freeLimit != 3 {
		t.Fatalf("free limit = %v, want 3", freeLimit)
	}
	if plan.MaxInterviewTypes(db.PlanTypePro) != nil {
		t.Fatal("pro should be unlimited")
	}
}

func TestMaxMeetLinksPerMonth(t *testing.T) {
	freeLimit := plan.MaxMeetLinksPerMonth(db.PlanTypeFree)
	if freeLimit == nil || *freeLimit != 15 {
		t.Fatalf("free limit = %v, want 15", freeLimit)
	}
	if plan.MaxMeetLinksPerMonth(db.PlanTypeBusiness) != nil {
		t.Fatal("business should be unlimited")
	}
}
