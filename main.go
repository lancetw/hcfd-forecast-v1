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
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/lancetw/hcfd-forecast-v1/db"
	"github.com/lancetw/hcfd-forecast-v1/rain"
	"github.com/line/line-bot-sdk-go/linebot"
)

const timeZone = "Asia/Taipei"

var bot *linebot.Client

func main() {
	var err error
	bot, err = linebot.New(os.Getenv("CHANNEL_SECRET"), os.Getenv("ACCESS_TOKEN"))
	if err != nil {
		log.Println("Bot:", bot, " err:", err)
		return
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/callback", callbackHandler)

	port := os.Getenv("PORT")
	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "HCFD world")
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	events, parseRequestErr := bot.ParseRequest(r)
	if parseRequestErr != nil {
		if parseRequestErr == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		replyToken := event.ReplyToken
		log.Println(event.Type)
		if event.Type == linebot.EventTypeMessage {
			switch event.Type {
			case linebot.EventTypeFollow:
				profile, getProfileErr := bot.GetProfile(event.Source.UserID).Do()
				if getProfileErr != nil {
					bot.ReplyMessage(replyToken, linebot.NewTextMessage(getProfileErr.Error()))
					log.Println(getProfileErr)
				}
				text := profile.DisplayName + " 您好，目前可用指令為：「加入」「退出」「雨量」「警報」「貓圖」「狀態」「時間」"
				if _, replyErr := bot.ReplyMessage(
					replyToken,
					linebot.NewTextMessage(text)).Do(); replyErr != nil {
					log.Println(replyErr)
				}
			case linebot.EventTypeMessage:
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					profile, getProfileErr := bot.GetProfile(event.Source.UserID).Do()
					if getProfileErr != nil {
						bot.ReplyMessage(replyToken, linebot.NewTextMessage(getProfileErr.Error()))
						log.Println(getProfileErr)
					}

					cmd := strings.Fields(message.Text)

					switch cmd[0] {
					case "加入":
						c := db.Connect(os.Getenv("REDISTOGO_URL"))
						status, addErr := c.Do("SADD", "user", replyToken)
						defer c.Close()
						if addErr != nil {
							log.Println("SADD to redis error", addErr, status)
						} else {
							text := profile.DisplayName + " 您好，已將您加入傳送對象，未來將會傳送天氣警報資訊給您 ^＿^ "
							if _, replyErr := bot.ReplyMessage(
								replyToken,
								linebot.NewTextMessage(text)).Do(); replyErr != nil {
								log.Println(replyErr)
							}
						}
					case "退出":
						c := db.Connect(os.Getenv("REDISTOGO_URL"))
						status, setErr := c.Do("SREM", "user", replyToken)
						defer c.Close()

						if setErr != nil {
							log.Println("DEL to redis error", setErr, status)
						} else {
							text := profile.DisplayName + " 掰掰 Q＿Q"
							if _, replyErr := bot.ReplyMessage(
								replyToken,
								linebot.NewTextMessage(text)).Do(); replyErr != nil {
								log.Println(replyErr)
							}
						}
					case "服務":
						c := db.Connect(os.Getenv("REDISTOGO_URL"))
						count, countErr := redis.Int(c.Do("SCARD", "user"))
						defer c.Close()

						if countErr != nil {
							log.Println(countErr)
						} else {
							text := fmt.Sprintf("目前有 %d 人加入自動警訊服務。", count)
							if _, replyErr := bot.ReplyMessage(
								replyToken,
								linebot.NewTextMessage(text)).Do(); replyErr != nil {
								log.Println(replyErr)
							}
						}
					case "狀態":
						c := db.Connect(os.Getenv("REDISTOGO_URL"))
						status, getErr := redis.Int(c.Do("SISMEMBER", "user", replyToken))

						var text string
						if getErr != nil || status == 0 {
							text = "目前沒有登記您的編號喔！"
						} else {
							text = "您已經是傳送對象 :D"
						}

						defer c.Close()

						if _, replyErr := bot.ReplyMessage(
							replyToken,
							linebot.NewTextMessage(text)).Do(); replyErr != nil {
							log.Println(replyErr)
						}

					case "時間":
						local := time.Now()
						location, timezoneErr := time.LoadLocation(timeZone)
						if timezoneErr == nil {
							local = local.In(location)
						}

						text := local.Format("2006/01/02 15:04:05")
						if _, replyErr := bot.ReplyMessage(
							replyToken,
							linebot.NewTextMessage(text)).Do(); replyErr != nil {
							log.Println(replyErr)
						}

					case "雨量":
						target := []string{"新竹市"}
						if len(cmd) > 1 {
							target[0] = cmd[1]
						}

						msgs, _ := rain.GetRainingInfo(target, true)

						local := time.Now()
						location, timezoneErr := time.LoadLocation(timeZone)
						if timezoneErr == nil {
							local = local.In(location)
						}
						now := local.Format("15:04:05")

						var text string
						if len(msgs) == 0 {
							text = "目前沒有雨量資訊！"
							if _, replyErr := bot.ReplyMessage(
								replyToken,
								linebot.NewTextMessage(text)).Do(); replyErr != nil {
								log.Println(replyErr)
							}
						} else {
							if len(msgs) > 0 {
								for _, msg := range msgs {
									text = text + msg + "\n\n"
								}
								text = strings.TrimSpace(text)
								text = text + "\n\n" + now
								if _, replyErr := bot.ReplyMessage(
									replyToken,
									linebot.NewTextMessage(text)).Do(); replyErr != nil {
									log.Println(replyErr)
								}
							}
						}

					case "警報":
						msgs, _ := rain.GetWarningInfo(nil)

						local := time.Now()
						location, timezoneErr := time.LoadLocation(timeZone)
						if timezoneErr == nil {
							local = local.In(location)
						}
						now := local.Format("15:04:05")
						var text string
						if len(msgs) <= 0 {
							text = "目前沒有天氣警報資訊！"
							if _, replyErr := bot.ReplyMessage(
								replyToken,
								linebot.NewTextMessage(text)).Do(); replyErr != nil {
								log.Println(replyErr)
							}
						} else {
							if len(msgs) > 0 {
								for _, msg := range msgs {
									text = text + msg + "\n\n"
								}
								text = strings.TrimSpace(text)
								text = text + "\n\n" + now
								if _, replyErr := bot.ReplyMessage(
									replyToken,
									linebot.NewTextMessage(text)).Do(); replyErr != nil {
									log.Println(replyErr)
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

						if _, replyErr := bot.ReplyMessage(
							replyToken,
							linebot.NewTextMessage("已重開")).Do(); replyErr != nil {
							log.Println(replyErr)
						}

					case "貓圖":
						image := "https://thecatapi.com/api/images/get?format=src&type=jpg&api_key=MTI5ODM2"
						if image != "" {
							if _, replyErr := bot.ReplyMessage(
								replyToken,
								linebot.NewImageMessage(image, image)).Do(); replyErr != nil {
								log.Println(replyErr)
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
								if image != "" {
									link := fmt.Sprintf("https://www.facebook.com/profile.php?id=%s", fbid)
									description := fmt.Sprintf("%s %s", data.Name, link)

									if _, replyErr := bot.ReplyMessage(
										replyToken,
										linebot.NewImageMessage(image, image),
										linebot.NewTextMessage(description)).Do(); replyErr != nil {
										log.Println(replyErr)
									}
								}

								break
							}
						}

						if getJSONErr != nil {
							log.Println(getJSONErr)
						}

					default:
						if _, replyErr := bot.ReplyMessage(
							replyToken,
							linebot.NewTextMessage("指令錯誤，請重試")).Do(); replyErr != nil {
							log.Println(replyErr)
						}
					}
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
