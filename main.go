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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/lancetw/hcfd-forecast/db"
	"github.com/lancetw/hcfd-forecast/rain"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/newrelic/go-agent"
)

const timeZone = "Asia/Taipei"

var bot *linebot.Client

func main() {
	config := newrelic.NewConfig("hcfd-forecast", "99e64c65dd2b25a7eecd35b5bfe636b5b992c9bc")
	app, err := newrelic.NewApplication(config)

	strID := os.Getenv("ChannelID")
	numID, err := strconv.ParseInt(strID, 10, 64)
	if err != nil {
		log.Fatal("Wrong environment setting about ChannelID")
	}
	bot, err = linebot.NewClient(numID, os.Getenv("ChannelSecret"), os.Getenv("MID"))
	if err != nil {
		log.Println("Bot:", bot, " err:", err)
	}

	http.HandleFunc(newrelic.WrapHandleFunc(app, "/", homeHandler))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/callback", callbackHandler))

	port := os.Getenv("PORT")
	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "HCFD world")
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
		if content != nil && content.IsOperation && content.OpType == linebot.OpTypeAddedAsFriend {
			op, err := content.OperationContent()
			if err != nil {
				log.Println(err)
				return
			}
			from := op.Params[0]

			user, err := bot.GetUserProfile([]string{from})
			if err != nil {
				log.Println(err)
				return
			}
			_, err = bot.SendText([]string{from}, user.Contacts[0].DisplayName+" 您好，目前可用指令為：「加入」「退出」「雨量」「警報」「貓圖」「狀態」「時間」")
			if err != nil {
				log.Println(err)
			}
		}

		if content != nil && content.IsMessage && content.ContentType == linebot.ContentTypeText {
			user, err := bot.GetUserProfile([]string{content.From})
			if err != nil {
				return
			}
			text, err := content.TextContent()
			if err != nil {
				log.Println(err)
			}
			cmd := strings.Fields(text.Text)

			switch cmd[0] {
			case "加入":
				c := db.Connect(os.Getenv("REDISTOGO_URL"))
				status, addErr := c.Do("SADD", "user", content.From)
				if addErr != nil {
					log.Println("SADD to redis error", addErr, status)
				} else {
					_, err = bot.SendText([]string{content.From}, user.Contacts[0].DisplayName+" 您好，已將您加入傳送對象，未來將會傳送天氣警報資訊給您 ^＿^ ")
					if err != nil {
						log.Println(err)
					}
				}
				defer c.Close()

			case "退出":
				c := db.Connect(os.Getenv("REDISTOGO_URL"))
				status, setErr := c.Do("SREM", "user", content.From)
				if setErr != nil {
					log.Println("DEL to redis error", setErr, status)
				} else {
					_, err = bot.SendText([]string{content.From}, user.Contacts[0].DisplayName+" 掰掰 Q＿Q")
					if err != nil {
						log.Println(err)
					}
				}
				defer c.Close()

			case "服務":
				c := db.Connect(os.Getenv("REDISTOGO_URL"))
				count, countErr := redis.Int(c.Do("SCARD", "user"))
				if countErr != nil {
					log.Println(countErr)
				} else {
					_, err = bot.SendText([]string{content.From}, fmt.Sprintf("目前有 %d 人加入自動警訊服務。", count))
					if err != nil {
						log.Println(err)
					}
				}

				defer c.Close()

			case "狀態":
				c := db.Connect(os.Getenv("REDISTOGO_URL"))
				status, getErr := redis.Int(c.Do("SISMEMBER", "user", content.From))
				if getErr != nil || status == 0 {
					_, err = bot.SendText([]string{content.From}, "目前沒有登記您的編號喔！")
					if err != nil {
						log.Println(err)
					}
				} else {
					_, err = bot.SendText([]string{content.From}, "您已經是傳送對象 :D")
					if err != nil {
						log.Println(err)
					}
				}
				defer c.Close()

			case "時間":
				local := time.Now()
				location, timezoneErr := time.LoadLocation(timeZone)
				if timezoneErr == nil {
					local = local.In(location)
				}
				_, err = bot.SendText([]string{content.From}, local.Format("2006/01/02 15:04:05"))
				if err != nil {
					log.Println(err)
				}

			case "雨量":
				target := []string{"新竹市"}
				if len(cmd) > 1 {
					target[0] = cmd[1]
				}

				msgs, _ := rain.GetRainingInfo(target, true)
				if len(msgs) == 0 {
					_, err = bot.SendText([]string{content.From}, "目前沒有雨量資訊！")
					if err != nil {
						log.Println(err)
					}
				} else {
					for _, msg := range msgs {
						_, err = bot.SendText([]string{content.From}, msg)
						if err != nil {
							log.Println(err)
						}
					}
				}

			case "警報":
				msgs, _ := rain.GetWarningInfo(nil)
				if len(msgs) <= 0 {
					msg := "目前沒有天氣警報資訊！"
					_, err = bot.SendText([]string{content.From}, msg)
					if err != nil {
						log.Println(err)
					}
				} else {
					for _, msg := range msgs {
						_, err = bot.SendText([]string{content.From}, msg)
						if err != nil {
							log.Println(err)
						}
					}
				}

			case "重開":
				c := db.Connect(os.Getenv("REDISTOGO_URL"))
				status0, clearErr0 := c.Do("DEL", "token0")
				if clearErr0 != nil {
					log.Println("DEL to redis error", clearErr0, status0)
				}
				status1, clearErr1 := c.Do("DEL", "token1")
				if clearErr1 != nil {
					log.Println("DEL to redis error", clearErr1, status1)
				}

				_, err = bot.SendText([]string{content.From}, "已重開")
				if err != nil {
					log.Println(err)
				}

			case "貓圖":
				type Cat struct {
					File string `json:"file"`
				}
				cat := new(Cat)
				getJSON("http://random.cat/meow", &cat)
				if cat.File != "" {
					_, err = bot.SendImage([]string{content.From}, cat.File, cat.File)
					if err != nil {
						log.Println(err)
					}
				}

			case "妹子":
				type MeisData struct {
					ID         string `json:"id"`
					Name       string `json:"name"`
					ImgNo      string `json:"img_no"`
					Fanpage    string `json:"fanpage"`
					Creator    string `json:"creator"`
					UpdateTime string `json:"update_time"`
				}

				type Beauty struct {
					Meis map[string]MeisData `json:"meis"`
				}

				beauty := new(Beauty)
				getJSONErr := getJSON("http://beauty.zones.gamebase.com.tw/wall?json", &beauty)
				if len(beauty.Meis) > 0 {
					for fbid, data := range beauty.Meis {
						image := fmt.Sprintf("https://graph.facebook.com/%s/picture?type=large", fbid)
						_, err = bot.SendImage([]string{content.From}, image, image)
						if err != nil {
							log.Println(err)
						}

						link := fmt.Sprintf("https://www.facebook.com/profile.php?id=%s", fbid)
						description := fmt.Sprintf("%s %s", data.Name, link)
						_, err = bot.SendText([]string{content.From}, description)
						if err != nil {
							log.Println(err)
						}
						break
					}

				}

				if getJSONErr != nil {
					log.Println(getJSONErr)
				}

			default:
				_, err = bot.SendText([]string{content.From}, "指令錯誤，請重試")
				if err != nil {
					log.Println(err)
				}

			}
		}
	}
}

// getJSON func
func getJSON(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}
