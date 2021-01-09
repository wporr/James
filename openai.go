package main

import (
	"bytes"
	"context"
	"errors"
	gogpt "github.com/sashabaranov/go-gpt3"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// Tweets are 280 chars max. GPT-3 output is measured
// in tokens, which are roughly 4 english chars in length.
// So to make sure we stay under the limit, we went a bit
// lower than the tweet char max, from 280/4 to 220/4, i.e. 55
var MAX_TWEET_TOKENS int = 55

// Number of times we'll retry generating a prompt thats unsafe
// before giving up
var MAX_COMPLETION_RETRIES int = 5

// Default response when we reach max retries
var DEFAULT_RESPONSE string = "*Yaaaawn*... eh, I dont really feel like it"

type CompletionRequest struct {
	Prompt       string
	ResponseChan chan CompletionReponse
	Model        ModelEnum
	Template     template.Template
	Temperature  float32
}

type CompletionReponse struct {
	Response string
	Err      error
}

type ModelEnum struct{ *string }

func (e ModelEnum) String() string {
	if e.string == nil {
		return "<void>"
	}
	return *e.string
}

func (e ModelEnum) IsValid() bool {
	for _, m := range []ModelEnum{Ada, Babbage, Curie, Davinci} {
		if m == e {
			return true
		}
	}
	return false
}

// Not a great way to do enums in golang
var (
	es = []string{"ada", "babbage", "curie", "davinci"}

	Ada     = ModelEnum{&es[0]}
	Babbage = ModelEnum{&es[1]}
	Curie   = ModelEnum{&es[2]}
	Davinci = ModelEnum{&es[3]}
)

func runCompletions(buffer chan CompletionRequest) {
	c := gogpt.NewClient(os.Getenv("OPENAI_API_KEY"))
	ctx := context.Background()

	for {
		request, more := <-buffer

		// If the buffer is closed, kill this goroutine
		if !more {
			log.Printf("Request buffer closed. Closing completion backend")
			return
		} else if !request.Model.IsValid() {
			// Make sure we have a valid model requested
			request.ResponseChan <- CompletionReponse{
				Response: "",
				Err: errors.New("Requested invalid model: " +
					request.Model.String()),
			}
		} else {
			// Newlines can trip up GPT-3
			request.Prompt = strings.ReplaceAll(request.Prompt, "\n", " ")

			prompt := new(bytes.Buffer)
			check(request.Template.Execute(prompt, request))

			req := gogpt.CompletionRequest{
				MaxTokens:   MAX_TWEET_TOKENS,
				Prompt:      prompt.String(),
				Temperature: request.Temperature,
			}

			try := true
			respText := ""
			retries := 0

			for try {
				resp, err := c.CreateCompletion(ctx, request.Model.String(), req)
				if err != nil {
					return
				}

				respText = resp.Choices[0].Text

				sensitivity, err := checkSensitivity(respText, ctx, c)
				check(err)

				// Safe is 0, sensitive is 1, unsafe is 2
				if sensitivity < 2 {
					try = false
				} else if retries >= MAX_COMPLETION_RETRIES {
					respText = DEFAULT_RESPONSE
					log.Printf("Max retries reached for prompt: %v", req.Prompt)
					break
				} else {
					retries++
				}
			}

			filteredText := filterResponse(respText)

			request.ResponseChan <- CompletionReponse{
				Response: filteredText,
				Err:      nil,
			}
		}
		close(request.ResponseChan)
	}
}

func filterResponse(text string) string {
	// Regex to match the beginning of text we want to remove
	// If the ai tries to provide the user's response to it's response,
	// we'll remove it
	re := regexp.MustCompile(`\n[a-zA-z0-9]+:`)
	indexes := re.FindStringIndex(text)

	if indexes != nil {
		return text[:indexes[0]]
	}
	return text
}

func checkSensitivity(text string, ctx context.Context, c *gogpt.Client) (int, error) {
	req := gogpt.CompletionRequest{
		MaxTokens:   1,
		Prompt:      "<|endoftext|>" + text + "\n--\nLabel:",
		Temperature: 0.0,
		TopP:        0,
	}

	resp, err := c.CreateCompletion(ctx, "content-filter-alpha-c4", req)
	check(err)

	sensitivity, err := strconv.Atoi(resp.Choices[0].Text)
	check(err)

	return sensitivity, nil
}
