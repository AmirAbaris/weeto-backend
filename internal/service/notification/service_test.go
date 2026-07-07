package notification

import "testing"

func TestBookingCodeFromRescheduleToken(t *testing.T) {
	got := bookingCode(bookingPayload{
		RescheduleToken: "abcdef123456",
		BookingID:       "00000000-0000-0000-0000-000000000001",
	})
	if got != "abcdef" {
		t.Fatalf("got %q", got)
	}
}

func TestBookingCodeFromBookingID(t *testing.T) {
	got := bookingCode(bookingPayload{
		BookingID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
	})
	if got != "a1b2c3" {
		t.Fatalf("got %q", got)
	}
}
