# msgraph-cli
Simple Microsoft Graph CLI Application

A simple tool based on the Microsoft Graph SDK to interact with Microsoft Graph, utilising the ms graph app tutorial.

## Example

```shell
Please choose one of the following options:
  0.  Exit
  1.  Display access token
  +-----------------------------------+
  2.  List All Users
  3.  List All Subscriptions
  4.  List All Rooms
  5.  List 7 days of Events - By Room [my_room@example.onmicrosoft.com]
  6.  List 7 days of Events - By Organiser [my_user@example.onmicrosoft.com]
  +-----------------------------------+
  7.  Create a 1 day subscription - By Room [my_room@example.onmicrosoft.com]
  8.  Delete a subscription by the subscription id
  +-----------------------------------+
  9.  Delete event id - By Room [my_room@example.onmicrosoft.com]
  10. Delete event id - By Organiser [my_useraul@example.onmicrosoft.com]
  +-----------------------------------+
:>
```


## Options

### Display access token
This option will display the access token that is being used to authenticate with Microsoft Graph.

### List All Users

This option will list all users in the tenant.

### List All Subscriptions

This option will list all subscriptions in the tenant.

### List All Rooms

This option will list all rooms in the tenant.

### List 7 days of Events - By Room

List all the events for the given room.

### List 7 days of Events - By Organiser

List all the events for the given organiser.

### Create a 1 day subscription - By Room

Create a subscription for the given room.

### Delete a subscription by the subscription id

Delete a subscription by the subscription id.

### Delete event id - By Room

Delete an event by the event id for the given room.

### Delete event id - By Organiser

Delete an event by the event id for the given organiser.

## Setup

Using the .env file

```shell
CLIENT_ID=Enter your client ID
CLIENT_SECRET=Enter your client secret
TENANT_ID=Enter your tenant ID
ORGANISER_EMAIL=Enter your organiser email
ROOM_EMAIL=enter your room email
ENDPOINT=enter your endpoint `https://ngrok.stuff/webhook` eg via ngrok
PORT=8080
```
