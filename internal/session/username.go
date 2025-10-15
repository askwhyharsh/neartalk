package session

import (
	"fmt"
	"math/rand"
	"time"
)

var adjectives = []string{
	"Happy", "Lucky", "Swift", "Bright", "Cool", "Smart", "Brave", "Quick",
	"Calm", "Bold", "Wise", "Silent", "Sharp", "Gentle", "Noble", "Wild",
}

var nouns = []string{
	"Panda", "Tiger", "Eagle", "Falcon", "Wolf", "Bear", "Fox", "Hawk",
	"Lion", "Otter", "Raven", "Lynx", "Deer", "Owl", "Cobra", "Shark",
}

func init() {
	// rand.Seed(time.Now().UnixNano())
	rand.NewSource(time.Now().UnixNano())
}

func generateRandomUsername() string {
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	num := rand.Intn(9999)
	return fmt.Sprintf("%s%s%d", adj, noun, num)
}