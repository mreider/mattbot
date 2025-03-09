package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	anthropic "github.com/anthropics/anthropic-go/v2"
	"github.com/chromedp/chromedp"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	logger      = logrus.New()
	redisClient *redis.Client
	phoneNumber string
)

const (
	redisKeyPhoneNumber  = "whatsapp_phone_number"
	redisKeyClaudeAPIKey = "claude_api_key"
)

func init() {
	// Set up logger
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Initialize Redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Update if your Redis is running elsewhere
		Password: "",               // No password by default
		DB:       0,                // Default DB
	})
}

func main() {
	Execute()
}

func initialize(cmd *cobra.Command, args []string) {
	var input string
	fmt.Print("Enter your WhatsApp phone number (including country code, without + or leading zeros): ")
	fmt.Scanln(&input)

	// Generate the WhatsApp direct link
	whatsappURL := fmt.Sprintf("https://wa.me/%s", input)

	// Check if the link is reachable
	resp, err := http.Head(whatsappURL)
	if err != nil {
		logger.Fatalf("Failed to check WhatsApp link: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Fatalf("WhatsApp link is not reachable. Status code: %d", resp.StatusCode)
	}

	// Store the phone number in Redis
	err = redisClient.Set(context.Background(), redisKeyPhoneNumber, input, 0).Err()
	if err != nil {
		logger.Fatalf("Failed to store phone number in Redis: %v", err)
	}

	logger.Info("Phone number stored in Redis.")

	fmt.Printf("WhatsApp Direct Link: %s\n", whatsappURL)
	logger.Infof("WhatsApp Direct Link: %s", whatsappURL)

	// Ask for Claude API key
	fmt.Print("Enter your Claude API key: ")
	fmt.Scanln(&input)

	// Validate Claude API key
	client := anthropic.NewClient(input)

	_, err = client.Messages.Create(context.Background(), &anthropic.MessageCreateParams{
		Model:     anthropic.ModelClaude3Opus20240229,
		MaxTokens: 10,
		Messages: []anthropic.MessageParam{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
	})

	if err != nil {
		logger.Fatalf("Failed to validate Claude API key: %v", err)
	}

	// Store Claude API key in Redis
	err = redisClient.Set(context.Background(), redisKeyClaudeAPIKey, input, 0).Err()
	if err != nil {
		logger.Fatalf("Failed to store Claude API key in Redis: %v", err)
	}

	logger.Info("Claude API key stored in Redis.")
}

func run(cmd *cobra.Command, args []string) {
	// Retrieve the phone number from Redis
	phoneNumber, err := redisClient.Get(context.Background(), redisKeyPhoneNumber).Result()
	if err != nil {
		logger.Fatalf("Failed to retrieve phone number from Redis: %v.  Did you run 'mattbot init'?", err)
	}

	// Retrieve the Claude API key from Redis
	claudeAPIKey, err := redisClient.Get(context.Background(), redisKeyClaudeAPIKey).Result()
	if err != nil {
		logger.Fatalf("Failed to retrieve Claude API key from Redis: %v. Did you run 'mattbot init'?", err)
	}

	// Run Chrome
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	err = chromedp.Run(ctx,
		chromedp.Navigate("https://web.whatsapp.com"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("Please scan the QR code in the WhatsApp Web interface.")
			fmt.Println("Waiting for WhatsApp Web to load...")
			return nil
		}),
		chromedp.Sleep(15*time.Second), // Wait for QR code to be scanned and WhatsApp to load
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("Listening for @ mentions...")
			return nil
		}),
		chromedp.ActionFunc(listenForMentions(ctx, claudeAPIKey)), // Pass the Claude API key
	)

	if err != nil {
		logger.Fatalf("Failed to run: %v", err)
	}
}

