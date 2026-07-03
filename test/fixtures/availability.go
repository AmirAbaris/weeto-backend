package fixtures

import availabilitysvc "github.com/AmirAbaris/weeto-backend/internal/service/availability"

func MonFri9to17(maxPerDay int32) availabilitysvc.Input {
	hours := make([]availabilitysvc.WorkingHourInput, 0, 5)
	for _, dow := range []int16{1, 2, 3, 4, 5} {
		hours = append(hours, availabilitysvc.WorkingHourInput{
			DayOfWeek: dow,
			StartTime: "09:00",
			EndTime:   "17:00",
		})
	}
	return availabilitysvc.Input{
		Timezone:            "Asia/Tehran",
		MaxInterviewsPerDay: maxPerDay,
		WorkingHours:        hours,
	}
}

func Monday9to17(maxPerDay int32) availabilitysvc.Input {
	return availabilitysvc.Input{
		Timezone:            "Asia/Tehran",
		MaxInterviewsPerDay: maxPerDay,
		WorkingHours: []availabilitysvc.WorkingHourInput{
			{DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00"},
		},
	}
}

func Monday9to17WithLunch(maxPerDay int32) availabilitysvc.Input {
	in := Monday9to17(maxPerDay)
	in.Breaks = []availabilitysvc.BreakInput{
		{DayOfWeek: 1, StartTime: "12:00", EndTime: "13:00"},
	}
	return in
}

func Tuesday9to17(maxPerDay int32) availabilitysvc.Input {
	return availabilitysvc.Input{
		Timezone:            "Asia/Tehran",
		MaxInterviewsPerDay: maxPerDay,
		WorkingHours: []availabilitysvc.WorkingHourInput{
			{DayOfWeek: 2, StartTime: "09:00", EndTime: "17:00"},
		},
	}
}

func WithTimeOff(in availabilitysvc.Input, startAt, endAt string) availabilitysvc.Input {
	in.TimeOff = append(in.TimeOff, availabilitysvc.TimeOffInput{
		StartAt: startAt,
		EndAt:   endAt,
	})
	return in
}
