package worker

import (
	"log"
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
)

// ConnectDB database connection helper
func ConnectDB(url string) redis.Conn {
	c, redisErr := redis.DialURL(url)
	if redisErr != nil {
		log.Println("Connect to redis error", redisErr)
		return nil
	}

	return c
}

func main() {

	for {
		log.Println("Working...")
		c := ConnectDB(os.Getenv("REDISTOGO_URL"))
		users, smembersErr := redis.Strings(c.Do("SMEMBERS", "user"))

		if smembersErr != nil {
			log.Println("SMEMBERS redis error", smembersErr)
		} else {
			log.Println(users)
		}
		defer c.Close()
		time.Sleep(60 * time.Second)
	}
}
