// Copyright (C) 2015 Josh Vega. All Rights Reserved.

package main

import (
	"html/template"
	"io/ioutil"
	"encoding/json"
	"net/http"
	"time"
	"strconv"
)

type WeatherData struct {
	LocationName string
	LocationId string
	TargetTime string

	SunriseTime string
	SunsetTime string

	Temperature string
	Humidity string

	WindSpeed float64
	WindDirection string
	WindGusts float64

	CloudCoverage float64

	WeatherDescription string

	RainHeight float64
	SnowHeight float64
}

type DetailedWeather struct {
	LocationName string
	LocationId string
	Count int
	Data []WeatherData
}

type MultiDayWeather struct {
	LocationName string
	Count int
	Data []struct {
		TempDay float64
		TempMin float64
		TempMax float64
		TempNight float64
		TempEve float64
		TempMorn float64

		Humidity float64

		WindSpeed float64
		WindDirection string
		WindGusts float64

		CloudCoverage float64

		RainHeight float64
		SnowHeight float64
	}
}

func parseTimestamp(t1 float64) string {
	var t2 time.Time
	t2 = time.Unix(int64(t1), 0)
	return t2.Format("Monday, January 2 2006 @ 03:04:05PM (MST)")
}

func parseTime(t1 float64) string {
	var t2 time.Time
	t2 = time.Unix(int64(t1), 0)
	return t2.Format("03:04:05pm (MST)")
}

func parseDateTime(t1 float64) string {
	var t2 time.Time
	t2 = time.Unix(int64(t1), 0)
	return t2.Format("Mon 3:04pm")
}

func parseWindDir(dir float64) string {
	if dir > 360 { dir -= 360 }

	if dir >= 348.75 && dir < 11.25 { return "N" }
	if dir >= 11.25 && dir < 33.75 { return "NNE" }
	if dir >= 33.75 && dir < 56.25 { return "NE" }
	if dir >= 56.25 && dir < 78.75 { return "ENE" }
	if dir >= 78.75 && dir < 101.25 { return "E" }
	if dir >= 101.25 && dir < 123.75 { return "ESE" }
	if dir >= 123.75 && dir < 146.25 { return "SE" }
	if dir >= 146.25 && dir < 168.75 { return "SSE" }
	if dir >= 168.75 && dir < 191.25 { return "S" }
	if dir >= 191.25 && dir < 213.75 { return "SSW" }
	if dir >= 213.25 && dir < 236.25 { return "SW" }
	if dir >= 236.25 && dir < 258.75 { return "WSW" }
	if dir >= 258.75 && dir < 281.25 { return "W" }
	if dir >= 281.25 && dir < 303.75 { return "WNW" }
	if dir >= 303.75 && dir < 326.25 { return "NW" }
	if dir >= 326.25 && dir < 348.75 { return "NNW" }
	
	return "UNDEFINED"
}

func parseWeatherData(data interface{}, timestamp bool) WeatherData {
	var ret WeatherData

	// Default Data
	ret.WindDirection = "Unavailable"

	tmp := data.(map[string]interface{})
	for key, val := range tmp {
		switch key {
		case "dt":
			if (timestamp) {
				ret.TargetTime = parseTimestamp(val.(float64))
			} else {
				ret.TargetTime = parseDateTime(val.(float64))
			}
		case "sys":
			val2 := val.(map[string]interface{})
			for key3, val3 := range val2 {
				switch key3 {
				case "sunrise":
					ret.SunriseTime = parseTime(val3.(float64))
				case "sunset":
					ret.SunsetTime = parseTime(val3.(float64))
				}
			}
		case "main":
			val2 := val.(map[string]interface{})
			for key3, val3 := range val2 {
				switch key3 {
				case "temp":
					ret.Temperature = strconv.FormatFloat(val3.(float64) - 273.15, 'f', 2, 32)
				case "humidity":
					ret.Humidity = strconv.FormatFloat(val3.(float64), 'f', 0, 32)
				}
			}
		case "weather":
			val2 := val.([]interface{})
			val3 := val2[0].(map[string]interface{})
			ret.WeatherDescription = val3["description"].(string)
		case "wind":
			val2 := val.(map[string]interface{})
			for key3, val3 := range val2 {
				switch key3 {
				case "speed":
					ret.WindSpeed = val3.(float64)
				case "deg":
					ret.WindDirection = parseWindDir(val3.(float64))
				case "gust":
					ret.WindGusts = val3.(float64)
				}
			}
		case "clouds":
			val2 := val.(map[string]interface{})
			ret.CloudCoverage = val2["all"].(float64)
		case "rain":
			val2 := val.(map[string]interface{})
			ret.RainHeight = val2["3h"].(float64)
		case "snow":
			val2 := val.(map[string]interface{})
			ret.SnowHeight = val2["3h"].(float64)
		}
	}

	return ret
}

