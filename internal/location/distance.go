package location

import (
	"math"
)

const earthRadiusMeters = 6371000.0 // Earth's radius in meters

// HaversineDistance calculates the distance between two points on Earth in meters
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Convert to radians
	lat1Rad := toRadians(lat1)
	lat2Rad := toRadians(lat2)
	deltaLat := toRadians(lat2 - lat1)
	deltaLon := toRadians(lon2 - lon1)
	
	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return earthRadiusMeters * c
}

// RoundToNearest50 rounds distance to the nearest 50 meters for privacy
func RoundToNearest50(distance float64) int {
	return int(math.Round(distance/50.0) * 50)
}

// FormatDistance returns a privacy-preserving distance string
func FormatDistance(distance float64) string {
	rounded := RoundToNearest50(distance)
	if rounded < 1000 {
		return "~" + string(rune(rounded)) + "m"
	}
	km := float64(rounded) / 1000.0
	return "~" + string(rune(int(km*10))/10) + "km"
}

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}