package main

import "fmt"

func (a *App) handleListRooms() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.output.Clear()
	fmt.Fprintf(a.output, "Listing rooms...\n")

	a.graphHelper.ListRooms(a.output)

	/*	rooms, err := a.graphHelper.ListRooms(a.output)
		if err != nil {
			fmt.Fprintf(a.output, "[red]Error making Graph call: %v[white]\n", err)
			return
		}

		if rooms == nil || rooms.GetOdataCount() == nil {
			fmt.Fprintf(a.output, "No rooms found\n")
			return
		} else {
			fmt.Fprintf(a.output, "Number of rooms %v", rooms.GetOdataCount())
		}

		// Use GetValue() to get the slice of rooms
		for _, room := range rooms.GetValue() {
			fmt.Fprintf(a.output, "[yellow]Room:[white] %s (%s)\n",
				*room.GetDisplayName(),
				*room.GetEmailAddress())
		} */
}
