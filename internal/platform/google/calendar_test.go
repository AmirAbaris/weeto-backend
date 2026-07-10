package google

import (
	"testing"

	"google.golang.org/api/calendar/v3"
)

func TestMeetLinkFromEventHangoutLink(t *testing.T) {
	link := meetLinkFromEvent(&calendar.Event{HangoutLink: "https://meet.google.com/abc-defg-hij"})
	if link != "https://meet.google.com/abc-defg-hij" {
		t.Fatalf("got %q", link)
	}
}

func TestMeetLinkFromEventEntryPoint(t *testing.T) {
	link := meetLinkFromEvent(&calendar.Event{
		ConferenceData: &calendar.ConferenceData{
			EntryPoints: []*calendar.EntryPoint{
				{EntryPointType: "video", Uri: "https://meet.google.com/xyz"},
			},
		},
	})
	if link != "https://meet.google.com/xyz" {
		t.Fatalf("got %q", link)
	}
}

func TestMeetLinkFromEventEmpty(t *testing.T) {
	if meetLinkFromEvent(&calendar.Event{}) != "" {
		t.Fatal("expected empty link")
	}
}
