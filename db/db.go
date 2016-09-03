package db

import (
	"log"

	"github.com/garyburd/redigo/redis"
)

// Connect database connection helper
func Connect(url string) redis.Conn {
	c, redisErr := redis.DialURL(url)
	if redisErr != nil {
		log.Println("Connect to redis error", redisErr)
		return nil
	}

	return c
}
