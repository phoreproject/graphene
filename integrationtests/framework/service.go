package testframework

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

// TestService is the test service
type TestService struct {
	entryArgs EntryArgList
	tempDir   string
}

// NewTestService create a new TestService object
func NewTestService() *TestService {
	return &TestService{}
}

// GetArgString return the arg string
func (service *TestService) GetArgString(name string, defaultValue string) string {
	p := service.entryArgs[name]
	if p == nil {
		return defaultValue
	}

	return *p.(*string)
}

// BindEntryArgs binds the entry args
func (service *TestService) BindEntryArgs(entryArgs EntryArgList) {
	service.entryArgs = entryArgs
}

// CleanUp acts as the destructor
func (service *TestService) CleanUp() {
	if service.tempDir != "" {
		os.RemoveAll(service.tempDir)
	}
}

// GetTempFileName returns temporary file name
func (service *TestService) GetTempFileName(prefix string, postfix string) string {
	dir := service.requireTempDir()
	i := 1
	for {
		fileName := filepath.Join(dir, prefix+strconv.Itoa(i)+postfix)
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			return fileName
		}
		i++
	}
}

func (service *TestService) requireTempDir() string {
	if service.tempDir != "" {
		service.tempDir, _ = ioutil.TempDir("", "synapse")
	}

	return service.tempDir
}
