package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/bovinemagnet/msgraph-cli/graphhelper"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("Go MS Graph App-Only Simple CLI Tool")
	fmt.Println()

	// Load .env files
	// .env.local takes precedence (if present)
	godotenv.Load(".env.local")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env")
	}

	// Set up app auth
	graphHelper := graphhelper.NewGraphHelper()

	initializeGraph(graphHelper)

	// Start up a simple the webserver for the subscription messages on the port in the .env file.
	go func() {
		port := graphHelper.GetPort()
		http.HandleFunc("/webhook", handleGraphSubscription)
		log.Println("Server starting... [port: " + port + "]")
		if err := http.ListenAndServe(port, nil); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// get the organiser and room email from the environment.
	organiserEmail := graphHelper.GetOrganiserEmail()
	if organiserEmail == "" {
		fmt.Println("No organiser found")
	}

	roomEmail := graphHelper.GetRoomEmail()
	if roomEmail == "" {
		fmt.Println("No room email found")
	}

	var choice int64 = -1

	for {
		fmt.Printf("\n\nPlease choose one of the following options:\n")
		fmt.Println("  0.  Exit")
		fmt.Println("  1.  Display access token")
		fmt.Println("  +-----------------------------------+")
		fmt.Println("  2.  List All Users")
		fmt.Println("  3.  List All Subscriptions")
		fmt.Println("  4.  List All Rooms")
		fmt.Println("  5.  List 7 days of Events - By Room [" + roomEmail + "]")
		fmt.Println("  6.  List 7 days of Events - By Organiser [" + organiserEmail + "]")
		fmt.Println("  +-----------------------------------+")
		fmt.Println("  7.  Create a 1 day subscription - By Room [" + roomEmail + "]")
		fmt.Println("  8.  Delete a subscription by the subscription id")
		fmt.Println("  +-----------------------------------+")
		fmt.Println("  9.  Delete event id - By Room [" + roomEmail + "]")
		fmt.Println("  10. Delete event id - By Organiser [" + organiserEmail + "]")
		fmt.Println("  +-----------------------------------+")
		fmt.Println("  11. Create Event at 10 to 10:30 tomorrow - By Organiser [" + organiserEmail + "]")
		fmt.Print(":> ")

		_, err = fmt.Scanf("%d", &choice)
		if err != nil {
			choice = -1
		}

		switch choice {
		case 0:
			// Exit the program
			fmt.Println("Goodbye...")
		case 1:
			// Display access token
			displayAccessToken(graphHelper)
		case 2:
			// List users
			listUsers(graphHelper)
		case 3:
			// List Subscriptions
			listSubscriptions(graphHelper)
		case 4:
			// list rooms
			listRooms(graphHelper)
		case 5:
			// list rooms
			listRoomBookingsAsRoom(graphHelper)
		case 6:
			// list rooms
			listRoomBookingsAsOrganiser(graphHelper)
		case 7:
			// create 1 day subscription
			createOneDaySubscription(graphHelper)
		case 8:
			// delete subscription by subscription id asked for as input
			deleteSubscription(graphHelper)
		case 9:
			// delete event by event id for the specified room//
			deleteEventByRoom(graphHelper)
		case 10:
			// delete event by event id for the specified organiser
			deleteEventByOrganiser(graphHelper)
		case 11:
			// delete event by event id for the specified organiser
			createEventByOrganiser(graphHelper)
		default:
			fmt.Println("Invalid choice! Please try again.")
		}

		if choice == 0 {
			break
		}
	}
}

func initializeGraph(graphHelper *graphhelper.GraphHelper) {
	err := graphHelper.InitializeGraphForAppAuth()
	if err != nil {
		log.Panicf("Error initializing Graph for app auth: %v\n", err)
	}
}

func displayAccessToken(graphHelper *graphhelper.GraphHelper) {
	token, err := graphHelper.GetAppToken()
	if err != nil {
		log.Panicf("Error getting user token: %v\n", err)
	}

	fmt.Printf("App-only token: %s", *token)
	fmt.Println()
}

