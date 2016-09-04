package rain

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func fetchXML(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}

	xmldata, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}

	return xmldata
}

// GetInfo from "中央氣象局"
func GetInfo() []string {
	var msgs = []string{}

	target := "新竹市"

	rainLevel := map[string]float32{
		"10minutes": 6.0,
		"1hour":     30.0,
	}

	type WeatherElement struct {
		Name  string  `xml:"elementName"`
		Value float32 `xml:"elementValue>value"`
	}

	type Parameter struct {
		Name  string `xml:"parameterName"`
		Value string `xml:"parameterValue"`
	}

	type ValidTime struct {
		StartTime time.Time `xml:"startTime"`
		EndTime   time.Time `xml:"endTime"`
	}

	type AffectedAreas struct {
		Name string `xml:"locationName"`
	}

	type HazardInfo0 struct {
		Language     string `xml:"language"`
		Phenomena    string `xml:"phenomena"`
		Significance string `xml:"significance"`
	}

	type HazardInfo1 struct {
		Language      string          `xml:"language"`
		Phenomena     string          `xml:"phenomena"`
		AffectedAreas []AffectedAreas `xml:"affectedAreas>location"`
	}

	type Hazards struct {
		Info       HazardInfo0 `xml:"info"`
		ValidTime  ValidTime   `xml:"validTime"`
		HazardInfo HazardInfo1 `xml:"hazard>info"`
	}

	type Location0 struct {
		Lat            float32          `xml:"lat"`
		Lng            float32          `xml:"lng"`
		Name           string           `xml:"locationName"`
		StationID      string           `xml:"stationId"`
		Time           time.Time        `xml:"time>obsTime"`
		WeatherElement []WeatherElement `xml:"weatherElement"`
		Parameter      []Parameter      `xml:"parameter"`
	}

	type Location1 struct {
		Geocode int     `xml:"geocode"`
		Name    string  `xml:"locationName"`
		Hazards Hazards `xml:"hazardConditions>hazards"`
	}

	type Result0 struct {
		Location []Location0 `xml:"location"`
	}

	type Result1 struct {
		Location []Location1 `xml:"dataset>location"`
	}

	authKey := "CWB-FB35C2AC-9286-4B7E-AD11-6BBB7F2855F7"
	baseURL := "http://opendata.cwb.gov.tw/opendataapi?dataid="

	url0 := baseURL + "O-A0002-001" + "&authorizationkey=" + authKey
	xmldata0 := fetchXML(url0)

	v0 := Result0{}
	err := xml.Unmarshal([]byte(xmldata0), &v0)
	if err != nil {
		log.Printf("error: %v", err)
		return []string{}
	}

	for _, location := range v0.Location {
		for _, parameter := range location.Parameter {
			if parameter.Name == "CITY" && parameter.Value == target {
				for _, element := range location.WeatherElement {
					switch element.Name {
					case "MIN_10":
						if element.Value < 0 {
							log.Printf("%s：%s", "十分鐘雨量", "-")
						} else {
							log.Printf("%s：%.2f", "十分鐘雨量", element.Value)
							if element.Value > rainLevel["10minutes"] {
								msgs = append(msgs, fmt.Sprintf("[豪大雨警報] %s 地區 %s 為 %f", element.Name, "十分鐘雨量", element.Value))
							}
						}
					case "RAIN":
						if element.Value < 0 {
							log.Printf("[%s]", location.Name)
							log.Printf("%s：%s", "一小時雨量", "-")
						} else {
							log.Printf("[%s]", location.Name)
							log.Printf("%s：%.2f", "一小時雨量", element.Value)
							if element.Value > rainLevel["1hour"] {
								msgs = append(msgs, fmt.Sprintf("[豪大雨警報] %s 地區 %s 為 %f", element.Name, "一小時雨量", element.Value))
							}
						}
					}
				}
			}
		}
	}

	url1 := baseURL + "W-C0033-001" + "&authorizationkey=" + authKey
	xmldata1 := fetchXML(url1)

	v1 := Result1{}
	if xml.Unmarshal([]byte(xmldata1), &v1) != nil {
		log.Printf("error: %v", err)
		return []string{}
	}

	var hazardmsgs = ""
	for _, location := range v1.Location {
		if location.Hazards.Info.Phenomena != "" {
			log.Println("***************************************")
			log.Printf("【%s%s%s】影響地區：", location.Name, location.Hazards.Info.Phenomena, location.Hazards.Info.Significance)
			m := fmt.Sprintf("【%s%s%s】影響地區：", location.Name, location.Hazards.Info.Phenomena, location.Hazards.Info.Significance)
			for _, str := range location.Hazards.HazardInfo.AffectedAreas {
				log.Printf("%s ", str.Name)
				m = m + fmt.Sprintf("%s ", str.Name)
			}
			m = m + "\n"
			log.Println("\n***************************************")
			hazardmsgs = hazardmsgs + m
		}
	}

	msgs = append(msgs, hazardmsgs)

	return msgs
}
