package spam

// loadProfanityList returns a basic list of profanity words
// In production, you should load this from a configurable external source
func loadProfanityList() []string {
	return []string{
		// Add your profanity words here
		// This is intentionally kept minimal for the example
		"porn",
		"dick",
		// We can expand this list or load from a file/database
	}
}