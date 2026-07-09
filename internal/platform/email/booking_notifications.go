package email

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
)

type BookingNotificationData struct {
	Recipient          string
	CandidateName      string
	CandidateEmail     string
	CandidatePhone     string
	RecruiterEmail     string
	OrganizationName   string
	InterviewTypeTitle string
	SlotStartAt        time.Time
	SlotEndAt          time.Time
	PreviousSlotStart  time.Time
	PreviousSlotEnd    time.Time
	HasPreviousSlot    bool
	MeetLink           string
	MeetingLocation    string
	RescheduleURL      string
	CancelURL          string
	CancelledBy        string
}

func ParseBookingNotificationPayload(payload []byte, frontendURL string) (BookingNotificationData, error) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return BookingNotificationData{}, err
	}

	startAt, err := time.Parse(time.RFC3339, stringField(raw, "slot_start_at"))
	if err != nil {
		return BookingNotificationData{}, fmt.Errorf("slot_start_at: %w", err)
	}
	endAt, err := time.Parse(time.RFC3339, stringField(raw, "slot_end_at"))
	if err != nil {
		return BookingNotificationData{}, fmt.Errorf("slot_end_at: %w", err)
	}

	data := BookingNotificationData{
		Recipient:          stringField(raw, "recipient"),
		CandidateName:      stringField(raw, "candidate_name"),
		CandidateEmail:     stringField(raw, "candidate_email"),
		CandidatePhone:     stringField(raw, "candidate_phone"),
		RecruiterEmail:     stringField(raw, "recruiter_email"),
		OrganizationName:   stringField(raw, "organization_name"),
		InterviewTypeTitle: stringField(raw, "interview_type_title"),
		SlotStartAt:        startAt,
		SlotEndAt:          endAt,
		MeetLink:           stringField(raw, "meet_link"),
		MeetingLocation:    stringField(raw, "meeting_location"),
		CancelledBy:        stringField(raw, "cancelled_by"),
	}

	if prevStart := stringField(raw, "previous_slot_start_at"); prevStart != "" {
		if prevEnd := stringField(raw, "previous_slot_end_at"); prevEnd != "" {
			ps, err := time.Parse(time.RFC3339, prevStart)
			if err != nil {
				return BookingNotificationData{}, fmt.Errorf("previous_slot_start_at: %w", err)
			}
			pe, err := time.Parse(time.RFC3339, prevEnd)
			if err != nil {
				return BookingNotificationData{}, fmt.Errorf("previous_slot_end_at: %w", err)
			}
			data.PreviousSlotStart = ps
			data.PreviousSlotEnd = pe
			data.HasPreviousSlot = true
		}
	}

	rescheduleToken := stringField(raw, "reschedule_token")
	cancelToken := stringField(raw, "cancel_token")
	base := strings.TrimRight(frontendURL, "/")
	data.RescheduleURL = fmt.Sprintf("%s/reschedule/%s", base, rescheduleToken)
	data.CancelURL = fmt.Sprintf("%s/cancel/%s", base, cancelToken)

	return data, nil
}

func formatTimeRange(start, end time.Time) string {
	loc, _ := time.LoadLocation("Asia/Tehran")
	startLocal := start.In(loc)
	endLocal := end.In(loc)
	return fmt.Sprintf(
		"%s تا %s",
		startLocal.Format("2006/01/02 15:04"),
		endLocal.Format("15:04"),
	)
}

func meetingDetailsHTML(meetLink, meetingLocation string) string {
	var details strings.Builder
	if meetLink != "" {
		fmt.Fprintf(&details, `<p><strong>لینک جلسه:</strong> <a href="%s">%s</a></p>`,
	html.EscapeString(meetLink),
	html.EscapeString(meetLink))
	}
	if meetingLocation != "" {
		details.WriteString(fmt.Sprintf(
			"<p><strong>محل برگزاری:</strong> %s</p>",
			html.EscapeString(meetingLocation),
		))
	}
	return details.String()
}

func BookingRescheduledMessage(data BookingNotificationData) Message {
	timeRange := formatTimeRange(data.SlotStartAt, data.SlotEndAt)
	subject := fmt.Sprintf("تغییر زمان رزرو: %s — %s", data.InterviewTypeTitle, data.OrganizationName)

	var previous string
	if data.HasPreviousSlot {
		previous = fmt.Sprintf(
			"<p><strong>زمان قبلی:</strong> %s</p>",
			html.EscapeString(formatTimeRange(data.PreviousSlotStart, data.PreviousSlotEnd)),
		)
	}

	body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="fa" dir="rtl">
<body style="font-family: Tahoma, Arial, sans-serif; line-height: 1.6; color: #111;">
  <p>سلام %s،</p>
  <p>زمان رزرو شما برای <strong>%s</strong> در <strong>%s</strong> تغییر کرد.</p>
  %s
  <p><strong>زمان جدید:</strong> %s</p>
  %s
  <p>
    <a href="%s" style="display:inline-block;padding:10px 16px;background:#111;color:#fff;text-decoration:none;border-radius:6px;">تغییر زمان</a>
    &nbsp;
    <a href="%s" style="display:inline-block;padding:10px 16px;background:#b42318;color:#fff;text-decoration:none;border-radius:6px;">لغو رزرو</a>
  </p>
</body>
</html>`,
		html.EscapeString(data.CandidateName),
		html.EscapeString(data.InterviewTypeTitle),
		html.EscapeString(data.OrganizationName),
		previous,
		html.EscapeString(timeRange),
		meetingDetailsHTML(data.MeetLink, data.MeetingLocation),
		html.EscapeString(data.RescheduleURL),
		html.EscapeString(data.CancelURL),
	)

	return Message{To: data.CandidateEmail, Subject: subject, HTML: body}
}

func BookingCancelledCandidateMessage(data BookingNotificationData) Message {
	timeRange := formatTimeRange(data.SlotStartAt, data.SlotEndAt)
	subject := fmt.Sprintf("لغو رزرو: %s — %s", data.InterviewTypeTitle, data.OrganizationName)

	body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="fa" dir="rtl">
<body style="font-family: Tahoma, Arial, sans-serif; line-height: 1.6; color: #111;">
  <p>سلام %s،</p>
  <p>رزرو شما برای <strong>%s</strong> در <strong>%s</strong> لغو شد.</p>
  <p><strong>زمان:</strong> %s</p>
</body>
</html>`,
		html.EscapeString(data.CandidateName),
		html.EscapeString(data.InterviewTypeTitle),
		html.EscapeString(data.OrganizationName),
		html.EscapeString(timeRange),
	)

	return Message{To: data.CandidateEmail, Subject: subject, HTML: body}
}