func listenForMentions(ctx context.Context, claudeAPIKey string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		for {
			time.Sleep(5 * time.Second) // Check every 5 seconds

			var lastMessage string
			err := chromedp.Run(ctx,
				// This selector needs to be adjusted to target the latest message content
				chromedp.Text(`span[dir="auto"]`, &lastMessage, chromedp.ByQueryAll, chromedp.At(0)),
			)
			if err != nil {
				logger.Errorf("Failed to retrieve last message: %v", err)
				continue
			}

			if strings.Contains(lastMessage, "@") {
				// Extract the message content after the "@" mention
				mentionIndex := strings.Index(lastMessage, "@")
				messageContent := lastMessage[mentionIndex+1:]

				// Use Claude to parse the message
				eventDetails, err := parseEventDetails(messageContent, claudeAPIKey)
				if err != nil {
					logger.Errorf("Failed to parse event details: %v", err)
					respondWithError(ctx, "Sorry, I couldn't understand the event details. Please provide all information (title, date, time, duration) in a clear format.")
					continue
				}

				// Check if all required details are present
				if eventDetails.Title == "" || eventDetails.Date == "" || eventDetails.Time == "" || eventDetails.Duration == "" {
					respondWithError(ctx, "Sorry, I need the event title, date, time, and duration to create the event.")
					continue
				}

				// Print the event details to the console
				fmt.Printf("Event Title: %s\n", eventDetails.Title)
				fmt.Printf("Event Date: %s\n", eventDetails.Date)
				fmt.Printf("Event Time: %s\n", eventDetails.Time)
				fmt.Printf("Event Duration: %s\n", eventDetails.Duration)

				// Respond to the user with a confirmation message
				respondWithSuccess(ctx, fmt.Sprintf("Okay, I've noted the event: %s on %s at %s for %s.", eventDetails.Title, eventDetails.Date, eventDetails.Time, eventDetails.Duration))

			}
		}
	}
}

type EventDetails struct {
	Title    string
	Date     string
	Time     string
	Duration string
}

func parseEventDetails(message string, claudeAPIKey string) (EventDetails, error) {
	client := anthropic.NewClient(claudeAPIKey)

	resp, err := client.Messages.Create(context.Background(), &anthropic.MessageCreateParams{
		Model:     anthropic.ModelClaude3Opus20240229,
		MaxTokens: 200,
		Messages: []anthropic.MessageParam{
			{
				Role:    "user",
				Content: fmt.Sprintf("Extract the event title, date, time, and duration from this text: %s. Respond in a JSON format like this: {\"title\": \"event title\", \"date\": \"event date\", \"time\": \"event time\", \"duration\": \"event duration\"}. If any information is missing, leave the field empty.", message),
			},
		},
	})
	if err != nil {
		return EventDetails{}, err
	}

	// Extract the JSON response from Claude
	var jsonResponse string
	if len(resp.Content) > 0 {
		jsonResponse = resp.Content[0].Text
	}

	// Unmarshal the JSON response into the EventDetails struct
	var eventDetails EventDetails
	err = json.Unmarshal([]byte(jsonResponse), &eventDetails)
	if err != nil {
		return EventDetails{}, fmt.Errorf("failed to unmarshal JSON: %w, raw response: %s", err, jsonResponse)
	}

	return eventDetails, nil
}

func respondWithError(ctx context.Context, message string) {
	err := chromedp.Run(ctx,
		// This selector needs to be adjusted to target the message input box
		chromedp.SendKeys(`div[title="Type a message"]`, message, chromedp.NodeVisible),
		chromedp.Click(`button[aria-label="Send"]`, chromedp.NodeVisible),
	)
	if err != nil {
		logger.Errorf("Failed to send error message: %v", err)
	}
}

func respondWithSuccess(ctx context.Context, message string) {
	err := chromedp.Run(ctx,
		// This selector needs to be adjusted to target the message input box
		chromedp.SendKeys(`div[title="Type a message"]`, message, chromedp.NodeVisible),
		chromedp.Click(`button[aria-label="Send"]`, chromedp.NodeVisible),
	)
	if err != nil {
		logger.Errorf("Failed to send success message: %v", err)
	}
}
