package integration

import (
	"io"
	"net/http"
	"testing"

	bookingsvc "github.com/AmirAbaris/weeto-backend/internal/service/booking"
	"github.com/AmirAbaris/weeto-backend/test/fixtures"
)

func TestDeleteInterviewTypeWithScheduledBookingBlocked(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	_, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Blocked Delete",
		Phone:  "+989121234567",
		Email:  "blocked@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	ts := newTestServer(t, env)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/interview-types/"+it.ID.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+IssueTestToken(t, env.UserID))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s, want 409", resp.StatusCode, body)
	}
}

func TestDeleteInterviewTypeAfterCancel(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	result, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Cancel Then Delete",
		Phone:  "+989121234567",
		Email:  "cancel@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	ts := newTestServer(t, env)
	defer ts.Close()
	token := IssueTestToken(t, env.UserID)

	cancelReq, err := http.NewRequest(http.MethodDelete, ts.URL+"/bookings/"+result.Booking.ID.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	cancelReq.Header.Set("Authorization", "Bearer "+token)

	cancelResp, err := http.DefaultClient.Do(cancelReq)
	if err != nil {
		t.Fatal(err)
	}
	cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusNoContent {
		t.Fatalf("cancel status = %d, want 204", cancelResp.StatusCode)
	}

	deleteReq, err := http.NewRequest(http.MethodDelete, ts.URL+"/interview-types/"+it.ID.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	deleteReq.Header.Set("Authorization", "Bearer "+token)

	deleteResp, err := http.DefaultClient.Do(deleteReq)
	if err != nil {
		t.Fatal(err)
	}
	defer deleteResp.Body.Close()
	if deleteResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(deleteResp.Body)
		t.Fatalf("delete status = %d, body = %s, want 204", deleteResp.StatusCode, body)
	}

	metaResp, err := http.Get(ts.URL + "/public/" + env.OrgSlug + "/" + it.Slug)
	if err != nil {
		t.Fatal(err)
	}
	defer metaResp.Body.Close()
	if metaResp.StatusCode != http.StatusNotFound {
		t.Fatalf("public metadata status = %d, want 404", metaResp.StatusCode)
	}
}

func TestDeleteInterviewTypeSuccess(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	env.UpsertAvailability(fixtures.Monday9to17(4))
	first := env.CreateInterviewType(60, 0)
	second := env.CreateInterviewType(30, 0)

	beforeSecond := len(env.ListSlots(second.ID))
	if beforeSecond != 0 {
		t.Fatalf("second type slots = %d, want 0 while first holds org quota", beforeSecond)
	}

	ts := newTestServer(t, env)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/interview-types/"+first.ID.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+IssueTestToken(t, env.UserID))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s, want 204", resp.StatusCode, body)
	}

	metaResp, err := http.Get(ts.URL + "/public/" + env.OrgSlug + "/" + first.Slug)
	if err != nil {
		t.Fatal(err)
	}
	metaResp.Body.Close()
	if metaResp.StatusCode != http.StatusNotFound {
		t.Fatalf("deleted type public status = %d, want 404", metaResp.StatusCode)
	}

	afterSecond := len(env.ListSlots(second.ID))
	if afterSecond == 0 {
		t.Fatal("remaining type should receive slots after delete freed org quota")
	}
}
