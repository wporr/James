package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dghubble/oauth1"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
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

type Event struct {
	TweetCreateEvents []Tweet `json:"tweet_create_events"`
}

type User struct {
	ID int64 `json:"id"`
}

type Entity struct {
	UserMentions []User `json:"user_mentions"`
}

type Tweet struct {
	CreatedAt       string  `json:"created_at"`
	Text            string  `json:"text"`
	User            User    `json:"user"`
	Entities        Entity  `json:"entities"`
	RetweetedStatus Retweet `json:"retweeted_status"`
	ID              int64   `json:"id"`
}

// Retweets are actually the same as tweets, but
// the struct is only used to test if a tweet is
// indeed a retweet, so we only store some values
type Retweet struct {
	CreatedAt string `json:"created_at"`
	ID        int    `json:"id"`
}

/* GLOBAL VARIABLES */

var ENV_NAME string = "AccountActivity"
var WEBHOOK_URL string = "https://alamo.ocf.berkeley.edu/webhook/twitter"
var JAMES User = User{
	ID: 1305226572564062208,
}

var TwitterApi url.URL = url.URL{
	Scheme: "https",
	Host:   "api.twitter.com",
	Path:   "/1.1",
}

// Tweets are 280 chars max. GPT-3 output is measured
// in tokens, which are roughly 4 english chars in length.
// So to make sure we stay under the limit, we went a bit
// lower than the tweet char max, from 280/4 to 220/4, i.e. 55
var MAX_TWEET_TOKENS int = 55

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
		log.Printf("webhook question response reqeust received")
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
		fmt.Println("Event received")
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)

		// This is not all thats returned in the body, but
		// we only store the things we need to know, keeping things lean
		resp := Event{}
		check(json.Unmarshal([]byte(body), &resp))

		if isMention(&resp) {
			check(postReply(resp.TweetCreateEvents[0]))
		} else {
			fmt.Printf("not mention")
		}
	}
}

func postReply(t Tweet) error {
	creds := Credentials{
		ConsumerKey:       os.Getenv("CONSUMER_KEY"),
		ConsumerSecret:    os.Getenv("CONSUMER_SECRET"),
		AccessToken:       os.Getenv("ACCESS_TOKEN"),
		AccessTokenSecret: os.Getenv("ACCESS_TOKEN_SECRET"),
	}

	client, err := getClient(&creds)

	statusUpdateEndpoint := TwitterApi
	statusUpdateEndpoint.Path = statusUpdateEndpoint.Path + "/" +
		url.PathEscape("statuses") + "/" +
		url.PathEscape("update.json")

	// Create the request for a text completion from GPT-3
	// TODO: Determine template by reading the status of the
	// mention and matching it to some template
	responseChan := make(chan CompletionReponse, 1)
	req := CompletionRequest{
		Prompt:       t.Text,
		ResponseChan: responseChan,
		Model:        Davinci,
		Template:     *StandardTmpl,
		Temperature:  0.9,
		Tokens:       MAX_TWEET_TOKENS,
	}

	JamesBuffer <- req

	// Wait for the completion and use it to create the tweet reply
	resp := <-responseChan
	check(resp.Err)

	// Tweets will only be registered as a response if the
	// "in_reply_to_status_id" parameter is set to the tweet that
	// is being responded to AND if the reponse itself contains a
	// mention of the user that created the original tweet
	query := url.Values{}
	query.Set("status", resp.Response)
	query.Set("in_reply_to_status_id", strconv.FormatInt(t.ID, 10))
	statusUpdateEndpoint.RawQuery = query.Encode()

	apiResp, err := client.Post(statusUpdateEndpoint.String(), "application/json", nil)

	body, err := ioutil.ReadAll(apiResp.Body)
	return err
}

// To differentiate a mention from other tweets is
// incredibly annoying. You can check by seeing if
// the user who made the tweet is different from
// the authenticated account, and by checking the
// mentions list in the entities key
func isMention(event *Event) bool {
	if len(event.TweetCreateEvents) == 0 {
		return false
	} else if t := event.TweetCreateEvents[0]; t.User.ID != JAMES.ID &&
		contains(t.Entities.UserMentions, JAMES) &&
		t.RetweetedStatus == (Retweet{}) {
		return true
	}
	return false
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
	req.Header.Set("authorization", "Bearer "+os.Getenv("BEARER_TOKEN"))
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
		log.Println("Registered webhook found. Reusing")
		var getResp = new([]Webhook)
		err = json.Unmarshal([]byte(body), &getResp)
		check(err)
		*w = (*getResp)[0]
	}

	// Subscribe to account activity for the requesting user in the
	// environment ENV_NAME. Max of 15 users per application in free tier.
	// Events are sent to the webhooks registered by the user `client`
	check(subscribe(client, ENV_NAME))
	//	fmt.Println("deleting webhook")
	//	check(deleteWebhook(w.ID, client))

}

func subscribe(client *http.Client, envName string) error {
	endpt := TwitterApi
	endpt.Path = endpt.Path + "/" +
		url.PathEscape("account_activity") + "/" +
		url.PathEscape("all") + "/" +
		url.PathEscape(envName) + "/" +
		url.PathEscape("subscriptions"+".json")

	// Check if the subscription exists first
	resp, err := client.Get(endpt.String())
	check(err)

	if resp.StatusCode == 204 {
		return nil
	}

	// Subscription doesnt exist, create it
	log.Println("Creating subscription")
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
		url.PathEscape(webhookID+".json")

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

func contains(list []User, user User) bool {
	for _, u := range list {
		if user.ID == u.ID {
			return true
		}
	}
	return false
}
