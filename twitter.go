package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
    "bytes"
    "errors"
    "io/ioutil"
	"github.com/dghubble/oauth1"
	"net/http"
    "encoding/base64"
    "crypto/sha256"
    "crypto/hmac"
	//	"os/signal"
	//	"syscall"
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

type CRCReponse struct {
    ResponseToken string `json:"response_token"`
}

var USERS_TO_TRACK [1]string = [1]string{"1331444879893942272"}
var ENV_NAME string = "AccountActivity"
var WEBHOOK_URL string = "http://alamo.ocf.berkeley.edu/webhook/twitter"

var TwitterApi string = "https://api.twitter.com/1.1/"

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
        resp := CRCReponse{
            ResponseToken: "sha256=" + responseToken,
        }

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

    postURL := TwitterApi + "account_activity/all/" + ENV_NAME +
        "/webhooks.json?url=https%3A%2F%2Falamo.ocf.berkeley.edu%2Fwebhook%2Ftwitter"
    postBodyJson, _ := json.Marshal(map[string]string{
        "url": WEBHOOK_URL,
    })
    postBody := bytes.NewBuffer(postBodyJson)

    resp, err := client.Post(postURL, "application/json", postBody)
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
	_, err := client.Get(TwitterApi + "account/verify_credentials.json")
	check(err)
}
