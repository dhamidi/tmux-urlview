package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type InputProvider interface {
	GetInput() ([]byte, error)
}

type URLSelector interface {
	SelectURL(urls []string) (string, error)
}

type URLOpener interface {
	OpenURL(url string) error
}

type Environment interface {
	GetEnv(key string) string
	IsStdinTTY() bool
}

type CommandRunner interface {
	RunCommand(name string, args ...string) ([]byte, error)
}

type realInputProvider struct {
	env    Environment
	cmdRunner CommandRunner
}

func (r *realInputProvider) GetInput() ([]byte, error) {
	if r.env.IsStdinTTY() && r.env.GetEnv("TMUX_PANE") != "" {
		tmuxPane := r.env.GetEnv("TMUX_PANE")
		return r.cmdRunner.RunCommand("tmux", "capture-pane", "-p", "-t", tmuxPane)
	}
	return io.ReadAll(os.Stdin)
}

type realURLSelector struct{}

func (r *realURLSelector) SelectURL(urls []string) (string, error) {
	cmd := exec.Command("fzf")
	cmd.Stdin = strings.NewReader(strings.Join(urls, "\n"))
	cmd.Stderr = os.Stderr
	
	output, err := cmd.Output()
	if err != nil {
		exitCode := cmd.ProcessState.ExitCode()
		if exitCode == 1 || exitCode == 130 {
			return "", nil
		}
		return "", err
	}
	
	return strings.TrimSpace(string(output)), nil
}

type realURLOpener struct{}

func (r *realURLOpener) OpenURL(url string) error {
	cmd := exec.Command("open", url)
	return cmd.Run()
}

type realEnvironment struct{}

func (r *realEnvironment) GetEnv(key string) string {
	return os.Getenv(key)
}

func (r *realEnvironment) IsStdinTTY() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

type realCommandRunner struct{}

func (r *realCommandRunner) RunCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

func run(inputProvider InputProvider, urlSelector URLSelector, urlOpener URLOpener) error {
	input, err := inputProvider.GetInput()
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}

	urls := extractURLs(string(input))
	
	if len(urls) == 0 {
		return nil
	}

	selectedURL, err := urlSelector.SelectURL(urls)
	if err != nil {
		return fmt.Errorf("error selecting URL: %v", err)
	}
	
	if selectedURL != "" {
		if err := urlOpener.OpenURL(selectedURL); err != nil {
			return fmt.Errorf("error opening URL: %v", err)
		}
	}
	
	return nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		fmt.Println("bind U display-popup -E 'tmux capture-pane -p | tmux-urlview'")
		return
	}
	
	env := &realEnvironment{}
	cmdRunner := &realCommandRunner{}
	inputProvider := &realInputProvider{env: env, cmdRunner: cmdRunner}
	urlSelector := &realURLSelector{}
	urlOpener := &realURLOpener{}
	
	if err := run(inputProvider, urlSelector, urlOpener); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func extractURLs(text string) []string {
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	matches := urlRegex.FindAllString(text, -1)
	
	var validURLs []string
	seen := make(map[string]bool)
	
	for _, match := range matches {
		cleaned := strings.TrimRight(match, ".,;!?)(]}")
		
		if _, err := url.Parse(cleaned); err == nil {
			if !seen[cleaned] {
				validURLs = append(validURLs, cleaned)
				seen[cleaned] = true
			}
		}
	}
	
	return validURLs
}
