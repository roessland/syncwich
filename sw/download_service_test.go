package sw

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestDownloadActivity_HappyPath_FIT(t *testing.T) {
	// Arrange
	mockClient := &MockRunalyzeClient{
		FitData: []byte("fake fit data"),
	}
	mockFS := NewMockFileSystem()
	mockLogger := &MockLogger{}
	service := NewDownloadService(mockClient, mockFS, mockLogger)

	activity := ActivityInfo{ID: "12345", Type: "running"}
	saveDir := "/tmp/activities"

	// Act
	result := service.DownloadActivity(activity, saveDir)

	// Assert
	if !result.Success {
		t.Errorf("Expected success, got failure: %v", result.Error)
	}
	if result.FileType != "FIT" {
		t.Errorf("Expected FIT, got %s", result.FileType)
	}
	if result.Existed {
		t.Error("Expected new download, got existing file")
	}

	expectedPath := filepath.Join(saveDir, "12345.fit")
	if result.FilePath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, result.FilePath)
	}

	// Verify file was written
	if len(mockFS.WriteCalls) != 1 {
		t.Errorf("Expected 1 write call, got %d", len(mockFS.WriteCalls))
	}
	if string(mockFS.WriteCalls[0].Data) != "fake fit data" {
		t.Errorf("Expected fit data to be written")
	}
}

func TestDownloadActivity_FallbackToTCX(t *testing.T) {
	// Arrange - FIT fails with 404, TCX succeeds
	mockClient := &MockRunalyzeClient{
		FitError: createNotFoundError(),
		TcxData:  []byte("fake tcx data"),
	}
	mockFS := NewMockFileSystem()
	mockLogger := &MockLogger{}
	service := NewDownloadService(mockClient, mockFS, mockLogger)

	activity := ActivityInfo{ID: "12345", Type: "cycling"}
	saveDir := "/tmp/activities"

	// Act
	result := service.DownloadActivity(activity, saveDir)

	// Assert
	if !result.Success {
		t.Errorf("Expected success, got failure: %v", result.Error)
	}
	if result.FileType != "TCX" {
		t.Errorf("Expected TCX, got %s", result.FileType)
	}

	expectedPath := filepath.Join(saveDir, "12345.tcx")
	if result.FilePath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, result.FilePath)
	}

	// Verify TCX file was written
	if len(mockFS.WriteCalls) != 1 {
		t.Errorf("Expected 1 write call, got %d", len(mockFS.WriteCalls))
	}
	if string(mockFS.WriteCalls[0].Data) != "fake tcx data" {
		t.Errorf("Expected tcx data to be written")
	}
}

func TestDownloadActivity_FileAlreadyExists_FIT(t *testing.T) {
	// Arrange - FIT file already exists
	mockClient := &MockRunalyzeClient{}
	mockFS := NewMockFileSystem()
	mockLogger := &MockLogger{}

	// Pre-populate with existing FIT file
	existingPath := filepath.Join("/tmp/activities", "12345.fit")
	mockFS.Files[existingPath] = []byte("existing data")

	service := NewDownloadService(mockClient, mockFS, mockLogger)
	activity := ActivityInfo{ID: "12345", Type: "running"}

	// Act
	result := service.DownloadActivity(activity, "/tmp/activities")

	// Assert
	if !result.Success {
		t.Errorf("Expected success for existing file, got failure: %v", result.Error)
	}
	if !result.Existed {
		t.Error("Expected existing file flag to be true")
	}
	if result.FileType != "FIT" {
		t.Errorf("Expected FIT, got %s", result.FileType)
	}

	// Verify no network calls were made
	if len(mockFS.WriteCalls) != 0 {
		t.Errorf("Expected no write calls for existing file, got %d", len(mockFS.WriteCalls))
	}
}

func TestDownloadActivity_FileAlreadyExists_TCX(t *testing.T) {
	// Arrange - TCX file already exists
	mockClient := &MockRunalyzeClient{}
	mockFS := NewMockFileSystem()
	mockLogger := &MockLogger{}

	// Pre-populate with existing TCX file
	existingPath := filepath.Join("/tmp/activities", "12345.tcx")
	mockFS.Files[existingPath] = []byte("existing tcx data")

	service := NewDownloadService(mockClient, mockFS, mockLogger)
	activity := ActivityInfo{ID: "12345", Type: "running"}

	// Act
	result := service.DownloadActivity(activity, "/tmp/activities")

	// Assert
	if !result.Success {
		t.Errorf("Expected success for existing file, got failure: %v", result.Error)
	}
	if !result.Existed {
		t.Error("Expected existing file flag to be true")
	}
	if result.FileType != "TCX" {
		t.Errorf("Expected TCX, got %s", result.FileType)
	}
}

func TestDownloadActivity_NeitherFormatAvailable(t *testing.T) {
	// Arrange - Both FIT and TCX return 404
	mockClient := &MockRunalyzeClient{
		FitError: createNotFoundError(),
		TcxError: createNotFoundError(),
	}
	mockFS := NewMockFileSystem()
	mockLogger := &MockLogger{}
	service := NewDownloadService(mockClient, mockFS, mockLogger)

	activity := ActivityInfo{ID: "12345", Type: "manual"}

	// Act
	result := service.DownloadActivity(activity, "/tmp/activities")

	// Assert
	if result.Success {
		t.Error("Expected failure when neither format available")
	}
	if result.FileType != "NONE" {
		t.Errorf("Expected NONE, got %s", result.FileType)
	}
	if result.Error == nil {
		t.Error("Expected error when neither format available")
	}

	// Verify no files were written
	if len(mockFS.WriteCalls) != 0 {
		t.Errorf("Expected no write calls, got %d", len(mockFS.WriteCalls))
	}
}

func TestDownloadActivity_FITDownloadError(t *testing.T) {
	// Arrange - FIT download fails with non-404 error
	mockClient := &MockRunalyzeClient{
		FitError: fmt.Errorf("network timeout"),
	}
	mockFS := NewMockFileSystem()
	mockLogger := &MockLogger{}
	service := NewDownloadService(mockClient, mockFS, mockLogger)

	activity := ActivityInfo{ID: "12345", Type: "running"}

	// Act
	result := service.DownloadActivity(activity, "/tmp/activities")

	// Assert
	if result.Success {
		t.Error("Expected failure on network error")
	}
	if result.FileType != "FIT" {
		t.Errorf("Expected FIT, got %s", result.FileType)
	}
	if result.Error == nil {
		t.Error("Expected error on network failure")
	}
}

func TestDownloadActivity_FileSaveError(t *testing.T) {
	// Arrange - Download succeeds but file save fails
	mockClient := &MockRunalyzeClient{
		FitData: []byte("fake fit data"),
	}
	mockFS := NewMockFileSystem()
	mockFS.WriteError = fmt.Errorf("disk full")
	mockLogger := &MockLogger{}
	service := NewDownloadService(mockClient, mockFS, mockLogger)

	activity := ActivityInfo{ID: "12345", Type: "running"}

	// Act
	result := service.DownloadActivity(activity, "/tmp/activities")

	// Assert
	if result.Success {
		t.Error("Expected failure on file save error")
	}
	if result.FileType != "FIT" {
		t.Errorf("Expected FIT, got %s", result.FileType)
	}
	if result.Error == nil {
		t.Error("Expected error on file save failure")
	}
}
