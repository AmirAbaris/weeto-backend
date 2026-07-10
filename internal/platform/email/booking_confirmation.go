package email

import (
	"fmt"
	"html"
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

// ParseBookingConfirmationPayload parses a booking_created candidate notification payload.
func ParseBookingConfirmationPayload(payload []byte, frontendURL string) (BookingConfirmationData, error) {
	data, err := ParseBookingNotificationPayload(payload, frontendURL)
	if err != nil {
		return BookingConfirmationData{}, err
	}
	return BookingConfirmationData{
		CandidateName:      data.CandidateName,
		CandidateEmail:     data.CandidateEmail,
		OrganizationName:   data.OrganizationName,
		InterviewTypeTitle: data.InterviewTypeTitle,
		SlotStartAt:        data.SlotStartAt,
		SlotEndAt:          data.SlotEndAt,
		MeetLink:           data.MeetLink,
		MeetingLocation:    data.MeetingLocation,
		RescheduleURL:      data.RescheduleURL,
		CancelURL:          data.CancelURL,
	}, nil
}

func BookingConfirmationMessage(data BookingConfirmationData) Message {
	timeRange := formatTimeRange(data.SlotStartAt, data.SlotEndAt)
	subject := fmt.Sprintf("تایید رزرو: %s — %s", data.InterviewTypeTitle, data.OrganizationName)

	body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="fa" dir="rtl">
<body style="font-family: Tahoma, Arial, sans-serif; line-height: 1.6; color: #111;">
  <p>سلام %s،</p>
  <p>رزرو شما برای <strong>%s</strong> در <strong>%s</strong> ثبت شد.</p>
  <p><strong>زمان:</strong> %s</p>
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
		html.EscapeString(timeRange),
		meetingDetailsHTML(data.MeetLink, data.MeetingLocation),
		html.EscapeString(data.RescheduleURL),
		html.EscapeString(data.CancelURL),
	)

	return Message{
		To:      data.CandidateEmail,
		Subject: subject,
		HTML:    body,
	}
}
