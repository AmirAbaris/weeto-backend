package plan

import "github.com/AmirAbaris/weeto-backend/internal/db"

const (
	FreeMaxInterviewTypes      int32 = 3
	FreeMaxMeetLinksPerMonth   int32 = 15
)

func MaxInterviewTypes(p db.PlanType) *int32 {
	if p == db.PlanTypeFree {
		limit := FreeMaxInterviewTypes
		return &limit
	}
	return nil
}

func MaxMeetLinksPerMonth(p db.PlanType) *int32 {
	if p == db.PlanTypeFree {
		limit := FreeMaxMeetLinksPerMonth
		return &limit
	}
	return nil
}

func MeetLinksLimitForIncrement(p db.PlanType) int32 {
	if limit := MaxMeetLinksPerMonth(p); limit != nil {
		return *limit
	}
	// Pro/business: SQL gate uses plan != 'free'; value is unused but must be positive.
	return FreeMaxMeetLinksPerMonth
}
