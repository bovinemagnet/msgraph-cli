package graphhelper

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	auth "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	graphusers "github.com/microsoftgraph/msgraph-sdk-go/users"
)

type GraphHelper struct {
	clientSecretCredential *azidentity.ClientSecretCredential
	appClient              *msgraphsdk.GraphServiceClient
}

func NewGraphHelper() *GraphHelper {
	g := &GraphHelper{}
	return g
}

// GetPort retrieves the port number from the environment variable "PORT".
// If the "PORT" environment variable is not set, it logs a fatal error message
// and returns the default port ":8080".
func (g *GraphHelper) GetPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT is not set in .env file, returning default port 8080")
		port = "8080"
	}
	return ":" + port
}

// GetRoomEmail retrieves the room email address from the environment variable "ROOM_EMAIL".
// If the environment variable is not set, the function logs a fatal error and terminates the program.
// Returns the room email address as a string.
func (g *GraphHelper) GetRoomEmail() string {
	roomEmail := os.Getenv("ROOM_EMAIL")
	if roomEmail == "" {
		log.Fatal("ROOM_EMAIL is not set in .env file")
		return ""

	}
	return roomEmail
}

// GetOrganiserEmail retrieves the organizer's email address from the environment variable "ORGANISER_EMAIL".
// If the environment variable is not set, the function logs a fatal error and terminates the program.
// Returns the organizer's email address as a string.
func (g *GraphHelper) GetOrganiserEmail() string {
	organiserEmail := os.Getenv("ORGANISER_EMAIL")

	if organiserEmail == "" {
		log.Fatal("ORGANISER_EMAIL is not set in .env file")
		return ""
	}
	return organiserEmail
}

// GetNotificationUrl retrieves the notification URL from the environment variable "ENDPOINT".
// If the "ENDPOINT" variable is not set, the function logs a fatal error and terminates the program.
// Returns the notification URL as a string.
func (g *GraphHelper) GetNotificationUrl() string {
	notificationURL := os.Getenv("ENDPOINT")

	if notificationURL == "" {
		log.Fatal("ENDPOINT is not set in .env file")
		return ""
	}
	return notificationURL
}

// InitializeGraphForAppAuth initializes the Microsoft Graph client for application authentication.
// It retrieves the client ID, tenant ID, and client secret from environment variables,
// creates a client secret credential, and uses it to create an authentication provider.
// The authentication provider is then used to create a request adapter, which is used to
// create a Graph client. The initialized Graph client is stored in the GraphHelper struct.
//
// Returns an error if any of the steps fail.
func (g *GraphHelper) InitializeGraphForAppAuth() error {
	clientId := os.Getenv("CLIENT_ID")
	tenantId := os.Getenv("TENANT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	credential, err := azidentity.NewClientSecretCredential(tenantId, clientId, clientSecret, nil)
	if err != nil {
		return err
	}

	g.clientSecretCredential = credential

	// Create an auth provider using the credential
	authProvider, err := auth.NewAzureIdentityAuthenticationProviderWithScopes(g.clientSecretCredential, []string{
		"https://graph.microsoft.com/.default",
	})
	if err != nil {
		return err
	}

	// Create a request adapter using the auth provider
	adapter, err := msgraphsdk.NewGraphRequestAdapter(authProvider)
	if err != nil {
		return err
	}

	// Create a Graph client using request adapter
	client := msgraphsdk.NewGraphServiceClient(adapter)
	g.appClient = client

	return nil
}

// GetAppToken retrieves an application token using the client secret credential.
// It requests a token with the scope "https://graph.microsoft.com/.default".
// Returns a pointer to the token string if successful, or an error if the token request fails.
func (g *GraphHelper) GetAppToken() (*string, error) {
	token, err := g.clientSecretCredential.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{
			"https://graph.microsoft.com/.default",
		},
	})
	if err != nil {
		return nil, err
	}

	return &token.Token, nil
}