func parseCurrent(res *http.Response) WeatherData {
	body, _ := ioutil.ReadAll(res.Body)

	var tmp interface{}
	json.Unmarshal(body, &tmp)

	return parseWeatherData(tmp, true)
}

func parseDetailed(res *http.Response) DetailedWeather {
	var ret DetailedWeather

	body, _ := ioutil.ReadAll(res.Body)
	var tmp interface{}
	json.Unmarshal(body, &tmp)

	data := tmp.(map[string]interface{})
	for key, val := range data {
		switch key {
		case "cnt":
			ret.Count = int(val.(float64))
		case "list":
			val2 := val.([]interface{})
			ret.Data = make([]WeatherData, ret.Count)
			for i := 0; i < ret.Count; i++ {
				ret.Data[i] = parseWeatherData(val2[i], false)
			}
		}
	}

	return ret
}

func parseMultiDay(res *http.Response) MultiDayWeather {
	var ret MultiDayWeather

	body, _ := ioutil.ReadAll(res.Body)
	var tmp interface{}
	json.Unmarshal(body, &tmp)

	data := tmp.(map[string]interface{})
	for key, val := range data {
		switch key {
		case "cnt":
			ret.Count = val.(int)
		case "list":
			val2 := val.(map[int]interface{})
			for i := 0; i < ret.Count; i++ {
				val3 := val2[i].(map[string]interface{})
				for key4, val4 := range val3 {
					switch key4 {
					case "dt":
						// TODO
					case "temp":
						val5 := val4.(map[string]interface{})
						ret.Data[i].TempDay = val5["day"].(float64)
						ret.Data[i].TempMin = val5["min"].(float64)
						ret.Data[i].TempMax = val5["max"].(float64)
						ret.Data[i].TempNight = val5["night"].(float64)
						ret.Data[i].TempEve = val5["eve"].(float64)
						ret.Data[i].TempMorn = val5["morn"].(float64)
					case "humidity":
						ret.Data[i].Humidity = val4.(float64)
					case "wind":
						val5 := val4.(map[string]interface{})
						ret.Data[i].WindSpeed = val5["speed"].(float64)
						// TODO: Wind Direction
						ret.Data[i].WindGusts = val5["gust"].(float64)
					case "clouds":
						val5 := val4.(map[string]interface{})
						ret.Data[i].CloudCoverage = val5["all"].(float64)
					case "rain":
						val5 := val4.(map[string]interface{})
						ret.Data[i].RainHeight = val5["3h"].(float64)
					case "snow":
						val5 := val4.(map[string]interface{})
						ret.Data[i].SnowHeight = val5["3h"].(float64)
					}
				}
			}
		}
	}

	return ret
}

func main() {
	locations := make(map[string]string)

	var templates = template.Must(template.ParseGlob("templates/*"))

	// Sample data
	locations["5128581"] = "New York City, New York"
	locations["5368361"] = "Los Angeles, California"
	locations["4684888"] = "Dallas, Texas"
	locations["2643743"] = "London, England"
	locations["524901"] = "Moscow, Russia"

	// Shows the current weather information for each of the selected locations.
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		weather := make([]WeatherData, len(locations))

		i := 0
		for id, name := range locations {
			resp, _ := http.Get("http://api.openweathermap.org/data/2.5/weather?id=" + id)
			weather[i] = parseCurrent(resp)
			weather[i].LocationName = name
			weather[i].LocationId = id
			i++
		}

		data := map[string]interface{} {
			"title": "Lirix | Overview",
			"weatherdata": weather,
		}

		templates.ExecuteTemplate(res, "index", data)
	})

	http.HandleFunc("/about", func(res http.ResponseWriter, req *http.Request) {
		templates.ExecuteTemplate(res, "about", nil)
	})

	http.HandleFunc("/help", func(res http.ResponseWriter, req *http.Request) {
		templates.ExecuteTemplate(res, "help", nil)
	})

	// Get detailed information about a single location.
	http.HandleFunc("/detail", func(res http.ResponseWriter, req *http.Request) {
		qs := req.URL.Query()

		var weather DetailedWeather

		resp, _ := http.Get("http://api.openweathermap.org/data/2.5/forecast?id=" + qs.Get("location"))
		weather = parseDetailed(resp)
		weather.LocationName = locations[qs.Get("location")]
		weather.LocationId = qs.Get("location")

		data := map[string]interface{} {
			"title": "Lirix | Detail",
			"weather": weather,
		}

		templates.ExecuteTemplate(res, "detail", data)
	})

	// Get multiday information about a single location.
	http.HandleFunc("/multiday", func(res http.ResponseWriter, req *http.Request) {
		//qs := req.URL.Query()
		//var weather MultiDayWeather

		//resp, _ := http.Get("http://api.openweathermap.org/data/2.5/forecast/daily?id=" + qs.Get("location"))
		//weather = parseMultiDay(resp)
		// TODO: Add location information

		templates.ExecuteTemplate(res, "multiday", nil)
	})

	http.ListenAndServe(":3000", nil)
}