package db

// KVStore represents the database schema
type KVStore struct {
	Key   string `gorm:"primaryKey;uniqueIndex"`
	Value any    `gorm:"serializer:json"`
}
