package rd

import (
	"fmt"
	"time"
)

// MockRunalyzeClient implements RunalyzeClient for testing
type MockRunalyzeClient struct {
	FitData            []byte
	TcxData            []byte
	FitError           error
	TcxError           error
	LoginError         error
	BrowserError       error
	LoginCalled        bool
	PersistCalled      bool
	GetDataBrowserFunc func(date time.Time) ([]byte, error) // Allow custom behavior
}

func (m *MockRunalyzeClient) GetFit(id string) ([]byte, string, error) {
	if m.FitError != nil {
		return nil, "", m.FitError
	}
	return m.FitData, "application/octet-stream", nil
}

func (m *MockRunalyzeClient) GetTcx(id string) ([]byte, string, error) {
	if m.TcxError != nil {
		return nil, "", m.TcxError
	}
	return m.TcxData, "application/xml", nil
}

func (m *MockRunalyzeClient) GetDataBrowser(date time.Time) ([]byte, error) {
	if m.GetDataBrowserFunc != nil {
		return m.GetDataBrowserFunc(date)
	}
	return []byte("<html>test</html>"), m.BrowserError
}

func (m *MockRunalyzeClient) Login() error {
	m.LoginCalled = true
	return m.LoginError
}

func (m *MockRunalyzeClient) PersistCookies() error {
	m.PersistCalled = true
	return nil
}

// MockFileSystem implements FileSystem for testing
type MockFileSystem struct {
	Files      map[string][]byte
	WriteError error
	MkdirError error
	WriteCalls []WriteCall
	MkdirCalls []string
}

type WriteCall struct {
	Path string
	Data []byte
	Perm int
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files: make(map[string][]byte),
	}
}

func (m *MockFileSystem) WriteFile(path string, data []byte, perm int) error {
	m.WriteCalls = append(m.WriteCalls, WriteCall{Path: path, Data: data, Perm: perm})
	if m.WriteError != nil {
		return m.WriteError
	}
	m.Files[path] = data
	return nil
}

func (m *MockFileSystem) Exists(path string) bool {
	_, exists := m.Files[path]
	return exists
}

func (m *MockFileSystem) MkdirAll(path string, perm int) error {
	m.MkdirCalls = append(m.MkdirCalls, path)
	return m.MkdirError
}

// MockLogger implements Logger for testing
type MockLogger struct {
	InfoCalls  []LogCall
	DebugCalls []LogCall
	WarnCalls  []LogCall
}

type LogCall struct {
	Message string
	Args    []any
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.InfoCalls = append(m.InfoCalls, LogCall{Message: msg, Args: args})
}

func (m *MockLogger) Debug(msg string, args ...any) {
	m.DebugCalls = append(m.DebugCalls, LogCall{Message: msg, Args: args})
}

func (m *MockLogger) Warn(msg string, args ...any) {
	m.WarnCalls = append(m.WarnCalls, LogCall{Message: msg, Args: args})
}

// Helper function to create a not found error (404)
func createNotFoundError() error {
	return fmt.Errorf("unexpected status code: 404")
}
