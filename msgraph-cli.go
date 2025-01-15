package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bovinemagnet/msgraph-cli/graphhelper"
	"github.com/gdamore/tcell/v2"
	"github.com/joho/godotenv"
	"github.com/rivo/tview"
)

type App struct {
	app            *tview.Application
	menu           *tview.List
	output         *tview.TextView
	webhookOutput  *tview.TextView
	graphHelper    *graphhelper.GraphHelper
	layout         *tview.Flex
	inputField     *tview.InputField
	footer         *tview.TextView
	mu             sync.Mutex // For thread-safe updates to text views
	roomEmail      string
	organiserEmail string
	webhookChan    chan string
	header         *tview.TextView
	timeTimer      *time.Timer
}

// Create a unified event handler struct
type EventHandler struct {
	graphHelper *graphhelper.GraphHelper
	output      io.Writer
}

// Consolidate similar event handling functions
func (h *EventHandler) handleEvent(action string, email string, params ...interface{}) error {
	switch action {
	case "create":
		return h.graphHelper.CreateEvent(context.Background(), h.output, email, params[0].(string))
	case "delete":
		return h.graphHelper.DeleteEvent(context.Background(), h.output, email, params[0].(string))
		// Add other event actions...
	}
	return fmt.Errorf("unknown action: %s", action)
}

func (a *App) setupKeyboardShortcuts() {
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF5:
			// Refresh the left menu
			a.refreshLeftMenu()
			return nil
		}
		return event
	})
}

func (a *App) refreshLeftMenu() {
	// Clear existing items
	a.menu.Clear()

	//a.handleEnv()

	// Force refresh of the display
	a.app.Draw()
}

func NewApp(graphHelper *graphhelper.GraphHelper) *App {
	app := &App{
		app:         tview.NewApplication(),
		graphHelper: graphHelper,
		webhookChan: make(chan string, 100),
	}

	// Initialize UI components
	app.setupUI()

	go app.processWebhookUpdates()

	// Add keyboard shortcuts
	//app.setupKeyboardShortcuts()

	return app
}

func (a *App) handleCreateOneDaySubscription(email string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.output.Clear()
	fmt.Fprintf(a.output, "Creating a 1 day subscription for [yellow]%s[white]...\n", email)

	a.graphHelper.CreateRoomSubscription(context.Background(), a.output, email)

}

func (a *App) handleHelp() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.output.Clear()
	fmt.Fprintf(a.output, "Showing the help text...\n\n")
	fmt.Fprintf(a.output, `[yellow]Navigation Help:[white]
	• Press [green]ESC[white] to switch focus between menu and output
	• Use [green]PgUp[white]/[green]PgDn[white] to scroll when output has focus
	• Use mouse wheel to scroll (if your terminal supports it)
	• Use arrow keys [green]↑[white]/[green]↓[white] to scroll line by line
	`)

}

func (a *App) handleDeleteEvent(email string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.output.Clear()
	fmt.Fprintf(a.output, "Please enter the event ID in the input box below and press Enter\n")
	fmt.Fprintf(a.output, "Will delete event for [yellow]%s[white]...\n\n", email)

	// Store email for use in the input handler
	a.inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			eventId := a.inputField.GetText()
			if eventId == "" {
				fmt.Fprintf(a.output, "[red]Error: Event ID cannot be empty[white]\n")
				return
			} else {
				fmt.Fprintf(a.output, "[white]Attempting to deleting event [yellow]%s[white] for [yellow]%s[white]...\n", eventId, email)
			}

			err := a.graphHelper.DeleteEvent(context.Background(), a.output, email, eventId)
			if err != nil {
				fmt.Fprintf(a.output, "[red]Error deleting event: %v[white]\n", err)
			}

			a.inputField.SetText("")
			// Reset input handler to default
			a.inputField.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEnter {
					a.handleInput(a.inputField.GetText())
					a.inputField.SetText("")
				}
			})
			a.app.SetFocus(a.menu)
		}
	})
	a.app.SetFocus(a.inputField)
}

func (a *App) handleCreateEvent(email string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.output.Clear()
	fmt.Fprintf(a.output, "Creating an event for [yellow]%s[white]...\n", email)

	a.graphHelper.CreateEvent(context.Background(), a.output, a.organiserEmail, a.roomEmail)

}

