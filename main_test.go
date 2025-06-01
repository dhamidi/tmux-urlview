package main

import (
	"errors"
	"testing"
)

// Fake implementations for testing

type fakeInputProvider struct {
	input []byte
	err   error
}

func (f *fakeInputProvider) GetInput() ([]byte, error) {
	return f.input, f.err
}

type fakeCommandRunner struct {
	output []byte
	err    error
	lastCommand string
	lastArgs    []string
}

func (f *fakeCommandRunner) RunCommand(name string, args ...string) ([]byte, error) {
	f.lastCommand = name
	f.lastArgs = args
	return f.output, f.err
}

type fakeURLSelector struct {
	selectedURL string
	err         error
}

func (f *fakeURLSelector) SelectURL(urls []string) (string, error) {
	return f.selectedURL, f.err
}

type fakeURLOpener struct {
	openedURLs []string
	err        error
}

func (f *fakeURLOpener) OpenURL(url string) error {
	f.openedURLs = append(f.openedURLs, url)
	return f.err
}

type fakeEnvironment struct {
	envVars    map[string]string
	isStdinTTY bool
}

func (f *fakeEnvironment) GetEnv(key string) string {
	return f.envVars[key]
}

func (f *fakeEnvironment) IsStdinTTY() bool {
	return f.isStdinTTY
}

// Tests for extractURLs function

func TestExtractURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "no URLs",
			input:    "This is just plain text with no URLs",
			expected: []string{},
		},
		{
			name:     "single HTTP URL",
			input:    "Check out https://example.com for more info",
			expected: []string{"https://example.com"},
		},
		{
			name:     "single HTTPS URL",
			input:    "Visit https://secure.example.com/path?param=value",
			expected: []string{"https://secure.example.com/path?param=value"},
		},
		{
			name:     "multiple URLs",
			input:    "See https://example.com and http://test.org",
			expected: []string{"https://example.com", "http://test.org"},
		},
		{
			name:     "URL with punctuation at end",
			input:    "Visit https://example.com. Also check https://test.org!",
			expected: []string{"https://example.com", "https://test.org"},
		},
		{
			name:     "duplicate URLs",
			input:    "https://example.com and https://example.com again",
			expected: []string{"https://example.com"},
		},
		{
			name:     "URL in multiline text",
			input:    "Line 1\nhttps://example.com\nLine 3",
			expected: []string{"https://example.com"},
		},
		{
			name:     "mixed valid and invalid URLs",
			input:    "Valid: https://example.com Invalid: https://",
			expected: []string{"https://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractURLs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d URLs, got %d", len(tt.expected), len(result))
				return
			}
			for i, url := range result {
				if url != tt.expected[i] {
					t.Errorf("expected URL %q, got %q", tt.expected[i], url)
				}
			}
		})
	}
}

// Tests for run function

func TestRun_NoURLsFound(t *testing.T) {
	inputProvider := &fakeInputProvider{
		input: []byte("This text has no URLs"),
		err:   nil,
	}
	urlSelector := &fakeURLSelector{}
	urlOpener := &fakeURLOpener{}

	err := run(inputProvider, urlSelector, urlOpener)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(urlOpener.openedURLs) != 0 {
		t.Errorf("expected no URLs to be opened, got %v", urlOpener.openedURLs)
	}
}

func TestRun_InputError(t *testing.T) {
	inputProvider := &fakeInputProvider{
		input: nil,
		err:   errors.New("input error"),
	}
	urlSelector := &fakeURLSelector{}
	urlOpener := &fakeURLOpener{}

	err := run(inputProvider, urlSelector, urlOpener)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Error() != "error reading input: input error" {
		t.Errorf("expected specific error message, got %v", err)
	}
}

func TestRun_URLSelectionError(t *testing.T) {
	inputProvider := &fakeInputProvider{
		input: []byte("Visit https://example.com"),
		err:   nil,
	}
	urlSelector := &fakeURLSelector{
		selectedURL: "",
		err:         errors.New("selection error"),
	}
	urlOpener := &fakeURLOpener{}

	err := run(inputProvider, urlSelector, urlOpener)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Error() != "error selecting URL: selection error" {
		t.Errorf("expected specific error message, got %v", err)
	}
}

func TestRun_URLOpeningError(t *testing.T) {
	inputProvider := &fakeInputProvider{
		input: []byte("Visit https://example.com"),
		err:   nil,
	}
	urlSelector := &fakeURLSelector{
		selectedURL: "https://example.com",
		err:         nil,
	}
	urlOpener := &fakeURLOpener{
		err: errors.New("open error"),
	}

	err := run(inputProvider, urlSelector, urlOpener)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Error() != "error opening URL: open error" {
		t.Errorf("expected specific error message, got %v", err)
	}
}

