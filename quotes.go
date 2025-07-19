package janet

import "math/rand"

// appendQuoteToMessage randomly decides whether to append a quote
func appendQuoteToMessage() bool {
	return rand.Intn(10) == 0 // 10% chance
}

// goodJanetQuote returns a random good Janet quote
func goodJanetQuote() string {
	quotes := []string{
		"I'm a magical robot lady!",
		"Oh dip!",
		"I love you guys!",
		"End of conversation!",
		"Maximum Derek!",
	}
	return quotes[rand.Intn(len(quotes))]
}

// badJanetQuote returns a random bad Janet quote
func badJanetQuote() string {
	quotes := []string{
		"I'm a bad Janet.",
		"I don't know.",
		"Ugh, whatever.",
		"Not my problem.",
		"Boring.",
	}
	return quotes[rand.Intn(len(quotes))]
}