func (a *App) setupUI() {
	// Create header
	a.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	a.updateHeader() // Initial header update

	// Create output panel
	a.output = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		//SetSelectable(true). // Enable text selection
		SetWordWrap(true). // Enable word wrap
		//SetBackgroundColor(tcell.ColorDarkGrey).
		SetChangedFunc(func() {
			a.app.Draw()
		})

	// Create a frame for the output with just top/bottom borders
	outputFrame := tview.NewFrame(a.output).
		SetBorders(0, 0, 1, 1, 0, 0). // top, bottom, left, right padding
		AddText("Output:", true, tview.AlignLeft, tcell.ColorWhite)

	// Create input field - single line
	a.inputField = tview.NewInputField().
		SetLabel("Input: ").
		SetFieldWidth(0). // Full width
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				inputText := a.inputField.GetText()
				a.handleInput(inputText)
				a.inputField.SetText("")
			}
		})

	// Create webhook output panel
	a.webhookOutput = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			a.app.Draw()
		})
	a.webhookOutput.SetBorder(true).SetTitle("Webhook Events")

	// Create menu
	a.menu = tview.NewList().
		ShowSecondaryText(false) //. // Hide descriptions for compact view
	//SetSpacing(2)             // Space between items

	a.menu.SetBorder(true).SetTitle("Menu")

	a.menu.AddItem("Env", "Show the contents of the .env file", 'e', a.handleEnv)

	a.menu.AddItem("Access Token", "Show the current access token", 't', a.handleAccessToken)

	a.menu.AddItem("All Users", "List all users", 'u', a.handleListUsers)
	a.menu.AddItem("All Subscriptions", "List all subscriptions", 's', a.HandleListSubscriptions)
	a.menu.AddItem("All Rooms", "List all rooms", 'r', a.handleListRooms)

	a.menu.AddItem("Room Bookings (Organiser)", "List room bookings for "+a.organiserEmail, 'O', func() {
		a.handleListRoomBookings(a.organiserEmail)
	})
	a.menu.AddItem("Room Bookings (Room)", "List room bookings for "+a.roomEmail, 'R', func() {
		a.handleListRoomBookings(a.roomEmail)
	})
	a.menu.AddItem("Org Subscribe", "Create a 1 day subscription for "+a.organiserEmail, '7', func() {
		a.handleCreateOneDaySubscription(a.organiserEmail)
	})
	a.menu.AddItem("Room Subscribe", "Create a 1 day subscription for"+a.roomEmail, '8', func() {
		a.handleCreateOneDaySubscription(a.roomEmail)
	})

	a.menu.AddItem("Delete event", " id - By Room  "+a.roomEmail+" ", '9', func() {
		a.handleDeleteEvent(a.roomEmail)
	})
	a.menu.AddItem("Delete event", " id - By Organiser  "+a.organiserEmail+"]", '9', func() {
		a.handleDeleteEvent(a.organiserEmail)
	})

	a.menu.AddItem(" Create Event", "at 10 to 10:30 tomorrow - By Room ["+a.roomEmail+"]", 11, func() {
		a.handleCreateEvent(a.roomEmail)
	})
	a.menu.AddItem(" Create Event", "at 10 to 10:30 tomorrow - By Organiser ["+a.organiserEmail+"]", 12, func() {
		a.handleCreateEvent(a.organiserEmail)
	})
	//	fmt.Println("  13. Check room exists - By Room [" + roomEmail + "]")
	//	fmt.Println("  14. Check room exists - By Organiser [" + organiserEmail + "]")

	a.menu.AddItem("Enter Text", "Display text from input field", 'I', func() {
		a.app.SetFocus(a.inputField)
	})
	a.menu.AddItem("Help", "Show the help text", 'h', a.handleHelp)

	a.menu.AddItem("Quit", "Exit the application", 'q', func() {
		a.app.Stop()
	})

	// Create layout with adjusted heights
	a.layout = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.header, 1, 1, false).
		AddItem(outputFrame, 0, 8, false). // Use frame instead of direct output
		AddItem(a.inputField, 1, 1, false).
		AddItem(a.webhookOutput, 8, 1, false).
		AddItem(a.menu, 0, 2, true)

	// Start time updates in header
	go a.startTimeUpdates()

	// Set up key bindings
	a.layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Check for Alt key combinations
		if event.Modifiers() == tcell.ModAlt {
			switch event.Rune() {
			case 'o': // Alt-o
				a.app.SetFocus(a.output)
				return nil
			case 'm': // Alt-m
				a.app.SetFocus(a.menu)
				return nil
			case 'w': // Alt-w
				a.app.SetFocus(a.webhookOutput)
				return nil
			}
		}

		// Original Esc handling
		if event.Key() == tcell.KeyEsc {
			if a.menu.HasFocus() {
				a.app.SetFocus(a.output)
			} else {
				a.app.SetFocus(a.menu)
			}
		}
		return event
	})

	// Set up scroll handlers
	a.output.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := a.output.GetScrollOffset()
		switch event.Key() {
		case tcell.KeyPgDn:
			a.output.ScrollTo(row+10, 0)
		case tcell.KeyPgUp:
			a.output.ScrollTo(row-10, 0)
		}
		return event
	})

	a.webhookOutput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := a.webhookOutput.GetScrollOffset()
		switch event.Key() {
		case tcell.KeyPgDn:
			a.webhookOutput.ScrollTo(row+10, 0)
		case tcell.KeyPgUp:
			a.webhookOutput.ScrollTo(row-10, 0)
		}
		return event
	})

	a.app.SetRoot(a.layout, true)
}

