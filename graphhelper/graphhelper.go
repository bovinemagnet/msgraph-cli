package graphhelper

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	auth "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	graphusers "github.com/microsoftgraph/msgraph-sdk-go/users"
	"golang.org/x/time/rate"
)

type GraphHelper struct {
	clientSecretCredential *azidentity.ClientSecretCredential
	appClient              *msgraphsdk.GraphServiceClient
	cache                  struct {
		rooms      map[string]graphmodels.Roomable
		users      map[string]graphmodels.Userable
		lastUpdate time.Time
		mu         sync.RWMutex
	}
	clientPool  sync.Pool
	rateLimiter *rate.Limiter
}

func NewGraphHelper() *GraphHelper {
	g := &GraphHelper{
		clientPool: sync.Pool{
			New: func() interface{} {
				// Create new client
				client := msgraphsdk.NewGraphServiceClient(nil) // ... initialize new client
				return client
			},
		},
		rateLimiter: rate.NewLimiter(rate.Every(time.Second/10), 10), // 10 requests per second
	}
	return g
}

func (g *GraphHelper) getClient() *msgraphsdk.GraphServiceClient {
	return g.clientPool.Get().(*msgraphsdk.GraphServiceClient)
}

func (g *GraphHelper) putClient(client *msgraphsdk.GraphServiceClient) {
	g.clientPool.Put(client)
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

func (g *GraphHelper) GetUsers(ctx context.Context) (graphmodels.UserCollectionResponseable, error) {
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
		Get(ctx,
			&graphusers.UsersRequestBuilderGetRequestConfiguration{
				QueryParameters: &query,
			})
}

func (g *GraphHelper) ListUsers(w io.Writer) {
	users, err := g.GetUsers(context.Background())
	if err != nil {
		log.Panicf("Error getting users: %v", err)
	}

	// Output each user's details
	for _, user := range users.GetValue() {
		fmt.Fprintf(w, "[yellow]User Id: [green]%s[white]\n", *user.GetId())
		fmt.Fprintf(w, "  Name: %s\n", *user.GetDisplayName())
		noEmail := "NO EMAIL"
		email := user.GetMail()
		if email == nil {
			email = &noEmail
		}
		fmt.Fprintf(w, "  Email: %s\n", *email)
		fmt.Fprintf(w, "  Enabled: %v\n", user.GetAccountEnabled())
		fmt.Fprintf(w, "  IsResourceAccount: %v\n", user.GetIsResourceAccount())
		fmt.Fprintf(w, "\n")
	}

	// If GetOdataNextLink does not return nil,
	// there are more users available on the server
	nextLink := users.GetOdataNextLink()

	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "More users available? %t\n", nextLink != nil)
	fmt.Fprintf(w, "\n")
}

func (g *GraphHelper) ListSubscriptions(ctx context.Context) (graphmodels.SubscriptionCollectionResponseable, error) {

	return g.appClient.Subscriptions().
		Get(ctx, nil)

}

func (g *GraphHelper) ListSubscriptionsByEmail(email string) (graphmodels.SubscriptionCollectionResponseable, error) {

	return g.appClient.Subscriptions().
		Get(context.Background(), nil)

}

func (g *GraphHelper) ListRoomsAll() (graphmodels.RoomListCollectionResponseable, error) {
	return g.appClient.Places().GraphRoomList().Get(context.Background(), nil)
}

// ListRooms
func (g *GraphHelper) ListRooms(w io.Writer) {
	// (places.GraphRoomRequestBuilder, error) {

	rooms, err := g.appClient.Places().GraphRoom().Get(context.Background(), nil)
	if err != nil {
		fmt.Fprintln(w, "Failed to list rooms:", err)
		return
	}

	for _, room := range rooms.GetValue() {
		fmt.Fprintf(w, "[yellow]Room ID: [green]%s[white]\n", *room.GetId())
		fmt.Fprintf(w, "  Name: %s\n", *room.GetDisplayName())
		fmt.Fprintf(w, "  Capacity: %d\n", *room.GetCapacity())
		fmt.Fprintf(w, "  Email: %s\n\n", *room.GetEmailAddress())
	}

	return

}

