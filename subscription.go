package main

import (
	"fmt"
	"log"

	"github.com/bovinemagnet/msgraph-cli/graphhelper"
)

func (a *App) HandleListSubscriptions() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.output.Clear()
	fmt.Fprintf(a.output, "Listing subscriptions...\n")

	subscriptions, err := a.graphHelper.ListSubscriptions()
	if err != nil {
		fmt.Fprintf(a.output, "[red]Error making Graph call: %v[white]\n", err)
		return
	}

	if subscriptions == nil {
		fmt.Fprintf(a.output, "No subscriptions found\n")
		return
	}

	if len(subscriptions.GetValue()) == 0 {
		fmt.Fprintf(a.output, "No subscriptions found\n")
		return
	}

	for _, subscription := range subscriptions.GetValue() {
		fmt.Fprintf(a.output, "[yellow]SubscriptionId:[white] %s\n", *subscription.GetId())
		fmt.Fprintf(a.output, "  ChangeType: %s\n", *subscription.GetChangeType())
		fmt.Fprintf(a.output, "  ExpirationDateTime: %s\n", subscription.GetExpirationDateTime().String())
		fmt.Fprintf(a.output, "  Resource: %s\n", *subscription.GetResource())
		fmt.Fprintf(a.output, "  ApplicationId: %s\n", *subscription.GetApplicationId())
		fmt.Fprintf(a.output, "  Additional Data length: %v\n", len(subscription.GetAdditionalData()))
		fmt.Fprintf(a.output, "  CreatorId: %v\n", *subscription.GetCreatorId())
		fmt.Fprintf(a.output, "  NotificationURL: %v\n\n", *subscription.GetNotificationUrl())
	}

}

func listSubscriptions(graphHelper *graphhelper.GraphHelper) {

	subscriptions, err := graphHelper.ListSubscriptions()
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
