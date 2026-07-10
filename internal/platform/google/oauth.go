package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	scopeCalendarEvents = "https://www.googleapis.com/auth/calendar.events"
	scopeUserEmail      = "https://www.googleapis.com/auth/userinfo.email"
)

type OAuth struct {
	cfg    *config.Config
	oauth2 *oauth2.Config
}

type TokenExchangeResult struct {
	GoogleID     string
	RefreshToken string
}

type googleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func NewOAuth(cfg *config.Config) *OAuth {
	return &OAuth{
		cfg: cfg,
		oauth2: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes:       []string{scopeCalendarEvents, scopeUserEmail},
			Endpoint:     google.Endpoint,
		},
	}
}

// generate google login URL
func (o *OAuth) AuthCodeURL(state string) string {
	return o.oauth2.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// take the temp auth code from google and exchange it for permanent refresh token
func (o *OAuth) Exchange(ctx context.Context, code string) (TokenExchangeResult, error) {
	token, err := o.oauth2.Exchange(ctx, code)
	if err != nil {
		return TokenExchangeResult{}, fmt.Errorf("exchange code: %w", err)
	}

	if token.RefreshToken == "" {
		return TokenExchangeResult{}, fmt.Errorf("no refresh token returned; revoke app access in Google account and reconnect")
	}

	info, err := fetchUserInfo(ctx, token.AccessToken)
	if err != nil {
		return TokenExchangeResult{}, err
	}

	return TokenExchangeResult{
		GoogleID:     info.ID,
		RefreshToken: token.RefreshToken,
	}, nil
}

// using access token to know the user from google
func fetchUserInfo(ctx context.Context, accessToken string) (googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return googleUserInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// do sends an http req
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return googleUserInfo{}, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return googleUserInfo{}, fmt.Errorf("userinfo status %d: %s", resp.StatusCode, string(body))
	}

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return googleUserInfo{}, fmt.Errorf("decode userinfo: %w", err)
	}
	if info.ID == "" {
		return googleUserInfo{}, fmt.Errorf("userinfo missing id")
	}

	return info, nil
}
