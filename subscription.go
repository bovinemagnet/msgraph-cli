package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bovinemagnet/msgraph-cli/graphhelper"
)

func (a *App) HandleListSubscriptions() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.output.Clear()
	fmt.Fprintf(a.output, "Listing subscriptions...\n")

	subscriptions, err := a.graphHelper.ListSubscriptions(context.Background())
	if err != nil {
		fmt.Fprintf(a.output, "[red]Error making Graph call: %v[white]\n", err)
		return
	}

	if subscriptions == nil {
		fmt.Fprintf(a.output, "[yellow]No subscriptions found (response was nil)[white]\n")
		return
	}

	values := subscriptions.GetValue()
	if values == nil {
		fmt.Fprintf(a.output, "[yellow]No subscription values found (GetValue() returned nil)[white]\n")
		return
	}

	if len(values) == 0 {
		fmt.Fprintf(a.output, "[yellow]No active subscriptions found[white]\n")
		return
	}

	fmt.Fprintf(a.output, "[green]Found %d subscription(s):[white]\n", len(values))
	for _, subscription := range values {
		fmt.Fprintf(a.output, "\n[yellow]SubscriptionId: [green]%s[white]\n", *subscription.GetId())
		fmt.Fprintf(a.output, "  ChangeType: %s\n", *subscription.GetChangeType())
		fmt.Fprintf(a.output, "  ExpirationDateTime: %s\n", subscription.GetExpirationDateTime().Format(time.RFC3339))
		fmt.Fprintf(a.output, "  Resource: %s\n", *subscription.GetResource())
		fmt.Fprintf(a.output, "  NotificationURL: %s\n", *subscription.GetNotificationUrl())
	}
}

func listSubscriptions(graphHelper *graphhelper.GraphHelper) {

	subscriptions, err := graphHelper.ListSubscriptions(context.Background())
	if err != nil {
		log.Panicf("Error making Graph call: %v", err)
	}

	// check for nil size on the subscriptions
	if subscriptions == nil {
		fmt.Println("No subscriptions found")
		return
	}

	for _, subscription := range subscriptions.GetValue() {
		fmt.Printf("SubscriptionId: %s\n", *subscription.GetId())
		fmt.Printf("  ChangeType: %s\n", *subscription.GetChangeType())
		fmt.Printf("  ExpirationDateTime: %s\n", subscription.GetExpirationDateTime().String())
		fmt.Printf("  Resource: %s\n", *subscription.GetResource())
		fmt.Printf("  ApplicationId: %s\n", *subscription.GetApplicationId())
		// print the additional data
		fmt.Printf("  Additional Data length: %v\n", len(subscription.GetAdditionalData()))
		//fmt.Printf("  LifecycleNotificationURL: %v\n", *subscription.GetLifecycleNotificationUrl())
		//fmt.Printf("  ClientState: %s\n", *subscription.GetClientState())
		fmt.Printf("  CreatorId: %v\n", *subscription.GetCreatorId())
		fmt.Printf("  NotificationURL: %v\n", *subscription.GetNotificationUrl())

		fmt.Println()

	}
}
