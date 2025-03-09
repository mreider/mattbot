package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/spf13/cobra"
)

func run(cmd *cobra.Command, args []string) {
	// Retrieve the phone number from Redis
	phoneNumber, err := GetPhoneNumber(context.Background())
	if err != nil {
		logger.Fatalf("Failed to retrieve phone number from Redis: %v.  Did you run 'mattbot init'?", err)
	}

	// Retrieve the Claude API key from Redis
	claudeAPIKey, err := GetClaudeAPIKey(context.Background())
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

				// Extract user identifier (example: using the sender's phone number)
				userIdentifier := extractUserIdentifier(lastMessage) // Implement this function

				// Use Claude to parse the message
				eventDetails, err := parseEventDetails(messageContent, claudeAPIKey)
				if err != nil {
					logger.Errorf("Failed to parse event details: %v", err)
					respondWithError(ctx, "Sorry, I couldn't understand the event details. Please provide all information (title, date, time, duration, recurrence) in a clear format.")
					continue
				}

				// Check if all required details are present
				if eventDetails.Title == "" || eventDetails.Date == "" || eventDetails.Time == "" || eventDetails.Duration == "" {
					respondWithError(ctx, "Sorry, I need the event title, date, time, and duration to create the event.")
					continue
				}

				// Store the event in Redis
				err = storeEvent(ctx, userIdentifier, eventDetails)
				if err != nil {
					logger.Errorf("Failed to store event in Redis: %v", err)
					respondWithError(ctx, "Sorry, I couldn't store the event. Please try again.")
					continue
				}

				// Print the event details to the console
				fmt.Printf("Event Title: %s\n", eventDetails.Title)
				fmt.Printf("Event Date: %s\n", eventDetails.Date)
				fmt.Printf("Event Time: %s\n", eventDetails.Time)
				fmt.Printf("Event Duration: %s\n", eventDetails.Duration)
				fmt.Printf("Event Recurrence: %s\n", eventDetails.Recurrence)

				// Respond to the user with a confirmation message
				respondWithSuccess(ctx, fmt.Sprintf("Okay, I've noted the event: %s on %s at %s for %s. Recurrence: %s", eventDetails.Title, eventDetails.Date, eventDetails.Time, eventDetails.Duration, eventDetails.Recurrence))

			}
		}
	}
}

func extractUserIdentifier(message string) string {
	// This is a placeholder.  In a real application, you would need to
	// extract the user's identifier from the WhatsApp message.  This might
	// involve parsing the message structure or using a WhatsApp API to
	// identify the sender.
	// For this example, we'll just return a placeholder.
	return "user123"
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
