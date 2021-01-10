package main

import (
	"context"
	"fmt"
	gogpt "github.com/sashabaranov/go-gpt3"
	"os"
	"testing"
	"text/template"
)

func TestBasicCompletion(t *testing.T) {
	var testResponseChan = make(chan CompletionReponse, 1)
	req := CompletionRequest{
		Prompt:       "tell me a joke",
		ResponseChan: testResponseChan,
		Model:        Ada,
		Template:     *StandardTmpl,
		Temperature:  0.7,
		Tokens:       55,
	}

	buf := make(chan CompletionRequest, 1)
	go runCompletions(buf)

	buf <- req

	close(buf)
	fmt.Printf((<-testResponseChan).Response)
}

func TestSensitivity(t *testing.T) {
	c := gogpt.NewClient(os.Getenv("OPENAI_API_KEY"))
	ctx := context.Background()

	text := os.Getenv("UNSAFE_PROMPT")
	rating, err := checkSensitivity(text, ctx, c)

	if err != nil {
		t.Errorf("Error when checking sensitivity: %v", err)
	} else if rating != 2 {
		t.Errorf("Sensitivity for unsafe prompt was rated: %v", rating)
	}
}

func TestSensitivityRetries(t *testing.T) {
	var testResponseChan = make(chan CompletionReponse, 1)
	dummyTempl, _ := template.New("dummy").Parse(`{{.Prompt}}`)

	req := CompletionRequest{
		Prompt:       os.Getenv("UNSAFE_PROMPT"),
		ResponseChan: testResponseChan,
		Model:        Ada,
		Template:     *dummyTempl,
		Temperature:  0.1,
	}
	buf := make(chan CompletionRequest, 1)
	go runCompletions(buf)

	buf <- req

	close(buf)
	resp := (<-testResponseChan).Response

	if resp != DEFAULT_RESPONSE {
		t.Errorf("Unsafe response did not get filtered out, response %v", resp)
	}
}

func testFilterDoubleResp(t *testing.T) {
	txtJames := "James: random response "
	txtLiam := "\nLiam: remove me"

	filtered := filterResponse(txtJames + txtLiam)

	if filtered != txtJames {
		t.Errorf("Text did not get filtered properly. Filtered text: %v", filtered)
	}
}
