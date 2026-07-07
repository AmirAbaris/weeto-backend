package email

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
)

type BookingConfirmationData struct {
	CandidateName      string
	CandidateEmail     string
	OrganizationName   string
	InterviewTypeTitle string
	SlotStartAt        time.Time
	SlotEndAt          time.Time
	MeetLink           string
	MeetingLocation    string
	RescheduleURL      string
	CancelURL          string
}

func ParseBookingConfirmationPayload(payload []byte, frontendURL string) (BookingConfirmationData, error) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return BookingConfirmationData{}, err
	}

	startAt, err := time.Parse(time.RFC3339, stringField(raw, "slot_start_at"))
	if err != nil {
		return BookingConfirmationData{}, fmt.Errorf("slot_start_at: %w", err)
	}
	endAt, err := time.Parse(time.RFC3339, stringField(raw, "slot_end_at"))
	if err != nil {
		return BookingConfirmationData{}, fmt.Errorf("slot_end_at: %w", err)
	}

	rescheduleToken := stringField(raw, "reschedule_token")
	cancelToken := stringField(raw, "cancel_token")
	base := strings.TrimRight(frontendURL, "/")

	return BookingConfirmationData{
		CandidateName:      stringField(raw, "candidate_name"),
		CandidateEmail:     stringField(raw, "candidate_email"),
		OrganizationName:   stringField(raw, "organization_name"),
		InterviewTypeTitle: stringField(raw, "interview_type_title"),
		SlotStartAt:        startAt,
		SlotEndAt:          endAt,
		MeetLink:           stringField(raw, "meet_link"),
		MeetingLocation:    stringField(raw, "meeting_location"),
		RescheduleURL:      fmt.Sprintf("%s/reschedule/%s", base, rescheduleToken),
		CancelURL:          fmt.Sprintf("%s/cancel/%s", base, cancelToken),
	}, nil
}

func BookingConfirmationMessage(data BookingConfirmationData) Message {
	loc, _ := time.LoadLocation("Asia/Tehran")
	startLocal := data.SlotStartAt.In(loc)
	endLocal := data.SlotEndAt.In(loc)
	timeRange := fmt.Sprintf(
		"%s تا %s",
		startLocal.Format("2006/01/02 15:04"),
		endLocal.Format("15:04"),
	)

	subject := fmt.Sprintf("تایید رزرو: %s — %s", data.InterviewTypeTitle, data.OrganizationName)

	var details strings.Builder
	details.WriteString(fmt.Sprintf("<p><strong>زمان:</strong> %s</p>", html.EscapeString(timeRange)))
	if data.MeetLink != "" {
		details.WriteString(fmt.Sprintf(
			`<p><strong>لینک جلسه:</strong> <a href="%s">%s</a></p>`,
			html.EscapeString(data.MeetLink),
			html.EscapeString(data.MeetLink),
		))
	}
	if data.MeetingLocation != "" {
		details.WriteString(fmt.Sprintf(
			"<p><strong>محل برگزاری:</strong> %s</p>",
			html.EscapeString(data.MeetingLocation),
		))
	}

	body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="fa" dir="rtl">
<body style="font-family: Tahoma, Arial, sans-serif; line-height: 1.6; color: #111;">
  <p>سلام %s،</p>
  <p>رزرو شما برای <strong>%s</strong> در <strong>%s</strong> ثبت شد.</p>
  %s
  <p>
    <a href="%s" style="display:inline-block;padding:10px 16px;background:#111;color:#fff;text-decoration:none;border-radius:6px;">تغییر زمان</a>
    &nbsp;
    <a href="%s" style="display:inline-block;padding:10px 16px;background:#b42318;color:#fff;text-decoration:none;border-radius:6px;">لغو رزرو</a>
  </p>
  <p style="color:#666;font-size:13px;">تا ۴ ساعت قبل از شروع جلسه می‌توانید زمان را تغییر دهید یا رزرو را لغو کنید.</p>
</body>
</html>`,
		html.EscapeString(data.CandidateName),
		html.EscapeString(data.InterviewTypeTitle),
		html.EscapeString(data.OrganizationName),
		details.String(),
		html.EscapeString(data.RescheduleURL),
		html.EscapeString(data.CancelURL),
	)

	return Message{
		To:      data.CandidateEmail,
		Subject: subject,
		HTML:    body,
	}
}

func stringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
