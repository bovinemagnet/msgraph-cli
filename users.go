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

/*
func (a *App) getUserIDByEmail(email string) (string, error) {

	filter := fmt.Sprintf("mail eq '%s'", email)
	requestConfiguration := &users.UsersRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.UsersRequestBuilderGetQueryParameters{
			Filter: &filter,
		},
	}

	users, err := a.graphHelper.GetUser(context.Background(), requestConfiguration)
	if err != nil {
		return "", err
	}

	if users != nil && len(users.GetValue()) > 0 {
		return *users.GetValue()[0].GetId(), nil
	}
	return "", fmt.Errorf("No user found with email %s", email)
}
*/
