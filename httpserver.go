package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type WebhookServer struct {
	app *App
}

func NewWebhookServer(app *App) *WebhookServer {
	return &WebhookServer{
		app: app,
	}
}

func (ws *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		ws.app.webhookChan <- fmt.Sprintf("[%s] Method not allowed: %s\n",
			time.Now().Format("15:04:05"), r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Handle validation request
	validationToken := r.URL.Query().Get("validationToken")
	if validationToken != "" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validationToken))
		ws.app.webhookChan <- fmt.Sprintf("[yellow][%s][white] Validation token sent: %s\n",
			time.Now().Format("15:04:05"), validationToken)
		return
	}

	// Handle notification
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ws.app.webhookChan <- fmt.Sprintf("[yellow][%s][white] Failed to read request body: %v\n",
			time.Now().Format("15:04:05"), err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	ws.app.webhookChan <- fmt.Sprintf("[yellow][%s][white] Received notification: %s\n",
		time.Now().Format("15:04:05"), string(body))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Notification received"))
}

func (ws *WebhookServer) Start() error {
	port := ws.app.graphHelper.GetPort()
	http.HandleFunc("/webhook", ws.handleWebhook)
	ws.app.webhookChan <- fmt.Sprintf("[yellow][%s][white] Webhook server starting on port %s...\n",
		time.Now().Format("15:04:05"), port)
	return http.ListenAndServe(port, nil)
}
