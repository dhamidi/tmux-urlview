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

func main() {
	var input []byte
	var err error
	
	// Check if stdin is a tty and TMUX_PANE is set
	if isStdinTTY() && os.Getenv("TMUX_PANE") != "" {
		tmuxPane := os.Getenv("TMUX_PANE")
		cmd := exec.Command("tmux", "capture-pane", "-p", "-t", tmuxPane)
		input, err = cmd.Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error capturing tmux pane: %v\n", err)
			os.Exit(1)
		}
	} else {
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
	}

	urls := extractURLs(string(input))
	
	if len(urls) == 0 {
		return
	}

	cmd := exec.Command("fzf")
	cmd.Stdin = strings.NewReader(strings.Join(urls, "\n"))
	cmd.Stderr = os.Stderr
	
	output, err := cmd.Output()
	if err != nil {
		// fzf exits with code 1 when cancelled or 130 when interrupted, don't treat as error
		exitCode := cmd.ProcessState.ExitCode()
		if exitCode == 1 || exitCode == 130 {
			return
		}
		fmt.Fprintf(os.Stderr, "Error running fzf: %v\n", err)
		os.Exit(1)
	}
	
	selectedURL := strings.TrimSpace(string(output))
	if selectedURL != "" {
		openCmd := exec.Command("open", selectedURL)
		if err := openCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error opening URL: %v\n", err)
			os.Exit(1)
		}
	}
}

func isStdinTTY() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func extractURLs(text string) []string {
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	matches := urlRegex.FindAllString(text, -1)
	
	var validURLs []string
	seen := make(map[string]bool)
	
	for _, match := range matches {
		cleaned := strings.TrimRight(match, ".,;!?")
		
		if _, err := url.Parse(cleaned); err == nil {
			if !seen[cleaned] {
				validURLs = append(validURLs, cleaned)
				seen[cleaned] = true
			}
		}
	}
	
	return validURLs
}
