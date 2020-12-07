package main

import (
	"bytes"
	"context"
	"errors"
	gogpt "github.com/sashabaranov/go-gpt3"
	"log"
	"os"
	"strings"
	"text/template"
)

// Tweets are 280 chars max. GPT-3 output is measured
// in tokens, which are roughly 4 english chars in length.
// So to make sure we stay under the limit, we went a bit
// lower than the tweet char max, from 280/4 to 220/4, i.e. 55
var MAX_TWEET_TOKENS int = 55

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

			resp, err := c.CreateCompletion(ctx, request.Model.String(), req)
			if err != nil {
				return
			}

			request.ResponseChan <- CompletionReponse{
				Response: resp.Choices[0].Text,
				Err:      nil,
			}
		}
		close(request.ResponseChan)
	}
}
