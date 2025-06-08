package rd

import (
	"fmt"
	"testing"
	"time"

	"github.com/roessland/runalyzedump/runalyze"
)

func TestAuthService_EnsureAuthenticated_AlreadyLoggedIn(t *testing.T) {
	// Arrange - Client already authenticated, GetDataBrowser succeeds
	mockClient := &MockRunalyzeClient{}
	mockLogger := &MockLogger{}
	authService := NewAuthService(mockClient, mockLogger)

	// Act
	err := authService.EnsureAuthenticated()

	// Assert
	if err != nil {
		t.Errorf("Expected no error for already authenticated client, got: %v", err)
	}

	// Verify GetDataBrowser was called but Login was not
	if mockClient.LoginCalled {
		t.Error("Expected Login not to be called for already authenticated client")
	}

	// Verify cookies were persisted
	if !mockClient.PersistCalled {
		t.Error("Expected PersistCookies to be called")
	}

	// Check logs
	if len(mockLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info log, got %d", len(mockLogger.InfoCalls))
	}
	if mockLogger.InfoCalls[0].Message != "using existing Runalyze session" {
		t.Errorf("Expected session reuse message, got: %s", mockLogger.InfoCalls[0].Message)
	}
}

func TestAuthService_EnsureAuthenticated_NeedsLogin(t *testing.T) {
	// Arrange - Client needs to login (gets redirect error first time)
	callCount := 0
	mockClient := &MockRunalyzeClient{
		GetDataBrowserFunc: func(date time.Time) ([]byte, error) {
			callCount++
			if callCount == 1 {
				return nil, runalyze.ErrRedirectedToLogin // First call fails
			}
			return []byte("<html>success</html>"), nil // Second call succeeds
		},
	}
	mockLogger := &MockLogger{}
	authService := NewAuthService(mockClient, mockLogger)

	// Act
	err := authService.EnsureAuthenticated()

	// Assert
	if err != nil {
		t.Errorf("Expected no error after successful login, got: %v", err)
	}

	// Verify Login was called
	if !mockClient.LoginCalled {
		t.Error("Expected Login to be called")
	}

	// Verify cookies were persisted
	if !mockClient.PersistCalled {
		t.Error("Expected PersistCookies to be called")
	}

	// Check logs - should have both login attempt and success
	foundLoginAttempt := false
	foundLoginSuccess := false
	for _, call := range mockLogger.InfoCalls {
		if call.Message == "attempting login" {
			foundLoginAttempt = true
		}
		if call.Message == "successfully logged in to Runalyze" {
			foundLoginSuccess = true
		}
	}
	if !foundLoginAttempt {
		t.Error("Expected login attempt log message")
	}
	if !foundLoginSuccess {
		t.Error("Expected login success log message")
	}
}

func TestAuthService_EnsureAuthenticated_LoginFails(t *testing.T) {
	// Arrange - Login fails
	mockClient := &MockRunalyzeClient{
		BrowserError: runalyze.ErrRedirectedToLogin,
		LoginError:   fmt.Errorf("invalid credentials"),
	}
	mockLogger := &MockLogger{}
	authService := NewAuthService(mockClient, mockLogger)

	// Act
	err := authService.EnsureAuthenticated()

	// Assert
	if err == nil {
		t.Error("Expected error when login fails")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("Expected login error, got: %v", err)
	}

	// Verify Login was attempted
	if !mockClient.LoginCalled {
		t.Error("Expected Login to be called")
	}
}

func TestAuthService_EnsureAuthenticated_BrowserErrorAfterLogin(t *testing.T) {
	// Arrange - Login succeeds but subsequent GetDataBrowser fails
	callCount := 0
	mockClient := &MockRunalyzeClient{
		GetDataBrowserFunc: func(date time.Time) ([]byte, error) {
			callCount++
			if callCount == 1 {
				return nil, runalyze.ErrRedirectedToLogin // First call fails - triggers login
			}
			return nil, fmt.Errorf("network error") // Second call fails after login
		},
	}
	mockLogger := &MockLogger{}
	authService := NewAuthService(mockClient, mockLogger)

	// Act
	err := authService.EnsureAuthenticated()

	// Assert
	if err == nil {
		t.Error("Expected error when GetDataBrowser fails after login")
	}
	if err.Error() != "network error" {
		t.Errorf("Expected network error, got: %v", err)
	}

	// Verify Login was called
	if !mockClient.LoginCalled {
		t.Error("Expected Login to be called")
	}
}

func TestAuthService_EnsureAuthenticated_NonRedirectError(t *testing.T) {
	// Arrange - GetDataBrowser fails with non-redirect error
	mockClient := &MockRunalyzeClient{
		BrowserError: fmt.Errorf("network timeout"),
	}
	mockLogger := &MockLogger{}
	authService := NewAuthService(mockClient, mockLogger)

	// Act
	err := authService.EnsureAuthenticated()

	// Assert
	if err == nil {
		t.Error("Expected error when GetDataBrowser fails with non-redirect error")
	}
	if err.Error() != "network timeout" {
		t.Errorf("Expected network timeout error, got: %v", err)
	}

	// Verify Login was NOT called (since it wasn't a redirect error)
	if mockClient.LoginCalled {
		t.Error("Expected Login not to be called for non-redirect errors")
	}
}
