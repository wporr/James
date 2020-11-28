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
    ID        string
    URL       string
    Valid     bool
    CreatedAt string
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

    postURL := TwitterApi
    postURL.Path = postURL.Path + "/" +
        url.PathEscape("account_activity") + "/" +
        url.PathEscape("all") + "/" +
        url.PathEscape(ENV_NAME) + "/" +
        url.PathEscape("webhooks.json")

    query := url.Values{}
    query.Set("url", WEBHOOK_URL)
    postURL.RawQuery = query.Encode()

    resp, err := client.Post(postURL.String(), "application/json", nil)
    check(err)

    body, err := ioutil.ReadAll(resp.Body)
    check(err)
    fmt.Println(string(body))

    var w = new(Webhook)
    err = json.Unmarshal([]byte(body), &w)
    check(err)

    fmt.Println(w)
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
