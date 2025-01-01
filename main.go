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
	"golang.org/x/term"
)

type AIService struct {
	client *anthropic.Client
}

const API_KEY_PATH = "/tmp/cmdlineai_apikey"

func NewAIService(apiKey string) *AIService {
	return &AIService{
		client: anthropic.NewClient(strings.TrimSpace(apiKey)),
	}
}

func main() {
	var apiKey string

	storedKey, storedKeyErr := checkApiKey()

	if storedKeyErr == nil {
		apiKey = storedKey
		fmt.Printf("Using api key \"%s...\" from disk\n", apiKey[0:20])
	} else {
		fmt.Print("Please give Anthropic api key: ")
		input, _ := term.ReadPassword(int(os.Stdin.Fd()))
		apiKey = string(input)
		fmt.Print("\n")
	}

	service := NewAIService(apiKey)

	service.validateApiKey()

	// TODO: tarviiko tässä?
	reader := bufio.NewReader(os.Stdin)

	if storedKey == "" {
		service.persistApiKey(reader, apiKey)
	}

	service.promptAI(reader)
}

func checkApiKey() (string, error) {
	contents, err := os.ReadFile(API_KEY_PATH)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

func (s *AIService) validateApiKey() {
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
				log.Fatal("Given api key is not valid")
			}
		}
		log.Fatal("Error validating api key")
	}
}

func (s *AIService) persistApiKey(reader *bufio.Reader, key string) {
	fmt.Print("Api key is valid. Would you like to save it for next time? y/n\n")

	answer, _ := reader.ReadString('\n')

	if strings.TrimSpace(answer) != "y" {
		fmt.Print("Not saving api key.")
		return
	}

	error := os.WriteFile(API_KEY_PATH, []byte(key), 0644)

	if error != nil {
		fmt.Print("Failed to save api key, continuing..")
	}

	fmt.Print("Api key saved.")

}

func (s *AIService) promptAI(reader *bufio.Reader) {
	fmt.Print("Ask Claude something: ")
	prompt, _ := reader.ReadString('\n')

	_, err := s.client.CreateMessagesStream(context.Background(), anthropic.MessagesStreamRequest{
		MessagesRequest: anthropic.MessagesRequest{
			Model: anthropic.ModelClaude3Dot5SonnetLatest,
			Messages: []anthropic.Message{
				anthropic.NewUserTextMessage(prompt),
			},
			MaxTokens: 1000,
		},
		OnContentBlockDelta: func(data anthropic.MessagesEventContentBlockDeltaData) {
			fmt.Print(*data.Delta.Text)
		},
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

}