func (g *GraphHelper) GetUsers() (graphmodels.UserCollectionResponseable, error) {
	var topValue int32 = 25
	query := graphusers.UsersRequestBuilderGetQueryParameters{
		// Only request specific properties
		Select: []string{"displayName", "id", "mail"},
		// Get at most 25 results
		Top: &topValue,
		// Sort by display name
		Orderby: []string{"displayName"},
	}

	return g.appClient.Users().
		Get(context.Background(),
			&graphusers.UsersRequestBuilderGetRequestConfiguration{
				QueryParameters: &query,
			})
}

func (g *GraphHelper) ListSubscriptions() (graphmodels.SubscriptionCollectionResponseable, error) {

	return g.appClient.Subscriptions().
		Get(context.Background(), nil)

}

// ListRooms
func (g *GraphHelper) ListRooms() {
	// (places.GraphRoomRequestBuilder, error) {

	rooms, err := g.appClient.Places().GraphRoom().Get(context.Background(), nil)
	if err != nil {
		fmt.Println("Failed to list rooms:", err)
		return
	}

	for _, room := range rooms.GetValue() {
		fmt.Printf("Room ID: %s\n", *room.GetId())
		fmt.Printf("  Name: %s\n", *room.GetDisplayName())
		fmt.Printf("  Capacity: %d\n", *room.GetCapacity())
		fmt.Printf("  Email: %s\n", *room.GetEmailAddress())
	}

	return

}

func (g *GraphHelper) ListRoom7DaysBookings(roomId string) {
	now := time.Now()
	startDateTime := now.Format(time.RFC3339)
	endDateTime := now.Add(7 * 24 * time.Hour).Format(time.RFC3339) // Next 7 days for example

	// Query parameters for fetching calendar events
	queryParams := &graphusers.ItemCalendarViewRequestBuilderGetQueryParameters{
		EndDateTime:   &endDateTime,
		StartDateTime: &startDateTime,
	}

	// Configuration for the request
	requestConfig := &graphusers.ItemCalendarViewRequestBuilderGetRequestConfiguration{
		QueryParameters: queryParams,
	}

	// Get the calendar view of the room
	events, err := g.appClient.Users().ByUserId(roomId).CalendarView().Get(context.Background(), requestConfig)
	if err != nil {
		fmt.Println("Failed to get calendar view:", err)
		return
	}

	for _, event := range events.GetValue() {
		fmt.Printf("Event Id : %s\n", *event.GetId())
		fmt.Printf("  Subject: %s\n", *event.GetSubject())
		fmt.Printf("  Start: %s, End: %s\n",
			*event.GetStart().GetDateTime(),
			*event.GetEnd().GetDateTime())
		// Print start and end in local time

		localStart, err := ConvertToLocalTime(*event.GetStart().GetDateTime())
		if err != nil {
			fmt.Println("Failed to convert start time to local:", err)
			continue
		} else {
			fmt.Printf("  Local Start: %v\n", localStart)
		}
		localEnd, err := ConvertToLocalTime(*event.GetEnd().GetDateTime())
		if err != nil {
			fmt.Println("Failed to convert end time to local:", err)
			continue
		} else {
			fmt.Printf("  Local End: %v\n", localEnd)
		}
		fmt.Printf("  OnlineMeeting: %t\n", *event.GetIsOnlineMeeting())
		fmt.Printf("  isOrganiser: %t\n", *event.GetIsOrganizer())
		fmt.Printf("  isCancelled: %t\n", *event.GetIsCancelled())
		fmt.Printf("  Organiser: %v\n", *event.GetOrganizer().GetEmailAddress().GetAddress())
	}
}

func ConvertToLocalTime(timeString string) (time.Time, error) {

	// Parse the input string in RFC3339Nano format
	//t, err := time.Parse(time.RFC3339Nano, timeString) // this fromat does not work, as it exptects the timezone added.
	t, err := time.Parse("2006-01-02T15:04:05.999999999", timeString)

	if err != nil {
		return time.Time{}, err
	}

	// Convert to local time
	localTime := t.Local()

	return localTime, nil
}

