package main

import (
	//	"encoding/json"
	"fmt"
	"github.com/dghubble/oauth1"
	"log"
	"net/http"
	"os"
	//	"os/signal"
	//	"syscall"
)

type Credentials struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

var USERS_TO_TRACK [1]string = [1]string{"1331444879893942272"}

var TwitterApi string = "https://api.twitter.com/1.1/"

func main() {
	fmt.Println("James v0.01")

	// Get credentials for twitter create a client
	creds := Credentials{
		ConsumerKey:       os.Getenv("CONSUMER_KEY"),
		ConsumerSecret:    os.Getenv("CONSUMER_SECRET"),
		AccessToken:       os.Getenv("ACCESS_TOKEN"),
		AccessTokenSecret: os.Getenv("ACCESS_TOKEN_SECRET"),
	}

	_, err := getClient(&creds)
	if err != nil {
		log.Println("Error getting Twitter Client")
		log.Println(err)
	}

	// Should have a similar setup for handling streams
	//	// Begin handling messages from the stream
	//	go demux.HandleChan(stream.Messages)
	//
	//	// Wait for SIGING and SIGTERM (ctrl-c)
	//	ch := make(chan os.Signal)
	//	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	//	log.Println(<-ch)
	//
	//	fmt.Println("Stopping stream...")
	//	stream.Stop()
}

// getClient is a helper function that will allow
// us to stream tweets. It takes in a credentials struct
// pointer for authentication.
func getClient(creds *Credentials) (*http.Client, error) {
	// Credentials for the developer account
	config := oauth1.NewConfig(creds.ConsumerKey, creds.ConsumerSecret)
	// Credentials for the authenticated user
	token := oauth1.NewToken(creds.AccessToken, creds.AccessTokenSecret)

	httpClient := config.Client(oauth1.NoContext, token)

	// we can retrieve the user and verify if the credentials
	// we have used successfullly allow us to log in
	VerifyCredentials(httpClient)

	return httpClient, nil
}

func VerifyCredentials(client *http.Client) {
	resp, err := client.Get(TwitterApi + "account/verify_credentials.json")
	check(err)
	fmt.Println(resp)
}

// For checking errors more easily
func check(e error) {
	if e != nil {
		panic(e)
	}
}
