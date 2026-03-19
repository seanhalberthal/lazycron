package mail

import (
	"bufio"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// Parse parses mbox-format text into a slice of Messages.
// The mbox format separates messages with lines starting with "From "
// (the envelope sender line).
func Parse(text string) ([]*Message, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}

	rawMessages := splitMessages(text)
	messages := make([]*Message, 0, len(rawMessages))

	for i, raw := range rawMessages {
		msg, err := parseMessage(raw)
		if err != nil {
			// Skip malformed messages rather than failing entirely
			continue
		}
		msg.Index = i
		messages = append(messages, msg)
	}

	return messages, nil
}

// splitMessages splits mbox text into individual raw message strings.
// Each message begins with a line matching "From " (the mbox envelope line).
func splitMessages(text string) []string {
	var messages []string
	var current strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(text))
	inMessage := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "From ") && isEnvelopeLine(line) {
			// Start of a new message
			if inMessage && current.Len() > 0 {
				messages = append(messages, current.String())
				current.Reset()
			}
			inMessage = true
			current.WriteString(line)
			current.WriteString("\n")
			continue
		}

		if inMessage {
			current.WriteString(line)
			current.WriteString("\n")
		}
	}

	// Flush the last message
	if inMessage && current.Len() > 0 {
		messages = append(messages, current.String())
	}

	return messages
}

// isEnvelopeLine performs a basic check that a "From " line looks like
// an mbox envelope line (has an email-like token and a date).
func isEnvelopeLine(line string) bool {
	// "From user@host Day Mon DD HH:MM:SS YYYY"
	// Minimum: "From x D" = at least 2 fields after "From "
	fields := strings.Fields(line)
	return len(fields) >= 3
}

// parseMessage parses a single raw message (including envelope line) into a Message.
func parseMessage(raw string) (*Message, error) {
	// Skip the "From " envelope line to get to RFC 5322 headers
	headerStart := strings.Index(raw, "\n")
	if headerStart < 0 {
		return nil, fmt.Errorf("message has no content after envelope line")
	}
	content := raw[headerStart+1:]

	// Use net/mail to parse headers
	r := strings.NewReader(content)
	parsed, err := mail.ReadMessage(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message headers: %w", err)
	}

	// Read body
	var body strings.Builder
	scanner := bufio.NewScanner(parsed.Body)
	for scanner.Scan() {
		body.WriteString(scanner.Text())
		body.WriteString("\n")
	}

	// Parse date
	var date time.Time
	if dateStr := parsed.Header.Get("Date"); dateStr != "" {
		if t, err := mail.ParseDate(dateStr); err == nil {
			date = t
		}
	}

	return &Message{
		From:    parsed.Header.Get("From"),
		To:      parsed.Header.Get("To"),
		Subject: parsed.Header.Get("Subject"),
		Date:    date,
		Body:    strings.TrimSpace(body.String()),
		Status:  parsed.Header.Get("Status"),
		Raw:     raw,
	}, nil
}
