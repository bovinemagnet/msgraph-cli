package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// Webhook handler
func (a *App) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Handle validation token
	if validationToken := r.URL.Query().Get("validationToken"); validationToken != "" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validationToken))
		a.appendToWebhookOutput(fmt.Sprintf("[green]Validation token received:[white] %s\n", validationToken))
		return
	}

	// Handle webhook notification
	a.appendToWebhookOutput(fmt.Sprintf("[yellow]Webhook notification:[white] %s\n", string(body)))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Notification received"))
}
