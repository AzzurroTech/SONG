module github.com/azzurrotech/song

go 1.21

require (
	github.com/gorilla/mux v1.8.1
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/redis/go-redis/v9 v9.5.1
	golang.org/x/crypto v0.18.0
	golang.org/x/oauth2 v0.16.0
)

// Note: Additional dependencies will be added as we implement specific features
// e.g., database drivers for MySQL, SQLite, MongoDB, etc.