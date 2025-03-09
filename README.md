# MattBot

MattBot is a WhatsApp automation tool.

## Prerequisites

*   Go installed
*   `chromedp`, `go-redis`, `cobra`, and `anthropic-go` Go packages
*   Redis server running (default: localhost:6379)
*   Claude API key

## Installation

1.  Clone the repository:

    ```bash
    git clone <repository_url>
    cd mattbot
    ```

2.  Download dependencies:

    ```bash
    go mod download
    ```

## Usage

1.  Compile MattBot:

    ```bash
    go build -o mattbot main.go
    ```

2.  Initialize MattBot with your WhatsApp phone number and Claude API key:

    ```bash
    ./mattbot init
    ```

    You will be prompted to enter your phone number (including country code, without the `+` sign or leading zeros) and your Claude API key.

3.  Run MattBot:

    ```bash
    ./mattbot run
    ```

    The program will open a Chrome window and navigate to WhatsApp Web. You need to manually scan the QR code with your WhatsApp mobile app. After scanning the QR code, MattBot will listen for "@" mentions. You can ask MattBot to create calendar events using natural language, for example: "@MattBot create an event called 'Meeting with John' on July 24th at 3pm for 1 hour". MattBot will respond with a confirmation or an error message if any information is missing.

## Notes

*   Make sure Redis is running and accessible.
*   The selectors used in the program might need to be adjusted based on the WhatsApp Web version. Inspect the WhatsApp Web page to find the correct selectors if the program fails.
*   Error handling is basic. More robust error handling should be implemented for production use.
*   This is a basic example and can be extended to perform other automated tasks on WhatsApp Web.
