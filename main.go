package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/liushuangls/go-anthropic/v2"
	"github.com/manifoldco/promptui"
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

func onShutdown() {
	var signalChan chan (os.Signal) = make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	fmt.Print("\nExiting...\n")
	os.Exit(0)
}

func main() {
	go onShutdown()

	var apiKey string

	storedKey, storedKeyErr := checkApiKey()

	if storedKeyErr == nil {
		apiKey = storedKey
		fmt.Printf("Using api key \"%s...\" from disk.\n", apiKey[0:20])
	} else {
		fmt.Print("Please give Anthropic api key: ")
		input, _ := term.ReadPassword(int(os.Stdin.Fd()))
		apiKey = string(input)
		fmt.Print("\n")
	}

	service := NewAIService(apiKey)

	service.validateApiKey()

	if storedKey == "" {
		service.persistApiKey(apiKey)
	}

	service.showMenu()
}

func (s *AIService) showMenu() {
	items := []string{"Ask Claude", "Exit"}

	if _, err := checkApiKey(); err == nil {
		items = append(items, "Clear api key")
	}

	prompt := promptui.Select{
		Items: items,
		Label: "Select action",
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	switch result {
	case "Ask Claude":
		fmt.Print("Ask Claude something: ")
		reader := bufio.NewReader(os.Stdin)
		prompt, _ := reader.ReadString('\n')
		s.promptAI(prompt)
	case "Clear api key":
		clearApiKey()
		s.showMenu()
	case "Exit":
		fmt.Print("Exiting...\n")
		os.Exit(0)
	}
}

func checkApiKey() (string, error) {
	contents, err := os.ReadFile(API_KEY_PATH)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

func clearApiKey() {
	err := os.Remove(API_KEY_PATH)
	if err != nil {
		log.Fatal("Error clearing api key")
	}
	fmt.Print("Api key cleared.\n")
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

func (s *AIService) persistApiKey(key string) {
	fmt.Print("Api key is valid. Would you like to save it for next time? y/n\n")

	reader := bufio.NewReader(os.Stdin)
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

func (s *AIService) promptAI(prompt string) {
	if strings.TrimSpace(prompt) == "menu" {
		s.showMenu()
		return
	}

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
		OnMessageStop: func(memsd anthropic.MessagesEventMessageStopData) {
			fmt.Print("\n\n")
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

	reader := bufio.NewReader(os.Stdin)
	p2, _ := reader.ReadString('\n')

	s.promptAI(p2)
}
