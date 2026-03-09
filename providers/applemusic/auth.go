package applemusic

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

const authPageHTML = `<!DOCTYPE html>
<html>
<head><title>Apple Music Authorization</title></head>
<body>
	<h1>Apple Music Authorization</h1>
	<p>Click the button below to authorize Universal Music Library with Apple Music.</p>
	<button id="auth-btn" disabled>Loading MusicKit...</button>
	<p id="status"></p>
	<script src="https://js-cdn.music.apple.com/musickit/v3/musickit.js" crossorigin></script>
	<script>
		document.addEventListener('musickitloaded', async () => {
			try {
				await MusicKit.configure({
					developerToken: '{{DEVELOPER_TOKEN}}',
					app: { name: 'Universal Music Library', build: '1.0.0' }
				});
				const btn = document.getElementById('auth-btn');
				btn.disabled = false;
				btn.textContent = 'Authorize with Apple Music';
				btn.addEventListener('click', async () => {
					btn.disabled = true;
					btn.textContent = 'Authorizing...';
					document.getElementById('status').textContent = '';
					try {
						const music = MusicKit.getInstance();
						const userToken = await music.authorize();
						const resp = await fetch('/callback', {
							method: 'POST',
							headers: {'Content-Type': 'application/json'},
							body: JSON.stringify({token: userToken})
						});
						if (resp.ok) {
							document.getElementById('status').textContent = 'Authorization successful! You can close this tab.';
							document.getElementById('status').style.color = 'green';
						} else { throw new Error('Failed to send token to CLI'); }
					} catch (err) {
						document.getElementById('status').textContent = 'Authorization failed: ' + err.message;
						document.getElementById('status').style.color = 'red';
						btn.disabled = false;
						btn.textContent = 'Try Again';
					}
				});
			} catch (err) {
				document.getElementById('status').textContent = 'Failed to load MusicKit: ' + err.message;
				document.getElementById('status').style.color = 'red';
			}
		});
	</script>
</body>
</html>`

func authenticate(ctx context.Context) error {
	teamID := viper.GetString("apple_music.team_id")
	keyID := viper.GetString("apple_music.key_id")
	privateKeyPath := viper.GetString("apple_music.private_key_path")

	if teamID == "" || keyID == "" || privateKeyPath == "" {
		return fmt.Errorf("apple_music config not set\n\n" +
			"1. Enable MusicKit in your Apple Developer account\n" +
			"2. Create a MusicKit private key (.p8 file)\n" +
			"3. Add to ~/.config/universal-music-library/config.yaml:\n" +
			"   apple_music:\n" +
			"     team_id: YOUR_TEAM_ID\n" +
			"     key_id: YOUR_KEY_ID\n" +
			"     private_key_path: /path/to/AuthKey_XXXX.p8\n" +
			"     storefront: us")
	}

	devToken, err := generateDeveloperToken(teamID, keyID, privateKeyPath)
	if err != nil {
		return fmt.Errorf("generating developer token: %w", err)
	}

	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)

	page := strings.Replace(authPageHTML, "{{DEVELOPER_TOKEN}}", devToken, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, page)
	})
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
			errCh <- fmt.Errorf("invalid callback")
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		tokenCh <- body.Token
		w.WriteHeader(http.StatusOK)
	})

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		return fmt.Errorf("starting auth server: %w", err)
	}
	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	authURL := "http://127.0.0.1:8080"
	fmt.Println("Opening browser for Apple Music authorization...")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n", authURL)
	openBrowser(authURL)

	var userToken string
	select {
	case userToken = <-tokenCh:
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}

	viper.Set("apple_music.music_user_token", userToken)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	slog.Info("Apple Music authentication successful")
	return nil
}

func generateDeveloperToken(teamID, keyID, privateKeyPath string) (string, error) {
	keyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("reading private key: %w", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block from private key")
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parsing private key: %w", err)
	}

	ecKey, ok := parsedKey.(*ecdsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("private key is not an ECDSA key")
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": teamID,
		"iat": now.Unix(),
		"exp": now.Add(180 * 24 * time.Hour).Unix(),
	})
	token.Header["kid"] = keyID

	return token.SignedString(ecKey)
}

func getDeveloperToken() (string, error) {
	teamID := viper.GetString("apple_music.team_id")
	keyID := viper.GetString("apple_music.key_id")
	privateKeyPath := viper.GetString("apple_music.private_key_path")

	if teamID == "" || keyID == "" || privateKeyPath == "" {
		return "", fmt.Errorf("apple_music config not set — run 'uml auth apple-music' first")
	}

	return generateDeveloperToken(teamID, keyID, privateKeyPath)
}

func getMusicUserToken() (string, error) {
	token := viper.GetString("apple_music.music_user_token")
	if token == "" {
		return "", fmt.Errorf("not authenticated — run 'uml auth apple-music' first")
	}
	return token, nil
}

func getStorefront() string {
	sf := viper.GetString("apple_music.storefront")
	if sf == "" {
		return "us"
	}
	return sf
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
