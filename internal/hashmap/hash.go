package hashmap

const (
	offset64 uint64 = 14695981039346656037
	prime64  uint64 = 1099511628211
)

func hashFNV1a(key string) uint64 {
	hash := offset64
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= prime64
	}
	return hash
}