func listUsers(graphHelper *graphhelper.GraphHelper) {
	users, err := graphHelper.GetUsers()
	if err != nil {
		log.Panicf("Error getting users: %v", err)
	}

	// Output each user's details
	for _, user := range users.GetValue() {
		fmt.Printf("User: %s\n", *user.GetDisplayName())
		fmt.Printf("  ID: %s\n", *user.GetId())

		noEmail := "NO EMAIL"
		email := user.GetMail()
		if email == nil {
			email = &noEmail
		}
		fmt.Printf("  Email: %s\n", *email)
	}

	// If GetOdataNextLink does not return nil,
	// there are more users available on the server
	nextLink := users.GetOdataNextLink()

	fmt.Println()
	fmt.Printf("More users available? %t\n", nextLink != nil)
	fmt.Println()
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

func listRooms(graphHelper *graphhelper.GraphHelper) {

	graphHelper.ListRooms()

}

func listRoomBookingsAsOrganiser(graphHelper *graphhelper.GraphHelper) {

	organiser := graphHelper.GetOrganiserEmail()
	if organiser == "" {
		fmt.Println("No organiser found")
		return
	}

	graphHelper.ListRoom7DaysBookings(organiser)

}

func listRoomBookingsAsRoom(graphHelper *graphhelper.GraphHelper) {

	roomEmail := graphHelper.GetRoomEmail()
	if roomEmail == "" {
		fmt.Println("No room email found")
		return
	}

	graphHelper.ListRoom7DaysBookings(roomEmail)

}

func handleGraphSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Check if this is a validation request
	if r.URL.Query().Get("validationToken") != "" {
		validationToken := r.URL.Query().Get("validationToken")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validationToken))
		log.Println("Validation token sent back to Microsoft Graph:", validationToken)
		return
	}

	// If not a validation request, this is likely an event notification
	log.Printf("Received notification: %s", string(body))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Notification received"))
}

func createOneDaySubscription(graphHelper *graphhelper.GraphHelper) {
	roomEmail := graphHelper.GetRoomEmail()
	if roomEmail == "" {
		fmt.Println("No room email found")
		return
	}

	values := graphHelper.CreateRoomSubscription(roomEmail)
	println(values)
}

func deleteSubscription(graphHelper *graphhelper.GraphHelper) {

	// As user to input the subscription id to delete
	var subscriptionId string
	fmt.Println("Enter the subscription id to delete")
	_, err := fmt.Scanf("%s", &subscriptionId)
	if err != nil {
		log.Printf("Error reading subscription id: %v", err)
		return
	}
	// now deleteSubscription
	err = graphHelper.DeleteSubscription(subscriptionId)
	if err != nil {
		log.Printf("Error deleting subscription: %v", err)
		return
	}
}

func deleteEventByOrganiser(graphHelper *graphhelper.GraphHelper) {

	organiser := graphHelper.GetOrganiserEmail()
	if organiser == "" {
		fmt.Println("No organiser found")
		return
	}

	var eventId string
	fmt.Println("Enter the event id to cancel:")
	_, err := fmt.Scanf("%s", &eventId)
	if err != nil {
		log.Printf("Error reading event id: %v", err)
		return
	}
	err = graphHelper.DeleteEvent(organiser, eventId)
	if err != nil {
		log.Printf("Error canceling event: %v", err)
		return
	}
}

func deleteEventByRoom(graphHelper *graphhelper.GraphHelper) {

	var eventId string
	fmt.Println("Enter the event id to cancel:")
	_, err := fmt.Scanf("%s", &eventId)
	if err != nil {
		log.Printf("Error reading event id: %v", err)
		return
	}

	roomEmail := graphHelper.GetRoomEmail()
	if roomEmail == "" {
		fmt.Println("No room email found")
		return
	}
	err = graphHelper.DeleteEvent(roomEmail, eventId)
	if err != nil {
		log.Printf("Error canceling event: %v", err)
		return
	}
}

func createEventByOrganiser(graphHelper *graphhelper.GraphHelper) {

	organiser := graphHelper.GetOrganiserEmail()
	if organiser == "" {
		fmt.Println("No organiser found")
		return
	}
	roomEmail := graphHelper.GetRoomEmail()
	if roomEmail == "" {
		fmt.Println("No room email found")
		return
	}

	err := graphHelper.CreateEvent(organiser, roomEmail)
	if err != nil {
		log.Printf("Error creating event: %v", err)
		return
	}

}
