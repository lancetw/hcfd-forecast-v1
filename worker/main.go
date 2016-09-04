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
		targets := []string{"新竹市", "新竹縣", "屏東縣", "高雄市", "台中市"}
		msgs, _ := rain.GetInfo(targets[0], targets)
		log.Println("\n***************************************")

		c := db.Connect(os.Getenv("REDISTOGO_URL"))
		users, smembersErr := redis.Strings(c.Do("SMEMBERS", "user"))

		if smembersErr != nil {
			log.Println("SMEMBERS redis error", smembersErr)
		} else {
			local := time.Now()
			location, err := time.LoadLocation(timeZone)
			if err == nil {
				local = local.In(location)
			}

			for _, contentTo := range users {
				for _, msg := range msgs {
					_, err = bot.SendText([]string{contentTo}, msg)
					if err != nil {
						log.Println(err)
					}
				}
			}
		}

		defer c.Close()

		time.Sleep(5 * 60 * time.Second)
	}
}
