package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/lancetw/hcfd-forecast/db"
	"github.com/line/line-bot-sdk-go/linebot"
)

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
		var msg = "測試自動發訊息～～ :D"

		log.Println("[Working] ", time.Now().Format("Mon Jan _2 15:04:05 2006"))

		c := db.Connect(os.Getenv("REDISTOGO_URL"))
		users, smembersErr := redis.Strings(c.Do("SMEMBERS", "user"))

		if smembersErr != nil {
			log.Println("SMEMBERS redis error", smembersErr)
		} else {
			for _, contentTo := range users {
				_, err = bot.SendText([]string{contentTo}, msg)
				if err != nil {
					log.Println(err)
				}
			}
		}
		defer c.Close()
		time.Sleep(60 * time.Second)
	}
}
