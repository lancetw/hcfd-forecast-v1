package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/lancetw/hcfd-forecast/db"
	"github.com/lancetw/hcfd-forecast/rain"
	"github.com/line/line-bot-sdk-go/linebot"
)

const timeZone = "Asia/Taipei"

var bot *linebot.Client

func main() {
	strID := os.Getenv("ChannelID")
	numID, err := strconv.ParseInt(strID, 10, 64)
	if err != nil {
		log.Fatal("Wrong environment setting about ChannelID")
	}
	bot, err = linebot.NewClient(numID, os.Getenv("ChannelSecret"), os.Getenv("MID"))
	if err != nil {
		log.Println("Bot:", bot, " err:", err)
	}

	for {
		c := db.Connect(os.Getenv("REDISTOGO_URL"))

		targets0 := []string{"新竹市", "新竹縣"}
		msgs0, token0 := rain.GetRainingInfo(targets0)

		status0, getErr := redis.Int(c.Do("SISMEMBER", "token0", token0))
		if getErr != nil {
			if err != nil {
				log.Println(err)
			}
		}

		targets1 := []string{"新竹市", "新竹縣", "台中市", "高雄市", "台北市"}
		msgs1, token1 := rain.GetWarningInfo(targets1)

		status1, getErr := redis.Int(c.Do("SISMEMBER", "token1", token1))
		if getErr != nil {
			if err != nil {
				log.Println(err)
			}
		}

		if status0 == 0 && status1 == 0 {
			log.Println("\n***************************************")

			users, smembersErr := redis.Strings(c.Do("SMEMBERS", "user"))

			if smembersErr != nil {
				log.Println("SMEMBERS redis error", smembersErr)
			} else {
				local := time.Now()
				location, err := time.LoadLocation(timeZone)
				if err == nil {
					local = local.In(location)
				}
				log.Println(users)
				log.Println(msgs0)
				log.Println(msgs1)
				for _, contentTo := range users {
					for _, msg := range msgs0 {
						_, err = bot.SendText([]string{contentTo}, msg)
						if err != nil {
							log.Println(err)
						}
					}
					for _, msg := range msgs1 {
						_, err = bot.SendText([]string{contentTo}, msg)
						if err != nil {
							log.Println(err)
						}
					}
				}
			}
		}

		n0, addErr := c.Do("SADD", "token0", token0)
		if addErr != nil {
			log.Println("SADD to redis error", addErr, n0)
		}

		n, addErr := c.Do("SADD", "token1", token1)
		if addErr != nil {
			log.Println("SADD to redis error", addErr, n)
		}

		defer c.Close()

		time.Sleep(60 * time.Second)
	}
}