func (g *GraphHelper) DisplayAccessTokenA(w io.Writer) {
	token, err := g.GetAppToken()
	if err != nil {
		log.Panicf("Error getting user token: %v\n", err)
		fmt.Fprintf(w, "No token: %v\n", err)
		return
	}
	fmt.Fprintf(w, "App-only token: %s\n", *token)
}

func (g *GraphHelper) ListRoom7DaysBookings(w io.Writer, roomId string) {
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
		fmt.Fprintln(w, "Failed to get calendar view:", err)
		return
	}

	for _, event := range events.GetValue() {
		fmt.Fprintf(w, "[yellow]Event Id : [green]%s[white]\n", *event.GetId())
		fmt.Fprintf(w, "  Subject: [blue]%s[white]\n", *event.GetSubject())
		fmt.Fprintf(w, "  Start: %s, End: %s\n",
			*event.GetStart().GetDateTime(),
			*event.GetEnd().GetDateTime())
		// Print start and end in local time

		localStart, err := ConvertToLocalTime(*event.GetStart().GetDateTime())
		if err != nil {
			fmt.Fprintln(w, "Failed to convert start time to local:", err)
			continue
		} else {
			fmt.Fprintf(w, "  Local Start: %v\n", localStart)
		}
		localEnd, err := ConvertToLocalTime(*event.GetEnd().GetDateTime())
		if err != nil {
			fmt.Fprintln(w, "Failed to convert end time to local:", err)
			continue
		} else {
			fmt.Fprintf(w, "  Local End: %v\n", localEnd)
		}
		fmt.Fprintf(w, "  OnlineMeeting: %t\n", *event.GetIsOnlineMeeting())
		if event.GetIsOrganizer() == nil || *event.GetIsOrganizer() == false {
			fmt.Fprintf(w, "  isOrganiser: [red]%t[white]\n", *event.GetIsOrganizer())
		} else {
			fmt.Fprintf(w, "  isOrganiser: [green]%t[white]\n", *event.GetIsOrganizer())
		}
		fmt.Fprintf(w, "  isCancelled: %t\n", *event.GetIsCancelled())
		fmt.Fprintf(w, "  Organiser: [yellow]%v[white]\n", *event.GetOrganizer().GetEmailAddress().GetAddress())
		fmt.Fprintf(w, " \n")
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
func (g *GraphHelper) CreateRoomSubscription(ctx context.Context, w io.Writer, roomID string) error {

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
	result, err := g.appClient.Subscriptions().Post(ctx, subscription, nil)
	if err != nil {
		fmt.Fprintf(w, "failed to create subscription: %v", err.Error())
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
func (g *GraphHelper) DeleteSubscription(w io.Writer, subscriptionId string) error {

	err := g.appClient.Subscriptions().BySubscriptionId(subscriptionId).Delete(context.Background(), nil)
	if err != nil {
		fmt.Fprintf(w, "failed to delete subscription: %v", err.Error())
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
func (g *GraphHelper) DeleteEvent(ctx context.Context, w io.Writer, userId string, eventId string) error {

	requestBody := graphusers.NewItemEventsItemCancelPostRequestBody()
	comment := "System Canceled Event"
	requestBody.SetComment(&comment) // Initialize a new Graph client

	err := g.appClient.Users().ByUserId(userId).Events().ByEventId(eventId).Delete(ctx, nil)
	if err != nil {
		fmt.Fprintf(w, "[red]failed to delete event:[%v] error:[%v][white] for [yellow]%s[white]\n", eventId, err.Error(), userId)
		return fmt.Errorf("failed to delete event: %v", err)
	}
	return nil
}

// Create custom error types
type GraphError struct {
	Operation string
	Err       error
}

func (e *GraphError) Error() string {
	return fmt.Sprintf("graph operation '%s' failed: %v", e.Operation, e.Err)
}

func (g *GraphHelper) CreateEvent(ctx context.Context, w io.Writer, organiserEmail string, roomEmail string) error { // Create an event for tomorrow at 10:00 AM for userId and set the room/location as roomId
	// Add timeout if not set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	startTime, endTime := GetTomorrowTimes()
	fmt.Fprintln(w, "Tomorrow at 10:00 AM:", startTime.String())
	fmt.Fprintln(w, "Tomorrow at 10:30 AM:", endTime.String())

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

	roomAttendee := graphmodels.NewAttendee()
	roomEmailAddress := graphmodels.NewEmailAddress()
	roomEmailAddress.SetAddress(&roomEmail)
	roomResourceType := graphmodels.RESOURCE_ATTENDEETYPE
	roomAttendee.SetTypeEscaped(&roomResourceType)
	roomAttendee.SetEmailAddress(roomEmailAddress)
	attendees := []graphmodels.Attendeeable{
		// organiser,
		roomAttendee,
	}
	requestBody.SetAttendees(attendees)

	location := graphmodels.NewLocation()
	location.SetLocationEmailAddress(&roomEmail)
	requestBody.SetLocation(location)

	//locations := []graphmodels.Locationable{
	//      location,
	//}
	//requestBody.SetLocations(locations)

	allowNewTimeProposals := false
	requestBody.SetAllowNewTimeProposals(&allowNewTimeProposals)

	// To initialize your graphClient, see https://learn.microsoft.com/en-us/graph/sdks/create-client?from=snippets&tabs=go
	//createdEvent, err := g.appClient.Users().ByUserId(organiserEmail).Events().Post(context.Background(), requestBody, configuration)

	// Use context in API calls
	createdEvent, err := g.appClient.Users().
		ByUserId(organiserEmail).
		Events().
		Post(ctx, requestBody, configuration)
	if err != nil {
		fmt.Fprintln(w, "Failed to create event:", err)
		return err
	}
	// g.appClient.Users().ByUserId(organiserEmail).Calendar().Events().Post(context.Background(), requestBody, configuration)
	if err != nil {
		fmt.Fprintln(w, "Failed to create event:", err)
		return err
	}

	fmt.Fprintf(w, "[yellow]Event Id : [green]%s[white]\n", *createdEvent.GetId())
	fmt.Fprintf(w, "  Subject: %s\n", *createdEvent.GetSubject())
	fmt.Fprintf(w, "  Start: %s, End: %s\n",
		*createdEvent.GetStart().GetDateTime(),
		*createdEvent.GetEnd().GetDateTime())
	// Print start and end in local time

	localStart, err := ConvertToLocalTime(*createdEvent.GetStart().GetDateTime())
	if err != nil {
		fmt.Fprintln(w, "Failed to convert start time to local:", err)

	} else {
		fmt.Fprintf(w, "  Local Start: %v\n", localStart)
	}
	localEnd, err := ConvertToLocalTime(*createdEvent.GetEnd().GetDateTime())
	if err != nil {
		fmt.Fprintln(w, "Failed to convert end time to local:", err)

	} else {
		fmt.Fprintf(w, "  Local End: %v\n", localEnd)
	}
	fmt.Fprintf(w, "  OnlineMeeting: %t\n", *createdEvent.GetIsOnlineMeeting())
	if createdEvent.GetIsOrganizer() == nil || *createdEvent.GetIsOrganizer() == false {
		fmt.Fprintf(w, "  isOrganiser: [red]%t[white]\n", *createdEvent.GetIsOrganizer())
	} else {
		fmt.Fprintf(w, "  isOrganiser: [green]%t[white]\n", *createdEvent.GetIsOrganizer())
	}
	fmt.Fprintf(w, "  isCancelled: %t\n", *createdEvent.GetIsCancelled())
	fmt.Fprintf(w, "  Organiser: [yellow]%v[white]\n", *createdEvent.GetOrganizer().GetEmailAddress().GetAddress())
	return nil
}

// Use structured error handling
func (g *GraphHelper) CreateEventz(ctx context.Context, w io.Writer, organiserEmail, roomEmail string) error {
	// Add timeout if not set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}
	headers := abstractions.NewRequestHeaders()
	headers.Add("Prefer", "outlook.timezone=\"Pacific Standard Time\"")

	configuration := &graphusers.ItemEventsRequestBuilderPostRequestConfiguration{
		Headers: headers,
	}

	requestBody := graphmodels.NewEvent()

	// Use context in API calls
	createdEvent, err := g.appClient.Users().
		ByUserId(organiserEmail).
		Events().
		Post(ctx, requestBody, configuration)
	if err != nil {
		fmt.Fprintln(w, "Failed to create event:", err)
		return err
	}

	fmt.Fprintf(w, "[yellow]Event Id : [green]%s[white]\n", *createdEvent.GetId())
	fmt.Fprintf(w, "  Subject: %s\n", *createdEvent.GetSubject())
	fmt.Fprintf(w, "  Start: %s, End: %s\n",
		*createdEvent.GetStart().GetDateTime(),
		*createdEvent.GetEnd().GetDateTime())
	// Print start and end in local time

	localStart, err := ConvertToLocalTime(*createdEvent.GetStart().GetDateTime())
	if err != nil {
		fmt.Fprintln(w, "Failed to convert start time to local:", err)

	} else {
		fmt.Fprintf(w, "  Local Start: %v\n", localStart)
	}
	localEnd, err := ConvertToLocalTime(*createdEvent.GetEnd().GetDateTime())
	if err != nil {
		fmt.Fprintln(w, "Failed to convert end time to local:", err)

	} else {
		fmt.Fprintf(w, "  Local End: %v\n", localEnd)
	}
	fmt.Fprintf(w, "  OnlineMeeting: %t\n", *createdEvent.GetIsOnlineMeeting())
	if createdEvent.GetIsOrganizer() == nil || *createdEvent.GetIsOrganizer() == false {
		fmt.Fprintf(w, "  isOrganiser: [red]%t[white]\n", *createdEvent.GetIsOrganizer())
	} else {
		fmt.Fprintf(w, "  isOrganiser: [green]%t[white]\n", *createdEvent.GetIsOrganizer())
	}
	fmt.Fprintf(w, "  isCancelled: %t\n", *createdEvent.GetIsCancelled())
	fmt.Fprintf(w, "  Organiser: [yellow]%v[white]\n", *createdEvent.GetOrganizer().GetEmailAddress().GetAddress())
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

func (g *GraphHelper) CreateEventAsRoom(w io.Writer, roomEmail string) error {

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
	roomAttendee := graphmodels.NewAttendee()
	roomEmailAddress := graphmodels.NewEmailAddress()
	roomEmailAddress.SetAddress(&roomEmail)
	roomResourceType := graphmodels.RESOURCE_ATTENDEETYPE
	roomAttendee.SetTypeEscaped(&roomResourceType)
	roomAttendee.SetEmailAddress(roomEmailAddress)
	attendees := []graphmodels.Attendeeable{
		roomAttendee,
	}
	requestBody.SetAttendees(attendees)

	location := graphmodels.NewLocation()
	location.SetLocationEmailAddress(&roomEmail)
	requestBody.SetLocation(location)

	allowNewTimeProposals := false
	requestBody.SetAllowNewTimeProposals(&allowNewTimeProposals)

	// To initialize your graphClient, see https://learn.microsoft.com/en-us/graph/sdks/create-client?from=snippets&tabs=go
	createdEvent, err := g.appClient.Users().ByUserId(roomEmail).Events().Post(context.Background(), requestBody, configuration)
	if err != nil {
		fmt.Fprintln(w, "Failed to create event:", err)
		return err
	}

	fmt.Fprintf(w, "Event Id : %s\n", *createdEvent.GetId())
	fmt.Fprintf(w, "  Subject: %s\n", *createdEvent.GetSubject())
	fmt.Fprintf(w, "  Start: %s, End: %s\n",
		*createdEvent.GetStart().GetDateTime(),
		*createdEvent.GetEnd().GetDateTime())
	// Print start and end in local time

	localStart, err := ConvertToLocalTime(*createdEvent.GetStart().GetDateTime())
	if err != nil {
		fmt.Fprintln(w, "Failed to convert start time to local:", err)

	} else {
		fmt.Fprintf(w, "  Local Start: %v\n", localStart)
	}
	localEnd, err := ConvertToLocalTime(*createdEvent.GetEnd().GetDateTime())
	if err != nil {
		fmt.Fprintln(w, "Failed to convert end time to local:", err)

	} else {
		fmt.Fprintf(w, "  Local End: %v\n", localEnd)
	}
	fmt.Fprintf(w, "  OnlineMeeting: %t\n", *createdEvent.GetIsOnlineMeeting())
	fmt.Fprintf(w, "  isOrganiser: %t\n", *createdEvent.GetIsOrganizer())
	fmt.Fprintf(w, "  isCancelled: %t\n", *createdEvent.GetIsCancelled())
	fmt.Fprintf(w, "  Organiser: %v\n", *createdEvent.GetOrganizer().GetEmailAddress().GetAddress())
	return nil
}

func (g *GraphHelper) RoomExists(w io.Writer, roomEmail string) (bool, error) {

	// look up the room by room email.
	room, err := g.appClient.Users().ByUserId(roomEmail).Get(context.Background(), nil)
	if err != nil {
		// If there's an error, it might mean the room doesn't exist or you don't have permission
		fmt.Fprintf(w, "Room with email %s does not exist or you lack permissions: %v\n", roomEmail, err)
		return false, err
	}

	if room != nil {
		fmt.Fprintf(w, "Room Info\n")
		fmt.Fprintf(w, "  Id : %v\n", *room.GetId())
		fmt.Fprintf(w, "  Display Name : %v\n", *room.GetDisplayName())
		fmt.Fprintf(w, "  Type : %v\n", room.GetUserType())
	}

	// Check if the user is actually a room
	if room.GetUserType() != nil && *room.GetUserType() == "Room" {
		fmt.Fprintf(w, "Room with email %v exists and is a type=%v and isResource=%v. [%s]\n", roomEmail, room.GetUserType(), room.GetIsResourceAccount(), *room.GetId())

		return true, nil
	} else {
		fmt.Fprintf(w, "User with email %v exists but is not a room.and is a type=%v and isResource=%v [%s]\n", roomEmail, room.GetUserType(), room.GetIsResourceAccount(), *room.GetId())
		return false, nil
	}

}

func (g *GraphHelper) RoomExists2(w io.Writer, roomEmail string) (bool, error) {

	// look up the room by room email.
	rooms, err := g.appClient.Places().ByPlaceId(roomEmail).GraphRoomList().Rooms().Get(context.Background(), nil)
	if err != nil {
		// If there's an error, it might mean the room doesn't exist or you don't have permission
		fmt.Fprintf(w, "Room with email %s does not exist or you lack permissions: %v\n", roomEmail, err)
		return false, err
	}

	for _, room := range rooms.GetValue() {
		if room != nil {
			fmt.Fprintf(w, "Room Info 2\n")
			fmt.Fprintf(w, "  Id : %v\n", *room.GetId())
			fmt.Fprintf(w, "  Display Name : %v\n", *room.GetDisplayName())
			fmt.Fprintf(w, "  Type : %v\n", room.GetCapacity())
		}
	}
	if err != nil {
		// If there's an error, it might mean the room doesn't exist or you don't have permission
		fmt.Fprintf(w, "Room with email %s does not exist or you lack permissions: %v\n", roomEmail, err)
	}

	return true, nil
}

func (g *GraphHelper) GetRoom(email string) (graphmodels.Roomable, error) {
	g.cache.mu.RLock()
	if room, ok := g.cache.rooms[email]; ok && time.Since(g.cache.lastUpdate) < 5*time.Minute {
		g.cache.mu.RUnlock()
		return room, nil
	}
	g.cache.mu.RUnlock()

	// Fetch and cache room
	g.cache.mu.Lock()
	defer g.cache.mu.Unlock()

	// 3. Build the query parameter object for /places?$filter=emailAddress eq '<room>'
	//filterString := fmt.Sprintf("emailAddress eq '%s'", email)
	//queryParams := &graphmodels.PlaceRequestBuilderGetQueryParameters{
	//	Filter: &filterString,
	//	}

	// 4. Build the request configuration with the query params
	//	requestConfig := g.appClient.PlacesRequestBuilderGetRequestConfiguration{
	//		QueryParameters: queryParams,
	//	}

	// 5. Call the .Get(...) method to retrieve the place(s)
	//	ctx := context.Background()
	// Fetch room from API
	//	room, err := g.appClient.Places().Get(ctx, requestConfig)

	// Fetch room from API
	//	room, err := g.appClient.Places().GraphRoom().ByPlaceId(email).Get(context.Background(), nil)
	//	if err != nil {
	//		return nil, err
	//	}

	// Initialize cache map if nil
	//	if g.cache.rooms == nil {
	//		g.cache.rooms = make(map[string]graphmodels.Roomable)
	//	}

	//	g.cache.rooms[email] = room
	//	g.cache.lastUpdate = time.Now()
	//	return room, nil
	return nil, nil
}

func (g *GraphHelper) makeRequest(ctx context.Context, fn func() error) error {
	if err := g.rateLimiter.Wait(ctx); err != nil {
		return err
	}
	return fn()
}
