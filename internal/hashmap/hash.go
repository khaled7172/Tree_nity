package hashmap

// Constants for 64-bit FNV-1a hash algorithm
const (
	offset64 uint64 = 14695981039346656037
	prime64  uint64 = 1099511628211
)

// hashFNV1a takes a string (like a ClientID) and mathematically converts it 
// into a seemingly random 64-bit integer using the FNV-1a algorithm.
func hashFNV1a(key string) uint64 {
	hash := offset64
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= prime64
	}
	return hash
}
