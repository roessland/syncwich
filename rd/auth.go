package rd

import (
	"errors"
	"time"

	"github.com/roessland/runalyzedump/runalyze"
)

// AuthService handles authentication and session management
type AuthService struct {
	client RunalyzeClient
	logger Logger
}

// NewAuthService creates a new authentication service
func NewAuthService(client RunalyzeClient, logger Logger) *AuthService {
	return &AuthService{
		client: client,
		logger: logger,
	}
}

// EnsureAuthenticated ensures the client is authenticated and ready to use
// It will attempt to verify the session and login if necessary
func (a *AuthService) EnsureAuthenticated() error {
	a.logger.Debug("attempting to verify login")

	// Try to get data to verify login
	_, err := a.client.GetDataBrowser(time.Now())
	if err != nil {
		// If we got redirected to login, try to login and retry
		if errors.Is(err, runalyze.ErrRedirectedToLogin) {
			a.logger.Info("attempting login")

			if err := a.client.Login(); err != nil {
				return err
			}

			// Retry getting data after successful login
			_, err = a.client.GetDataBrowser(time.Now())
			if err != nil {
				return err
			}

			// Persist cookies immediately after successful login verification
			if err := a.client.PersistCookies(); err != nil {
				a.logger.Warn("failed to persist cookies", "error", err)
			}

			a.logger.Info("successfully logged in to Runalyze")
			return nil
		}
		return err
	}

	// Persist cookies immediately after successful verification with existing cookies
	if err := a.client.PersistCookies(); err != nil {
		a.logger.Warn("failed to persist cookies", "error", err)
	}

	a.logger.Info("using existing Runalyze session")
	return nil
}
