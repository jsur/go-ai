package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/liushuangls/go-anthropic/v2"
)

type AIService struct {
	client *anthropic.Client
}

func NewAIService(apiKey string) *AIService {
	return &AIService{
		client: anthropic.NewClient(strings.TrimSpace(apiKey)),
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Please give Anthropic api key: ")
	apikey, _ := reader.ReadString('\n')

	service := NewAIService(apikey)

	isValid := service.validateApiKey()

	if !isValid {
		log.Fatal("Given api key is not valid")
	}

	fmt.Print("Give prompt: ")
	text, _ := reader.ReadString('\n')

	service.promptAI(text)
}

func (s *AIService) validateApiKey() bool {
	_, err := s.client.CreateMessages(context.Background(), anthropic.MessagesRequest{
		Model: anthropic.ModelClaude3Sonnet20240229,
		Messages: []anthropic.Message{
			anthropic.NewUserTextMessage("a"),
		},
		MaxTokens: 1,
	})

	if err != nil {
		var e *anthropic.APIError
		if errors.As(err, &e) {
			if e.IsAuthenticationErr() {
				return false
			}
		}
		log.Fatal("Error validating api key")
	}

	return true
}

func (s *AIService) promptAI(prompt string) {
	resp, err := s.client.CreateMessages(context.Background(), anthropic.MessagesRequest{
		Model: anthropic.ModelClaude3Sonnet20240229,
		Messages: []anthropic.Message{
			anthropic.NewUserTextMessage(prompt),
		},
		MaxTokens: 1000,
	})

	if err != nil {
		var e *anthropic.APIError
		if errors.As(err, &e) {
			fmt.Printf("Messages error, type: %s, message: %s", e.Type, e.Message)
		} else {
			fmt.Printf("Messages error: %v\n", err)
		}
		return
	}

	fmt.Println(resp.Content[0].GetText())

}