func TestRun_UserCancelsSelection(t *testing.T) {
	inputProvider := &fakeInputProvider{
		input: []byte("Visit https://example.com"),
		err:   nil,
	}
	urlSelector := &fakeURLSelector{
		selectedURL: "", // Empty string indicates cancellation
		err:         nil,
	}
	urlOpener := &fakeURLOpener{}

	err := run(inputProvider, urlSelector, urlOpener)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(urlOpener.openedURLs) != 0 {
		t.Errorf("expected no URLs to be opened, got %v", urlOpener.openedURLs)
	}
}

func TestRun_SuccessfulFlow(t *testing.T) {
	inputProvider := &fakeInputProvider{
		input: []byte("Visit https://example.com and https://test.org"),
		err:   nil,
	}
	urlSelector := &fakeURLSelector{
		selectedURL: "https://example.com",
		err:         nil,
	}
	urlOpener := &fakeURLOpener{}

	err := run(inputProvider, urlSelector, urlOpener)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(urlOpener.openedURLs) != 1 {
		t.Errorf("expected 1 URL to be opened, got %d", len(urlOpener.openedURLs))
	}
	if urlOpener.openedURLs[0] != "https://example.com" {
		t.Errorf("expected https://example.com to be opened, got %s", urlOpener.openedURLs[0])
	}
}

// Tests for fake implementations behavior

func TestInputProvider_WithTmuxPane(t *testing.T) {
	env := &fakeEnvironment{
		envVars: map[string]string{
			"TMUX_PANE": "%1",
		},
		isStdinTTY: true,
	}
	
	cmdRunner := &fakeCommandRunner{
		output: []byte("tmux pane content with https://example.com"),
		err:    nil,
	}
	
	provider := &realInputProvider{env: env, cmdRunner: cmdRunner}
	
	input, err := provider.GetInput()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if string(input) != "tmux pane content with https://example.com" {
		t.Errorf("expected tmux content, got %s", string(input))
	}
	if cmdRunner.lastCommand != "tmux" {
		t.Errorf("expected tmux command, got %s", cmdRunner.lastCommand)
	}
	expectedArgs := []string{"capture-pane", "-p", "-t", "%1"}
	if len(cmdRunner.lastArgs) != len(expectedArgs) {
		t.Errorf("expected %d args, got %d", len(expectedArgs), len(cmdRunner.lastArgs))
	}
	for i, arg := range cmdRunner.lastArgs {
		if arg != expectedArgs[i] {
			t.Errorf("expected arg %s, got %s", expectedArgs[i], arg)
		}
	}
}

func TestInputProvider_WithTmuxPaneError(t *testing.T) {
	env := &fakeEnvironment{
		envVars: map[string]string{
			"TMUX_PANE": "%1",
		},
		isStdinTTY: true,
	}
	
	cmdRunner := &fakeCommandRunner{
		output: nil,
		err:    errors.New("tmux command failed"),
	}
	
	provider := &realInputProvider{env: env, cmdRunner: cmdRunner}
	
	_, err := provider.GetInput()
	if err == nil {
		t.Error("expected error when tmux command fails, got nil")
	}
	if err.Error() != "tmux command failed" {
		t.Errorf("expected tmux error, got %v", err)
	}
}

func TestInputProvider_NoTmuxPane(t *testing.T) {
	env := &fakeEnvironment{
		envVars:    map[string]string{},
		isStdinTTY: false,
	}
	
	cmdRunner := &fakeCommandRunner{}
	
	provider := &realInputProvider{env: env, cmdRunner: cmdRunner}
	
	// This will try to read from os.Stdin, which will be empty in tests
	// but we're testing that the tmux path is not taken
	_, _ = provider.GetInput()
	
	if cmdRunner.lastCommand != "" {
		t.Errorf("expected no command to be run, but %s was called", cmdRunner.lastCommand)
	}
}

func TestFakeEnvironment(t *testing.T) {
	env := &fakeEnvironment{
		envVars: map[string]string{
			"TEST_VAR": "test_value",
		},
		isStdinTTY: true,
	}

	if env.GetEnv("TEST_VAR") != "test_value" {
		t.Errorf("expected test_value, got %s", env.GetEnv("TEST_VAR"))
	}
	if env.GetEnv("NONEXISTENT") != "" {
		t.Errorf("expected empty string for nonexistent var, got %s", env.GetEnv("NONEXISTENT"))
	}
	if !env.IsStdinTTY() {
		t.Error("expected IsStdinTTY to return true")
	}
}