// Function to create a Microsoft Graph subscription for room events
func (g *GraphHelper) CreateRoomSubscription(roomID string) error {

	println("CreateRoomSubscription" + roomID)

	// Define subscription parameters
	subscription := graphmodels.NewSubscription()
	changeType := "created,updated,deleted"
	subscription.SetChangeType(&changeType)
	notificationURL := g.GetNotificationUrl()
	if notificationURL == "" {
		log.Fatal("ENDPOINT is not set in .env file")
		return fmt.Errorf("ENDPOINT is not set in .env file")
	}
	subscription.SetNotificationUrl(&notificationURL)
	//subResource := fmt.Sprintf("/places/microsoft.graph.room/%s", roomID)
	subResource := fmt.Sprintf("/users/%s/events", roomID)
	subscription.SetResource(&subResource)
	// End time is today.
	//expirationDateTime, err := time.Now().Format(time.RFC3339)
	tomorrow := time.Now().Add(24 * time.Hour)
	tomorrow.Format("2016-11-20T18:23:45.9356913Z")
	//expirationDateTime, err := time.Parse(tomorrow, "2016-11-20T18:23:45.9356913Z")
	//expirationDateTime, err := time.Parse(time.RFC3339, "2016-11-20T18:23:45.9356913Z")
	//if err != nil {
	//	return fmt.Errorf("failed to parse expiration datetime: %v", err)
	//}
	subscription.SetExpirationDateTime(&tomorrow)
	//subscription.SetExpirationDateTime(&expirationDateTime)

	//	clientState := "secretClientValue"
	//	subscription.SetClientState(&clientState)
	//	latestSupportedTlsVersion := "v1_2"
	//	subscription.SetLatestSupportedTlsVersion(&latestSupportedTlsVersion)

	// Create the subscription
	result, err := g.appClient.Subscriptions().Post(context.Background(), subscription, nil)
	if err != nil {
		fmt.Printf("failed to create subscription: %v", err.Error())
		return fmt.Errorf("failed to create subscription: %v", err)
	}

	log.Printf("Subscription created with ID: %s", *result.GetId())
	return nil
}

// DeleteSubscription deletes a subscription with the given subscription ID.
// It returns an error if the deletion fails.
//
// Parameters:
//   - subscriptionId: The ID of the subscription to be deleted.
//
// Returns:
//   - error: An error object if the deletion fails, otherwise nil.
func (g *GraphHelper) DeleteSubscription(subscriptionId string) error {

	err := g.appClient.Subscriptions().BySubscriptionId(subscriptionId).Delete(context.Background(), nil)
	if err != nil {
		fmt.Printf("failed to delete subscription: %v", err.Error())
		return fmt.Errorf("failed to create subscription: %v", err)
	}
	return nil
}

// DeleteEvent deletes an event for a specified user.
//
// Parameters:
//   - userId: The ID of the user whose event is to be deleted.
//   - eventId: The ID of the event to be deleted.
//
// Returns:
//   - error: An error object if the deletion fails, otherwise nil.
func (g *GraphHelper) DeleteEvent(userId string, eventId string) error {

	requestBody := graphusers.NewItemEventsItemCancelPostRequestBody()
	comment := "System Canceled Event"
	requestBody.SetComment(&comment) // Initialize a new Graph client

	err := g.appClient.Users().ByUserId(userId).Events().ByEventId(eventId).Delete(context.Background(), nil)
	if err != nil {
		fmt.Printf("failed to delete event: %v", err.Error())
		return fmt.Errorf("failed to delete event: %v", err)
	}
	return nil
}

