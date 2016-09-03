package main

import (
	"log"
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/lancetw/hcfd-forecast/db"
)

func main() {

	for {
		log.Println("Working...")
		c := db.Connect(os.Getenv("REDISTOGO_URL"))
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
