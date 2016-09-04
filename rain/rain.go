package rain

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Location0 struct
type Location0 struct {
	Lat            float32          `xml:"lat"`
	Lng            float32          `xml:"lng"`
	Name           string           `xml:"locationName"`
	StationID      string           `xml:"stationId"`
	Time           time.Time        `xml:"time>obsTime"`
	WeatherElement []WeatherElement `xml:"weatherElement"`
	Parameter      []Parameter      `xml:"parameter"`
}

// Location1 struct
type Location1 struct {
	Geocode int     `xml:"geocode"`
	Name    string  `xml:"locationName"`
	Hazards Hazards `xml:"hazardConditions>hazards"`
}

// WeatherElement struct
type WeatherElement struct {
	Name  string  `xml:"elementName"`
	Value float32 `xml:"elementValue>value"`
}

// Parameter struct
type Parameter struct {
	Name  string `xml:"parameterName"`
	Value string `xml:"parameterValue"`
}

// ValidTime struct
type ValidTime struct {
	StartTime time.Time `xml:"startTime"`
	EndTime   time.Time `xml:"endTime"`
}

// AffectedAreas struct
type AffectedAreas struct {
	Name string `xml:"locationName"`
}

// HazardInfo0 struct
type HazardInfo0 struct {
	Language     string `xml:"language"`
	Phenomena    string `xml:"phenomena"`
	Significance string `xml:"significance"`
}

// HazardInfo1 struct
type HazardInfo1 struct {
	Language      string          `xml:"language"`
	Phenomena     string          `xml:"phenomena"`
	AffectedAreas []AffectedAreas `xml:"affectedAreas>location"`
}

// Hazards struct
type Hazards struct {
	Info       HazardInfo0 `xml:"info"`
	ValidTime  ValidTime   `xml:"validTime"`
	HazardInfo HazardInfo1 `xml:"hazard>info"`
}

// Result0 struct
type Result0 struct {
	Location []Location0 `xml:"location"`
}

// Result1 struct
type Result1 struct {
	Location []Location1 `xml:"dataset>location"`
}

const timeZone = "Asia/Taipei"

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
func GetInfo(place string, targets []string) ([]string, string) {
	var msgs = []string{}

	rainLevel := map[string]float32{
		"10minutes": 6.0,
		"1hour":     30.0,
	}

	authKey := "CWB-FB35C2AC-9286-4B7E-AD11-6BBB7F2855F7"
	baseURL := "http://opendata.cwb.gov.tw/opendataapi?dataid="

	url0 := baseURL + "O-A0002-001" + "&authorizationkey=" + authKey
	xmldata0 := fetchXML(url0)

	v0 := Result0{}
	err := xml.Unmarshal([]byte(xmldata0), &v0)
	if err != nil {
		log.Printf("error: %v", err)
		return []string{}, ""
	}

	for _, location := range v0.Location {
		for _, parameter := range location.Parameter {
			if parameter.Name == "CITY" && parameter.Value == place {
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
		return []string{}, ""
	}

	local := time.Now()
	location, err := time.LoadLocation(timeZone)
	if err == nil {
		local = local.In(location)
	}

	var hazardmsgs = ""
	var token = ""
	for i, location := range v1.Location {
		if i == 0 {
			token = token + strconv.Itoa(location.Geocode)
		}
		if location.Hazards.Info.Phenomena != "" && location.Hazards.ValidTime.EndTime.After(local) {
			if targets != nil {
				for _, name := range targets {
					if name == location.Name {
						hazardmsgs = hazardmsgs + saveHazards(location)
					}
				}
			} else {
				hazardmsgs = hazardmsgs + saveHazards(location)
			}
		}
	}

	msgs = append(msgs, hazardmsgs)

	return msgs, token
}

func saveHazards(location Location1) string {
	log.Println("***************************************")
	log.Printf("【%s%s%s %s ~ %s】影響地區：", location.Name, location.Hazards.Info.Phenomena, location.Hazards.Info.Significance, location.Hazards.ValidTime.StartTime.Format("01/02 15:04"), location.Hazards.ValidTime.EndTime.Format("01/02 15:04"))
	m := fmt.Sprintf("【%s%s%s %s ~ %s】影響地區：", location.Name, location.Hazards.Info.Phenomena, location.Hazards.Info.Significance, location.Hazards.ValidTime.StartTime.Format("01/02 15:04"), location.Hazards.ValidTime.EndTime.Format("01/02 15:04"))
	for _, str := range location.Hazards.HazardInfo.AffectedAreas {
		log.Printf("%s ", str.Name)
		m = m + fmt.Sprintf("%s ", str.Name)
	}
	m = m + "\n"
	log.Println("\n***************************************")

	return m
}
