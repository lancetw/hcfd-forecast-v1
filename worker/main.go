package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/lancetw/hcfd-forecast/db"
	"github.com/lancetw/hcfd-forecast/rain"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/robfig/cron"
)

const timeZone = "Asia/Taipei"

var bot *linebot.Client

func main() {
	c := cron.New()
	c.AddFunc("0 */1 * * * *", GoProcess)
	c.Start()

	for {
		time.Sleep(10000000000000)
		fmt.Println("sleep")
	}
}

// GoProcess is main process
func GoProcess() {
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
		log.Println("=== 查詢。開始 ===")

		c := db.Connect(os.Getenv("REDISTOGO_URL"))

		targets0 := []string{"新竹市"}
		msgs0, token0 := rain.GetRainingInfo(targets0, false)

		if token0 != "" {
			status0, getErr := redis.Int(c.Do("SISMEMBER", "token0", token0))
			if getErr != nil {
				if err != nil {
					log.Println(err)
				}
			}

			if status0 == 0 {
				users0, smembersErr := redis.Strings(c.Do("SMEMBERS", "user"))

				if smembersErr != nil {
					log.Println("GetRainingInfo SMEMBERS redis error", smembersErr)
				} else {
					local := time.Now()
					location, timeZoneErr := time.LoadLocation(timeZone)
					if timeZoneErr == nil {
						local = local.In(location)
					}
					for _, contentTo := range users0 {
						for _, msg := range msgs0 {
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
				log.Println("GetRainingInfo SADD to redis error", addErr, n0)
			}
		}

		targets1 := []string{"新竹市", "新竹縣"}
		msgs1, token1 := rain.GetWarningInfo(targets1)

		if token1 != "" {
			status1, getErr := redis.Int(c.Do("SISMEMBER", "token1", token1))
			if getErr != nil {
				log.Println(getErr)
			}

			if status1 == 0 {
				users1, smembersErr := redis.Strings(c.Do("SMEMBERS", "user"))

				if smembersErr != nil {
					log.Println("GetWarningInfo SMEMBERS redis error", smembersErr)
				} else {
					local := time.Now()
					location, locationErr := time.LoadLocation(timeZone)
					if locationErr == nil {
						local = local.In(location)
					}
					for _, contentTo := range users1 {
						for _, msg := range msgs1 {
							_, msgErr := bot.SendText([]string{contentTo}, msg)
							if msgErr != nil {
								log.Println(err)
							}
						}
					}
				}
			}
		}

		if token1 != "" {
			n, addErr := c.Do("SADD", "token1", token1)
			if addErr != nil {
				log.Println("GetWarningInfo SADD to redis error", addErr, n)
			}
		}

		defer c.Close()

		log.Println("=== 查詢。結束 ===")

		time.Sleep(60 * time.Second)
	}
}
