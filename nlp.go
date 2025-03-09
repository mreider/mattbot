package main

import (
	"context"
	"encoding/json"
	"fmt"

	anthropic "github.com/anthropics/anthropic-go/v2"
)

type EventDetails struct {
	Title      string
	Date       string
	Time       string
	Duration   string
	Recurrence string // e.g., "annually", "none"
}

func parseEventDetails(message string, claudeAPIKey string) (EventDetails, error) {
	client := anthropic.NewClient(claudeAPIKey)

	resp, err := client.Messages.Create(context.Background(), &anthropic.MessageCreateParams{
		Model:     anthropic.ModelClaude3Opus20240229,
		MaxTokens: 250,
		Messages: []anthropic.MessageParam{
			{
				Role: "user",
				Content: fmt.Sprintf(`Extract the event title, date, time, duration, and recurrence from this text: %s. 
Respond in a JSON format like this: 
{"title": "event title", "date": "event date", "time": "event time", "duration": "event duration", "recurrence": "annually"}. 
The recurrence can be "annually" for birthdays or anniversaries, or "none" for single day events. If any information is missing, leave the field empty.`, message),
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
