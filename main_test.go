package main

import (
	"fmt"
	"testing"
)

var testResponseChan = make(chan CompletionReponse, 1)

var testRequest = CompletionRequest{
	Prompt:       "tell me a joke",
	ResponseChan: testResponseChan,
	Model:        Davinci,
	Template:     *StandardTmpl,
	Temperature:  0.7,
}

func TestBasicCompletion(t *testing.T) {
	buf := make(chan CompletionRequest, 1)
	go runCompletions(buf)

	buf <- testRequest

	close(buf)
	fmt.Printf((<-testResponseChan).Response)
}