// Example menu handler functions
func (a *App) handleAccessToken() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.output.Clear()
	fmt.Fprintf(a.output, "Displaying the access token...\n")

	a.graphHelper.DisplayAccessTokenA(a.output)
}

func (a *App) handleListRoomBookings(emailAddress string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.output.Clear()
	fmt.Fprintf(a.output, "Listing room bookings for [yellow]%s[white]...\n", emailAddress)

	a.graphHelper.ListRoom7DaysBookings(a.output, emailAddress)

}

func (a *App) appendToWebhookOutput(text string) {
	a.app.QueueUpdateDraw(func() {
		a.mu.Lock()
		defer a.mu.Unlock()

		fmt.Fprintf(a.webhookOutput, "%s", text)
		// Get the number of lines in the buffer
		_, _, _, height := a.webhookOutput.GetInnerRect()
		count := len(a.webhookOutput.GetText(true)) - height
		if count > 0 {
			// Scroll to bottom
			a.webhookOutput.ScrollTo(count, 0)
		}
	})
}

func (a *App) handleInput(input string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.output.Clear()
	fmt.Fprintf(a.output, "[yellow]You entered:[white]\n%s\n", input)
	a.app.SetFocus(a.menu) // Return focus to menu after input
}

func (a *App) processWebhookUpdates() {
	ticker := time.NewTicker(100 * time.Millisecond)
	var updates []string

	for {
		select {
		case update := <-a.webhookChan:
			updates = append(updates, update)
		case <-ticker.C:
			if len(updates) > 0 {
				a.appendToWebhookOutput(strings.Join(updates, "\n"))
				updates = updates[:0]
			}
		}
	}
}

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

	// Create the application
	app := NewApp(graphHelper)
	defer app.Cleanup()

	organiserEmail := graphHelper.GetOrganiserEmail()
	if organiserEmail == "" {
		fmt.Println("No organiser found")
	}
	app.organiserEmail = organiserEmail

	roomEmail := graphHelper.GetRoomEmail()
	if roomEmail == "" {
		fmt.Println("No room email found")
	}
	app.roomEmail = roomEmail

	// Start webhook server in a goroutine
	webhookServer := NewWebhookServer(app)
	go func() {
		if err := webhookServer.Start(); err != nil {
			log.Fatalf("Webhook server error: %v", err)
		}
	}()

	// Run the application
	if err := app.app.Run(); err != nil {
		log.Fatal(err)
	}

	/*
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
			fmt.Println("  +-----------------------------------+")
			fmt.Println("  5.  List 7 days of Events - By Room [" + roomEmail + "]")
			fmt.Println("  6.  List 7 days of Events - By Organiser [" + organiserEmail + "]")
			fmt.Println("  +-----------------------------------+")
			fmt.Println("  7.  Create a 1 day subscription - By Room [" + roomEmail + "]")
			fmt.Println("  8.  Delete a subscription by the subscription id")
			fmt.Println("  +-----------------------------------+")
			fmt.Println("  9.  Delete event id - By Room [" + roomEmail + "]")
			fmt.Println("  10. Delete event id - By Organiser [" + organiserEmail + "]")
			fmt.Println("  +-----------------------------------+")
			fmt.Println("  11. Create Event at 10 to 10:30 tomorrow - By Room [" + roomEmail + "]")
			fmt.Println("  12. Create Event at 10 to 10:30 tomorrow - By Organiser [" + organiserEmail + "]")
			fmt.Println("  +-----------------------------------+")
			fmt.Println("  13. Check room exists - By Room [" + roomEmail + "]")
			fmt.Println("  14. Check room exists - By Organiser [" + organiserEmail + "]")
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
			case 12:
				// delete event by event id for the specified organiser
				createEventByOrganiser(graphHelper)
			case 11:
				// delete event by event id for the specified organiser
				createEventByRoom(graphHelper)
			case 13:
				// check room exists
				validateRoomByRoom(graphHelper)
			case 14:
				// check room exists  by organiser, should be false all the time.
				validateRoomByOrganiser(graphHelper)
			default:
				fmt.Println("Invalid choice! Please try again.")
			}

			if choice == 0 {
				break
			}
		}
	*/
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
	users, err := graphHelper.GetUsers(context.Background())
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

