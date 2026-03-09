package spotify

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/viper"
)

const (
	authURL     = "https://accounts.spotify.com/authorize"
	tokenURL    = "https://accounts.spotify.com/api/token"
	redirectURI = "http://127.0.0.1:8080/callback"
	scopes      = "playlist-read-private playlist-read-collaborative playlist-modify-public playlist-modify-private user-library-read user-library-modify"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

func authenticate(ctx context.Context) error {
	clientID := viper.GetString("spotify.client_id")
	if clientID == "" {
		return fmt.Errorf("spotify.client_id not set in config\n\n" +
			"1. Create an app at https://developer.spotify.com/dashboard\n" +
			"2. Set the redirect URI to http://localhost:8080/callback\n" +
			"3. Add your client ID to ~/.config/universal-music-library/config.yaml:\n" +
			"   spotify:\n" +
			"     client_id: YOUR_CLIENT_ID")
	}

	verifier, err := generateCodeVerifier()
	if err != nil {
		return fmt.Errorf("generating code verifier: %w", err)
	}
	challenge := codeChallenge(verifier)
	state := generateState()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errCh <- fmt.Errorf("auth error: %s", errParam)
			fmt.Fprintf(w, "Authentication failed: %s. You can close this tab.", errParam)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}
		codeCh <- code
		fmt.Fprint(w, "Authentication successful! You can close this tab.")
	})

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		return fmt.Errorf("starting callback server: %w", err)
	}
	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	params := url.Values{
		"client_id":             {clientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"scope":                 {scopes},
		"state":                 {state},
		"code_challenge_method": {"S256"},
		"code_challenge":        {challenge},
	}
	authFullURL := authURL + "?" + params.Encode()

	fmt.Println("Opening browser for Spotify authorization...")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n", authFullURL)
	openBrowser(authFullURL)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}

	token, err := exchangeCode(clientID, code, verifier)
	if err != nil {
		return fmt.Errorf("exchanging code: %w", err)
	}

	viper.Set("spotify.access_token", token.AccessToken)
	viper.Set("spotify.refresh_token", token.RefreshToken)
	viper.Set("spotify.token_expiry", time.Now().Add(time.Duration(token.ExpiresIn)*time.Second).Unix())

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	slog.Info("Spotify authentication successful")
	return nil
}

func exchangeCode(clientID, code, verifier string) (*tokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {clientID},
		"code_verifier": {verifier},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		return nil, fmt.Errorf("%s: %s", token.Error, token.ErrorDesc)
	}
	return &token, nil
}

func refreshAccessToken() error {
	clientID := viper.GetString("spotify.client_id")
	refreshToken := viper.GetString("spotify.refresh_token")
	if refreshToken == "" {
		return fmt.Errorf("no refresh token — run 'uml auth spotify' first")
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return err
	}
	if token.Error != "" {
		return fmt.Errorf("%s: %s", token.Error, token.ErrorDesc)
	}

	viper.Set("spotify.access_token", token.AccessToken)
	viper.Set("spotify.token_expiry", time.Now().Add(time.Duration(token.ExpiresIn)*time.Second).Unix())
	if token.RefreshToken != "" {
		viper.Set("spotify.refresh_token", token.RefreshToken)
	}

	return viper.WriteConfig()
}

func getAccessToken() (string, error) {
	expiry := viper.GetInt64("spotify.token_expiry")
	if time.Now().Unix() >= expiry-60 {
		if err := refreshAccessToken(); err != nil {
			return "", err
		}
	}
	token := viper.GetString("spotify.access_token")
	if token == "" {
		return "", fmt.Errorf("not authenticated — run 'uml auth spotify' first")
	}
	return token, nil
}

func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func codeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	default:
		return
	}
	args = append(args, url)
	exec.Command(cmd, args...).Start()
}
