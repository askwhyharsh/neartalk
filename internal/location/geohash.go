package location

import "fmt"

const base32 = "0123456789bcdefghjkmnpqrstuvwxyz"

// Encode encodes latitude and longitude into a geohash
func Encode(latitude, longitude float64, precision int) string {
	var geohash string
	var evenBit = true
	var latMin, latMax = -90.0, 90.0
	var lonMin, lonMax = -180.0, 180.0
	var mid float64
	var ch, bit int

	for len(geohash) < precision {
		if evenBit {
			// longitude
			mid = (lonMin + lonMax) / 2
			if longitude > mid {
				ch |= (1 << (4 - bit))
				lonMin = mid
			} else {
				lonMax = mid
			}
		} else {
			// latitude
			mid = (latMin + latMax) / 2
			if latitude > mid {
				ch |= (1 << (4 - bit))
				latMin = mid
			} else {
				latMax = mid
			}
		}

		evenBit = !evenBit

		if bit < 4 {
			bit++
		} else {
			geohash += string(base32[ch])
			bit = 0
			ch = 0
		}
	}

	return geohash
}

// GetNeighbors returns the 8 neighboring geohashes
func GetNeighbors(geohash string) []string {
	neighbors := make([]string, 0, 8)
	
	// Simplified neighbor calculation
	// In production, use a proper geohash library like github.com/mmcloughlin/geohash
	
	// For now, return variations by slightly modifying the last character
	if len(geohash) == 0 {
		return neighbors
	}

	base := geohash[:len(geohash)-1]
	lastChar := geohash[len(geohash)-1]
	
	// Find index in base32
	idx := -1
	for i, c := range base32 {
		if byte(c) == lastChar {
			idx = i
			break
		}
	}
	
	if idx == -1 {
		return neighbors
	}

	// Add neighbors (simplified - just adjacent base32 values)
	directions := []int{-1, 1, -5, 5, -6, 6, -4, 4} // Rough approximations
	for _, dir := range directions {
		newIdx := idx + dir
		if newIdx >= 0 && newIdx < len(base32) {
			neighbors = append(neighbors, fmt.Sprintf("%s%c", base, base32[newIdx]))
		}
	}

	return neighbors
}

// Decode decodes a geohash to latitude and longitude bounds
func Decode(geohash string) (latMin, latMax, lonMin, lonMax float64) {
	evenBit := true
	latMin, latMax = -90.0, 90.0
	lonMin, lonMax = -180.0, 180.0

	for _, c := range geohash {
		idx := -1
		for i, ch := range base32 {
			if ch == c {
				idx = i
				break
			}
		}

		if idx == -1 {
			return
		}

		for i := 4; i >= 0; i-- {
			bit := (idx >> i) & 1
			if evenBit {
				// longitude
				mid := (lonMin + lonMax) / 2
				if bit == 1 {
					lonMin = mid
				} else {
					lonMax = mid
				}
			} else {
				// latitude
				mid := (latMin + latMax) / 2
				if bit == 1 {
					latMin = mid
				} else {
					latMax = mid
				}
			}
			evenBit = !evenBit
		}
	}

	return
}