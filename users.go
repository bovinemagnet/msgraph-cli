package main

import (
	"fmt"
)

func (a *App) handleListUsers() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.output.Clear()
	fmt.Fprintf(a.output, "Listing users...\n")

	a.graphHelper.ListUsers(a.output)
}
