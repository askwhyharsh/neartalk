package spam

import (
	"regexp"
	"strings"
	"unicode"
)

// Common spam patterns
var (
	// URL patterns
	urlPattern = regexp.MustCompile(`https?://[^\s]+|www\.[^\s]+`)
	
	// Email patterns
	emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	
	// Phone number patterns (various formats)
	phonePattern = regexp.MustCompile(`(\+?\d{1,3}[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`)
	
	// Suspicious patterns
	moneyPattern = regexp.MustCompile(`(?i)(free|win|winner|prize|cash|money|$\d+|\d+\s*(dollars?|usd|euro|₹))`)
	
	// Promotional/scam patterns
	scamPattern = regexp.MustCompile(`(?i)(click here|buy now|limited time|act now|guarantee|risk free|no obligation)`)
)

// Profanity/offensive words list (expand as needed)
var profanityList = []string{
	// Add common profanity words here
	// Note: Using mild examples for demonstration
	// "spam", "scam", "fraud",
}

// DetectPattern checks if the message contains spam patterns
func DetectPattern(content string) (bool, string) {
	// Check for excessive repeated characters
	if hasExcessiveRepeatedChars(content, 5) {
		return true, "excessive_repeated_chars"
	}
	
	// Check for URLs
	if urlPattern.MatchString(content) {
		return true, "contains_url"
	}
	
	// Check for emails
	if emailPattern.MatchString(content) {
		return true, "contains_email"
	}
	
	// Check for phone numbers
	if phonePattern.MatchString(content) {
		return true, "contains_phone"
	}
	
	// Check for money/prize patterns
	if moneyPattern.MatchString(content) {
		return true, "suspicious_money_mention"
	}
	
	// Check for scam patterns
	if scamPattern.MatchString(content) {
		return true, "suspicious_promotional_content"
	}
	
	// Check for excessive caps
	if hasExcessiveCaps(content) {
		return true, "excessive_caps"
	}
	
	return false, ""
}

// DetectProfanity checks if the message contains profanity
func DetectProfanity(content string) bool {
	lowerContent := strings.ToLower(content)
	
	for _, word := range profanityList {
		if strings.Contains(lowerContent, word) {
			return true
		}
	}
	
	return false
}

// hasExcessiveRepeatedChars checks if a string has too many repeated characters
func hasExcessiveRepeatedChars(s string, max int) bool {
	if len(s) == 0 {
		return false
	}
	
	count := 1
	lastChar := rune(0)
	
	for _, char := range s {
		if char == lastChar {
			count++
			if count > max {
				return true
			}
		} else {
			count = 1
			lastChar = char
		}
	}
	
	return false
}

// hasExcessiveCaps checks if message has too many capital letters
func hasExcessiveCaps(s string) bool {
	if len(s) < 10 {
		return false // Too short to judge
	}
	
	capsCount := 0
	letterCount := 0
	
	for _, char := range s {
		if unicode.IsLetter(char) {
			letterCount++
			if unicode.IsUpper(char) {
				capsCount++
			}
		}
	}
	
	if letterCount == 0 {
		return false
	}
	
	// If more than 70% caps, consider it excessive
	capsPercentage := float64(capsCount) / float64(letterCount)
	return capsPercentage > 0.7
}

// SanitizeMessage removes or replaces suspicious content
func SanitizeMessage(content string) string {
	// Remove URLs
	content = urlPattern.ReplaceAllString(content, "[URL removed]")
	
	// Remove emails
	content = emailPattern.ReplaceAllString(content, "[Email removed]")
	
	// Remove phone numbers
	content = phonePattern.ReplaceAllString(content, "[Phone removed]")
	
	return strings.TrimSpace(content)
}

// CalculateSpamScore returns a spam score (0-100)
func CalculateSpamScore(content string) int {
	score := 0
	
	if hasExcessiveRepeatedChars(content, 5) {
		score += 20
	}
	
	if urlPattern.MatchString(content) {
		score += 30
	}
	
	if emailPattern.MatchString(content) {
		score += 25
	}
	
	if phonePattern.MatchString(content) {
		score += 25
	}
	
	if moneyPattern.MatchString(content) {
		score += 15
	}
	
	if scamPattern.MatchString(content) {
		score += 20
	}
	
	if hasExcessiveCaps(content) {
		score += 15
	}
	
	if DetectProfanity(content) {
		score += 10
	}
	
	if score > 100 {
		score = 100
	}
	
	return score
}