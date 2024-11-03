package csrf

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

// Global CSRF token and expiration details
var (
	Token         string
	ExpiryDate    time.Time
	mu            sync.RWMutex
	Client        *http.Client
	ValidDuration = 10 * time.Minute // CSRF token validity duration (can vary based on requirements)
)

// Init initializes the HTTP client with a configurable timeout
func Init(clientTimeout time.Duration) {
	Client = &http.Client{
		Timeout: clientTimeout,
	}
}

// UpdateCSRF fetches a new CSRF token and updates it in a thread-safe manner
func UpdateCSRF(cookie string) error {
	mu.Lock()
	defer mu.Unlock()

	// If token is still valid, skip refresh
	if time.Now().Before(ExpiryDate) {
		return nil
	}

	// HINT: if things start to go wrong, change "/login" to "/logout"
	req, err := http.NewRequest("POST", "https://auth.roblox.com/v2/login", nil)
	if err != nil {
		return err
	}

	// Add the Roblox security cookie to the request
	req.AddCookie(&http.Cookie{
		Name:  ".ROBLOSECURITY",
		Value: cookie,
	})

	// Execute the request
	resp, err := Client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// Retrieve the CSRF token from the response header
	newToken := resp.Header.Get("x-csrf-token")
	if newToken == "" {
		log.Warn("Failed to retreive CSRF token from Roblox API, might wanna re-check the Cookie.")
		return errors.New("failed to retrieve CSRF token")
	}

	// Update token and set the expiration
	Token = newToken
	ExpiryDate = time.Now().Add(ValidDuration)

	return nil
}

// GetCSRF safely returns the current CSRF token, ensuring it's valid
func GetCSRF(cookie string) (string, error) {
	mu.RLock()
	if Token != "" && time.Now().Before(ExpiryDate) {
		defer mu.RUnlock()
		return Token, nil
	}
	mu.RUnlock()

	// If the token is invalid or expired, refresh it
	if err := UpdateCSRF(cookie); err != nil {
		return "", err
	}

	// Safely return the updated token
	mu.RLock()
	defer mu.RUnlock()
	return Token, nil
}
