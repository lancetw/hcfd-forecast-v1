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

// ResultRaining struct
type ResultRaining struct {
	Location []Location0 `xml:"location"`
}

// ResultWarning struct
type ResultWarning struct {
	Location []Location1 `xml:"dataset>location"`
}

const baseURL = "http://opendata.cwb.gov.tw/opendataapi?dataid="
const authKey = "CWB-FB35C2AC-9286-4B7E-AD11-6BBB7F2855F7"
const timeZone = "Asia/Taipei"

func fetchXML(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("fetchXML http.Get error: %v", err)
		os.Exit(1)
	}

	xmldata, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Printf("fetchXML ioutil.ReadAll error: %v", err)
		return nil
	}

	return xmldata
}

// GetRainingInfo "雨量警示"
func GetRainingInfo(targets []string, noLevel bool) ([]string, string) {
	var token = "O-A0002-001 "
	var msgs = []string{}

	rainLevel := map[string]float32{
		"10minutes": 5,  // 5
		"1hour":     20, // 20
	}

	url := baseURL + "O-A0002-001" + "&authorizationkey=" + authKey
	xmldata := fetchXML(url)

	v := ResultRaining{}
	err := xml.Unmarshal([]byte(xmldata), &v)
	if err != nil {
		log.Printf("GetRainingInfo fetchXML error: %v", err)
		return []string{}, ""
	}

	log.Printf("[取得 %d 筆地區雨量資料]\n", len(v.Location))

	for _, location := range v.Location {
		var msg string
		for _, parameter := range location.Parameter {
			if parameter.Name == "CITY" {
				for _, target := range targets {
					if parameter.Value == target {
						for _, element := range location.WeatherElement {
							token = location.Time.Format("20060102150405")

							switch element.Name {
							case "MIN_10":
								if noLevel {
									if element.Value <= 0 {
										msg = msg + fmt.Sprintf("%s：%s", "$ 10分鐘雨量 $", "-")
									} else {
										msg = msg + fmt.Sprintf("%s：%.1f", "$ 10分鐘雨量 $", element.Value)
									}
								} else {
									if element.Value <= 0 {
										//log.Printf("%s：%s", "*10分鐘雨量*", "-")
									} else {
										//log.Printf("%s：%.1f", "*10分鐘雨量*", element.Value)
										if element.Value >= rainLevel["10minutes"] {
											msg = msg + fmt.Sprintf("【%s】豪大雨警報\n%s：%.1f \n", location.Name, "$ 10分鐘雨量 $", element.Value)
										}
									}
								}

							case "RAIN":
								if noLevel {
									if element.Value <= 0 {
										msg = msg + fmt.Sprintf("【%s】\n%s：%s\n", location.Name, "(時雨量)", "-")
									} else {
										msg = msg + fmt.Sprintf("【%s】\n%s：%.1f\n", location.Name, "(時雨量)", element.Value)
									}
								} else {
									if element.Value <= 0 {
										//log.Printf("[%s]", location.Name)
										//log.Printf("%s：%s", "時雨量", "-")
									} else {
										//log.Printf("[%s]", location.Name)
										//log.Printf("%s：%.1f", "時雨量", element.Value)
										if element.Value >= rainLevel["1hour"] {
											msg = msg + fmt.Sprintf("【%s】豪大雨警報\n%s：%.1f \n", location.Name, "(時雨量)", element.Value)
										}
									}
								}
							}
						}
					}
				}
			}
		}

		if msg != "" {
			msgs = append(msgs, msg)
		}
	}

	return msgs, token
}

// GetWarningInfo "豪大雨特報"
func GetWarningInfo(targets []string) ([]string, string) {
	var token = "W-C0033-001 "
	var msgs = []string{}

	url := baseURL + "W-C0033-001" + "&authorizationkey=" + authKey
	xmldata := fetchXML(url)

	v := ResultWarning{}
	err := xml.Unmarshal([]byte(xmldata), &v)
	if err != nil {
		log.Printf("GetWarningInfo fetchXML error: %v", err)
		return []string{}, ""
	}

	log.Printf("[取得 %d 筆地區天氣警報資料]\n", len(v.Location))

	local := time.Now()
	location, err := time.LoadLocation(timeZone)
	if err == nil {
		local = local.In(location)
	}

	var hazardmsgs = ""

	for i, location := range v.Location {
		if i == 0 {
			token = token + location.Hazards.ValidTime.StartTime.Format("20060102150405") + " " + location.Hazards.ValidTime.EndTime.Format("20060102150405")
		}
		if location.Hazards.Info.Phenomena != "" && location.Hazards.ValidTime.EndTime.After(local) {
			if targets != nil {
				for _, name := range targets {
					if name == location.Name {
						hazardmsgs = hazardmsgs + saveHazards(location) + "\n"
					}
				}
			} else {
				hazardmsgs = hazardmsgs + saveHazards(location) + "\n"
			}
		}
	}

	if hazardmsgs != "" {
		msgs = append(msgs, hazardmsgs)
	}

	return msgs, token
}

func saveHazards(location Location1) string {
	var m string

	//log.Printf("【%s】%s%s\n %s ~\n %s\n", location.Name, location.Hazards.Info.Phenomena, location.Hazards.Info.Significance, location.Hazards.ValidTime.StartTime.Format("01/02 15:04"), location.Hazards.ValidTime.EndTime.Format("01/02 15:04"))
	m = fmt.Sprintf("【%s】%s%s\n %s ~\n %s\n", location.Name, location.Hazards.Info.Phenomena, location.Hazards.Info.Significance, location.Hazards.ValidTime.StartTime.Format("01/02 15:04"), location.Hazards.ValidTime.EndTime.Format("01/02 15:04"))
	if len(location.Hazards.HazardInfo.AffectedAreas) > 0 {
		//log.Printf("影響地區：")
		m = m + "影響地區："
		for _, str := range location.Hazards.HazardInfo.AffectedAreas {
			//log.Printf("%s ", str.Name)
			m = m + fmt.Sprintf("%s ", str.Name)
		}
	}

	return m
}
