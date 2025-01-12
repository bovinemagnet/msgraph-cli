package main

import (
	"fmt"
	"os"
	"strings"
)

func (a *App) handleEnv() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.output.Clear()
	fmt.Fprintf(a.output, "Showing the contents of the .env file...\n\n")

	envContents, err := os.ReadFile(".env")
	if err != nil {
		fmt.Fprintf(a.output, "[red]Error reading .env file: %v\n", err)
		return
	}

	// Process each line with colors
	lines := strings.Split(string(envContents), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			fmt.Fprintf(a.output, "[darkgray]%s[white]\n", line)
		} else {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				fmt.Fprintf(a.output, "[yellow]%s[white]=[white]%s\n", key, value)
			} else {
				fmt.Fprintf(a.output, "%s\n", line)
			}
		}
	}
}