func BookingCancelledRecruiterMessage(data BookingNotificationData) Message {
	timeRange := formatTimeRange(data.SlotStartAt, data.SlotEndAt)
	subject := fmt.Sprintf("لغو رزرو: %s", data.CandidateName)

	var intro string
	switch data.CancelledBy {
	case "recruiter":
		intro = fmt.Sprintf("شما رزرو <strong>%s</strong> را لغو کردید.", html.EscapeString(data.CandidateName))
	default:
		intro = fmt.Sprintf("<strong>%s</strong> رزرو خود را لغو کرد.", html.EscapeString(data.CandidateName))
	}

	body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="fa" dir="rtl">
<body style="font-family: Tahoma, Arial, sans-serif; line-height: 1.6; color: #111;">
  <p>سلام،</p>
  <p>%s</p>
  <p><strong>نوع مصاحبه:</strong> %s</p>
  <p><strong>زمان:</strong> %s</p>
  <p><strong>ایمیل:</strong> %s</p>
  <p><strong>تلفن:</strong> %s</p>
</body>
</html>`,
		intro,
		html.EscapeString(data.InterviewTypeTitle),
		html.EscapeString(timeRange),
		html.EscapeString(data.CandidateEmail),
		html.EscapeString(data.CandidatePhone),
	)

	return Message{To: data.RecruiterEmail, Subject: subject, HTML: body}
}

func BookingCreatedRecruiterMessage(data BookingNotificationData) Message {
	timeRange := formatTimeRange(data.SlotStartAt, data.SlotEndAt)
	subject := fmt.Sprintf("رزرو جدید: %s — %s", data.CandidateName, data.InterviewTypeTitle)

	body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="fa" dir="rtl">
<body style="font-family: Tahoma, Arial, sans-serif; line-height: 1.6; color: #111;">
  <p>سلام،</p>
  <p><strong>%s</strong> یک زمان برای <strong>%s</strong> در <strong>%s</strong> رزرو کرد.</p>
  <p><strong>زمان:</strong> %s</p>
  <p><strong>ایمیل:</strong> %s</p>
  <p><strong>تلفن:</strong> %s</p>
  %s
</body>
</html>`,
		html.EscapeString(data.CandidateName),
		html.EscapeString(data.InterviewTypeTitle),
		html.EscapeString(data.OrganizationName),
		html.EscapeString(timeRange),
		html.EscapeString(data.CandidateEmail),
		html.EscapeString(data.CandidatePhone),
		meetingDetailsHTML(data.MeetLink, data.MeetingLocation),
	)

	return Message{To: data.RecruiterEmail, Subject: subject, HTML: body}
}

func BookingReminder24hMessage(data BookingNotificationData) Message {
	timeRange := formatTimeRange(data.SlotStartAt, data.SlotEndAt)
	subject := fmt.Sprintf("یادآوری مصاحبه فردا: %s — %s", data.InterviewTypeTitle, data.OrganizationName)

	body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="fa" dir="rtl">
<body style="font-family: Tahoma, Arial, sans-serif; line-height: 1.6; color: #111;">
  <p>سلام %s،</p>
  <p>یادآوری: مصاحبه <strong>%s</strong> در <strong>%s</strong> فردا برگزار می‌شود.</p>
  <p><strong>زمان:</strong> %s</p>
  %s
  <p>
    <a href="%s" style="display:inline-block;padding:10px 16px;background:#111;color:#fff;text-decoration:none;border-radius:6px;">تغییر زمان</a>
    &nbsp;
    <a href="%s" style="display:inline-block;padding:10px 16px;background:#b42318;color:#fff;text-decoration:none;border-radius:6px;">لغو رزرو</a>
  </p>
</body>
</html>`,
		html.EscapeString(data.CandidateName),
		html.EscapeString(data.InterviewTypeTitle),
		html.EscapeString(data.OrganizationName),
		html.EscapeString(timeRange),
		meetingDetailsHTML(data.MeetLink, data.MeetingLocation),
		html.EscapeString(data.RescheduleURL),
		html.EscapeString(data.CancelURL),
	)

	return Message{To: data.CandidateEmail, Subject: subject, HTML: body}
}
