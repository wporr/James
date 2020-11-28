package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
    "errors"
    "io/ioutil"
	"github.com/dghubble/oauth1"
	"net/http"
    "net/url"
    "encoding/base64"
    "crypto/sha256"
    "crypto/hmac"
)

type Credentials struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

type Webhook struct {
    ID               string `json:"id"`
    URL              string `json:"url"`
    Valid            bool   `json:"valid"`
    CreatedTimestamp string `json:"created_timestamp"`
}

/* GLOBAL VARIABLES */
var USERS_TO_TRACK [1]string = [1]string{"1331444879893942272"}
var ENV_NAME string = "AccountActivity"
var WEBHOOK_URL string = "https://alamo.ocf.berkeley.edu/webhook/twitter"

var TwitterApi url.URL = url.URL {
    Scheme: "https",
    Host: "api.twitter.com",
    Path: "/1.1",
}
/* --------------- */

func generateResponseToken(token []byte) string {
    mac := hmac.New(sha256.New, []byte(os.Getenv("CONSUMER_SECRET")))
    mac.Write(token)
    tokenBytes := mac.Sum(nil)

    return base64.StdEncoding.EncodeToString(tokenBytes)
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    switch method := r.Method; method {
    case "GET":
        crcToken, ok := r.URL.Query()["crc_token"]
        if !ok {
            panic(errors.New("Couldnt get crc_token"))
        }

        responseToken := generateResponseToken([]byte(crcToken[0]))
        resp := struct {
            ResponseToken string `json:"response_token"`
        }{ResponseToken: "sha256=" + responseToken}

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    case "POST":
        fmt.Println("Event received\n")
        fmt.Println(r)
    }
}


func registerWebhook() {
	// Get credentials for twitter create a client
	creds := Credentials{
		ConsumerKey:       os.Getenv("CONSUMER_KEY"),
		ConsumerSecret:    os.Getenv("CONSUMER_SECRET"),
		AccessToken:       os.Getenv("ACCESS_TOKEN"),
		AccessTokenSecret: os.Getenv("ACCESS_TOKEN_SECRET"),
	}

	client, err := getClient(&creds)
	if err != nil {
		log.Println("Error getting Twitter Client")
		log.Println(err)
	}

    webhkEndpt := TwitterApi
    webhkEndpt.Path = webhkEndpt.Path + "/" +
        url.PathEscape("account_activity") + "/" +
        url.PathEscape("all") + "/" +
        url.PathEscape(ENV_NAME) + "/" +
        url.PathEscape("webhooks.json")

    // First try GET to see if we already have registered webhooks.
    // Otherwise, POST to register one
    // We use an app client to get all the webhooks registered for this app,
    // not just the current user context (represented by `client`)
    appClient := &http.Client{}
    req, _ := http.NewRequest("GET", webhkEndpt.String(), nil)
    req.Header.Set("authorization", "Bearer " + os.Getenv("BEARER_TOKEN"))
    resp, err := appClient.Do(req)
    defer resp.Body.Close()
    check(err)
    body, _ := ioutil.ReadAll(resp.Body)
    var w = new(Webhook)

    // If true, no webhook is registered, make one
    if string(body) == "[]" {
        log.Println("No registered webhooks found, registering new one")
        query := url.Values{}
        query.Set("url", WEBHOOK_URL)
        webhkEndpt.RawQuery = query.Encode()

        resp, err := client.Post(webhkEndpt.String(), "application/json", nil)
        check(err)

        body, _ = ioutil.ReadAll(resp.Body)
        defer resp.Body.Close()
        check(json.Unmarshal([]byte(body), &w))
    } else {
        var getResp = new([]Webhook)
        err = json.Unmarshal([]byte(body), &getResp)
        check(err)
        *w = (*getResp)[0]
    }

    check(subscribe(client, ENV_NAME))

//    fmt.Println("deleting webhook")
//    check(deleteWebhook(w[0].ID, client))
}

func subscribe(client *http.Client, envName string) error {
    endpt := TwitterApi
    endpt.Path = endpt.Path + "/" +
        url.PathEscape("account_activity") + "/" +
        url.PathEscape("all") + "/" +
        url.PathEscape(envName) + "/" +
        url.PathEscape("subscriptions" + ".json")

    // Check if the subscription exists first
    resp, err := client.Get(endpt.String())
    check(err)

    if resp.StatusCode == 204 {
        return nil
    }

    // Subscription doesnt exist, create it
    resp, err = client.Post(endpt.String(), "text/plain", nil)
    check(err)

    if resp.StatusCode != 204 {
        defer resp.Body.Close()
        response, _ := ioutil.ReadAll(resp.Body)
        return errors.New("Could not subscribe environment " +
            envName + "\nResponse: " + string(response))
    }

    // This will check to make sure the subscription POST
    // was processed correctly (GET should return 204)
    return subscribe(client, envName)
}

func deleteWebhook(webhookID string, client *http.Client) error {
    endpt := TwitterApi
    endpt.Path = endpt.Path + "/" +
        url.PathEscape("account_activity") + "/" +
        url.PathEscape("all") + "/" +
        url.PathEscape(ENV_NAME) + "/" +
        url.PathEscape("webhooks") + "/" +
        url.PathEscape(webhookID + ".json")

    req, err := http.NewRequest("DELETE", endpt.String(), nil)
    check(err)
    _, err = client.Do(req)
    return err
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
    verifyURL := TwitterApi
    verifyURL.Path = verifyURL.Path + "/" +
        url.PathEscape("account") + "/" +
        url.PathEscape("verify_credentials.json")
	_, err := client.Get(verifyURL.String())
	check(err)
}
