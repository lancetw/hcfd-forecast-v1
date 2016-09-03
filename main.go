// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/garyburd/redigo/redis"
	"github.com/lancetw/hcfd-forecast/worker"
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
	log.Println("Bot:", bot, " err:", err)
	http.HandleFunc("/callback", callbackHandler)
	port := os.Getenv("PORT")
	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	received, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}
	for _, result := range received.Results {
		content := result.Content()
		if content != nil && content.IsMessage && content.ContentType == linebot.ContentTypeText {
			user, err := bot.GetUserProfile([]string{content.From})
			if err != nil {
				return
			}
			text, err := content.TextContent()
			if err != nil {
				log.Println(err)
			}
			switch text.Text {
			case "加入":
				if user.Count == 1 {
					c := worker.ConnectDB(os.Getenv("REDISTOGO_URL"))
					n, appendErr := c.Do("SADD", "user", content.From)
					if appendErr != nil {
						log.Println("SET to redis error", appendErr, n)
					} else {
						_, err = bot.SendText([]string{content.From}, user.Contacts[0].DisplayName+" 您好，已將您加入傳送對象 ^＿^ "+content.From)
						if err != nil {
							log.Println(err)
						}
					}
					defer c.Close()
				}
			case "退出":
				if user.Count == 1 {
					c := worker.ConnectDB(os.Getenv("REDISTOGO_URL"))
					n, setErr := c.Do("SREM", "user", user.Contacts[0].MID)
					if setErr != nil {
						log.Println("DEL to redis error", setErr, n)
					} else {
						_, err = bot.SendText([]string{content.From}, user.Contacts[0].DisplayName+" 掰掰 Q＿Q")
						if err != nil {
							log.Println(err)
						}
					}
					defer c.Close()
				}
			case "狀態":
				if user.Count == 1 {
					c := worker.ConnectDB(os.Getenv("REDISTOGO_URL"))
					status, getErr := redis.Int(c.Do("SISMEMBER", user.Contacts[0].MID))
					if getErr != nil || status == 0 {
						_, err = bot.SendText([]string{content.From}, "目前沒有登記您的編號喔！")
						if err != nil {
							log.Println(err)
						}
					} else {
						_, err = bot.SendText([]string{content.From}, user.Contacts[0].MID)
						if err != nil {
							log.Println(err)
						}
					}
					defer c.Close()
				}
			}
		}
	}
}
