package config

import "os"

type Config struct {
	Port          string
	MongoURI      string
	MongoDB       string
	RedisAddr     string
	JWTSecret     string
	CORSOrigin    string
	RedisPassword string
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8080"),
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:       getEnv("MONGO_DB", "sreechat"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", "root"),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-change-me"),
		CORSOrigin:    getEnv("CORS_ORIGIN", "http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