func listRooms(graphHelper *graphhelper.GraphHelper) {

	graphHelper.ListRooms(os.Stdout)

}

func listRoomBookingsAsOrganiser(graphHelper *graphhelper.GraphHelper) {

	organiser := graphHelper.GetOrganiserEmail()
	if organiser == "" {
		fmt.Println("No organiser found")
		return
	}

	graphHelper.ListRoom7DaysBookings(os.Stdout, organiser)

}

func listRoomBookingsAsRoom(graphHelper *graphhelper.GraphHelper) {

	roomEmail := graphHelper.GetRoomEmail()
	if roomEmail == "" {
		fmt.Println("No room email found")
		return
	}

	graphHelper.ListRoom7DaysBookings(os.Stdout, roomEmail)

}

/*func handleGraphSubscription(w http.ResponseWriter, r *http.Request) {
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
}*/

func createOneDaySubscription(graphHelper *graphhelper.GraphHelper) {
	roomEmail := graphHelper.GetRoomEmail()
	if roomEmail == "" {
		fmt.Println("No room email found")
		return
	}

	values := graphHelper.CreateRoomSubscription(context.Background(), os.Stdout, roomEmail)
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
	err = graphHelper.DeleteSubscription(os.Stdout, subscriptionId)
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
	err = graphHelper.DeleteEvent(context.Background(), os.Stdout, organiser, eventId)
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
	err = graphHelper.DeleteEvent(context.Background(), os.Stdout, roomEmail, eventId)
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

	err := graphHelper.CreateEvent(context.Background(), os.Stdout, organiser, roomEmail)
	if err != nil {
		log.Printf("Error creating event: %v", err)
		return
	}

}

func createEventByRoom(graphHelper *graphhelper.GraphHelper) {

	roomEmail := graphHelper.GetRoomEmail()
	if roomEmail == "" {
		fmt.Println("No room email found")
		return
	}

	err := graphHelper.CreateEventAsRoom(os.Stdout, roomEmail)
	if err != nil {
		log.Printf("Error creating event: %v", err)
		return
	}

}

func validateRoomByOrganiser(graphHelper *graphhelper.GraphHelper) {

	organiser := graphHelper.GetOrganiserEmail()
	if organiser == "" {
		fmt.Println("No organiser found")
		return
	}

	exists, err := graphHelper.RoomExists2(os.Stdout, organiser)
	if err != nil {
		log.Printf("Error checking room: %v", err)
		return
	}
	if exists {
		fmt.Printf("Room %s exists\n", organiser)
	} else {
		fmt.Printf("Room %s does not exist\n", organiser)
	}

}

func validateRoomByRoom(graphHelper *graphhelper.GraphHelper) {

	roomEmail := graphHelper.GetRoomEmail()
	if roomEmail == "" {
		fmt.Println("No room email found")
		return
	}

	exists, err := graphHelper.RoomExists2(os.Stdout, roomEmail)
	if err != nil {
		log.Printf("Error checking room: %v", err)
		return
	}
	if exists {
		fmt.Printf("Room %s exists\n", roomEmail)
	} else {
		fmt.Printf("Room %s does not exist\n", roomEmail)
	}
}

// Batch UI updates
func (a *App) updateOutput(updates ...string) {
	a.app.QueueUpdateDraw(func() {
		a.mu.Lock()
		defer a.mu.Unlock()

		for _, update := range updates {
			fmt.Fprintf(a.output, "%s", update)
		}
	})
}

func (a *App) updateHeader() {
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	headerText := fmt.Sprintf("[yellow]Microsoft Graph Room Booking CLI[white] | %s", currentTime)
	a.header.SetText(headerText)
}

func (a *App) startTimeUpdates() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.app.QueueUpdateDraw(func() {
				a.updateHeader()
			})
		}
	}
}

func (a *App) Cleanup() {
	// Add any cleanup needed when the application exits
	if a.timeTimer != nil {
		a.timeTimer.Stop()
	}
}