func (g *GraphHelper) CreateEvent(organiserEmail string, roomEmail string) error {

	// Create an event for tomorrow at 10:00 AM for userId and set the room/location as roomId

	startTime, endTime := GetTomorrowTimes()
	println("Tomorrow at 10:00 AM:", startTime.String())
	println("Tomorrow at 10:30 AM:", endTime.String())

	headers := abstractions.NewRequestHeaders()
	headers.Add("Prefer", "outlook.timezone=\"Pacific Standard Time\"")

	configuration := &graphusers.ItemEventsRequestBuilderPostRequestConfiguration{
		Headers: headers,
	}
	// description
	requestBody := graphmodels.NewEvent()
	subject := "Plan summer company picnic"
	requestBody.SetSubject(&subject)
	body := graphmodels.NewItemBody()
	contentType := graphmodels.HTML_BODYTYPE
	body.SetContentType(&contentType)
	content := "Let's kick-start this event planning!"
	body.SetContent(&content)
	requestBody.SetBody(body)
	// Time stuff
	start := graphmodels.NewDateTimeTimeZone()
	dateTime := startTime.UTC().Format("2006-01-02T15:04:05.999999999")
	start.SetDateTime(&dateTime)
	timeZone := "Pacific Standard Time"
	start.SetTimeZone(&timeZone)
	requestBody.SetStart(start)
	end := graphmodels.NewDateTimeTimeZone()
	dateTime = endTime.UTC().Format("2006-01-02T15:04:05.999999999")
	end.SetDateTime(&dateTime)
	timeZone = "Pacific Standard Time"
	end.SetTimeZone(&timeZone)
	requestBody.SetEnd(end)

	// Attendees

	// add organiser

	organiser := graphmodels.NewAttendee()
	orgEmailAddress := graphmodels.NewEmailAddress()
	orgEmailAddress.SetAddress(&organiserEmail)
	organiser.SetEmailAddress(orgEmailAddress)
	orgType := graphmodels.REQUIRED_ATTENDEETYPE
	organiser.SetTypeEscaped(&orgType)

	/*	attendee := graphmodels.NewAttendee()
		emailAddress := graphmodels.NewEmailAddress()
		address := "DanaS@contoso.com"
		emailAddress.SetAddress(&address)
		name := "Dana Swope"
		emailAddress.SetName(&name)
		attendee.SetEmailAddress(emailAddress)
		type1 := graphmodels.REQUIRED_ATTENDEETYPE
		attendee.SetType(&type1)
		attendee1 := graphmodels.NewAttendee()
		emailAddress := graphmodels.NewEmailAddress()
		address := "AlexW@contoso.com"
		emailAddress.SetAddress(&address)
		name := "Alex Wilber"
		emailAddress.SetName(&name)
		attendee1.SetEmailAddress(emailAddress)
		type2 := graphmodels.REQUIRED_ATTENDEETYPE
		attendee1.SetType(&type2) */

	roomAttendee := graphmodels.NewAttendee()
	roomEmailAddress := graphmodels.NewEmailAddress()
	roomEmailAddress.SetAddress(&roomEmail)
	roomResourceType := graphmodels.RESOURCE_ATTENDEETYPE
	roomAttendee.SetTypeEscaped(&roomResourceType)
	roomAttendee.SetEmailAddress(roomEmailAddress)
	attendees := []graphmodels.Attendeeable{
		//organiser,
		roomAttendee,
	}
	requestBody.SetAttendees(attendees)

	location := graphmodels.NewLocation()
	//locationType := graphmodels.DEFAULT_LOCATIONTYPE
	//location.SetLocationType(&locationType)
	location.SetLocationEmailAddress(&roomEmail)
	//displayName := "10.10.10"
	//location.SetDisplayName(&displayName)
	requestBody.SetLocation(location)

	//locations := []graphmodels.Locationable{
	//	location,
	//}
	//requestBody.SetLocations(locations)

	allowNewTimeProposals := false
	requestBody.SetAllowNewTimeProposals(&allowNewTimeProposals)

	// To initialize your graphClient, see https://learn.microsoft.com/en-us/graph/sdks/create-client?from=snippets&tabs=go
	events, err := g.appClient.Users().ByUserId(organiserEmail).Events().Post(context.Background(), requestBody, configuration)
	//g.appClient.Users().ByUserId(organiserEmail).Calendar().Events().Post(context.Background(), requestBody, configuration)
	if err != nil {
		fmt.Println("Failed to create event:", err)
		return err
	}

	fmt.Printf("%v", events)
	return nil
}

func GetTomorrowTimes() (time.Time, time.Time) {
	// Get today's date
	now := time.Now()

	// Calculate tomorrow's date
	tomorrow := now.AddDate(0, 0, 1)

	// Set the times for tomorrow at 10:00 AM and 10:30 AM
	tomorrow10am := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 0, 0, 0, tomorrow.Location())
	tomorrow1030am := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 30, 0, 0, tomorrow.Location())

	tomorrow10am.Format("2016-11-20T18:23:45.9356913Z")
	tomorrow1030am.Format("2016-11-20T18:23:45.9356913Z")

	return tomorrow10am, tomorrow1030am
}
