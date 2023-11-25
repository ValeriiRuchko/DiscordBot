package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

// here we define which command our bot has
var commands = []api.CreateCommandData{{
	Name:        "get-weather",
	Description: "Fetch weather from data-source!",
}}

// declaring variable which helps to answer to Discord "slash commands"
var r *cmdroute.Router = cmdroute.NewRouter()

type WeatherItem struct {
	Id          int
	Main        string
	Description string
}

type MainInfo struct {
	Temp       float64
	Feels_like float64
	Temp_min   float64
	Temp_max   float64
	Pressure   float64
	Humidity   float64
}

type ParsedBody struct {
	Weather []WeatherItem
	Main    MainInfo
}

func main() {
	// weather-api endpoint URL
	weatherApi := "https://api.openweathermap.org/data/2.5/weather?lat=50.0619474&lon=19.997153435836697&appid=" + os.Getenv("WEATHER_API_KEY") + "&units=metric"

	// getting pointer to response struct, and to get res.Body, we then have to
	// read it with io.ReadAll (it gives array of bytes so we have to convert it to string
	// to further parse and unparse as JSON) and after use close with defer res.Body.close()
	// (because in Go Body is like stream I guess)
	res, err := http.Get(weatherApi)
	if err != nil {
		log.Fatalln(err)
	}

	// here we read response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}
	// body version converted from []byte which we can print and parse to and from JSON
	stringBody := string(body)
	// need to be pointer so that we can write to it parsed JSON data
	var parsedJSON *ParsedBody

	err = json.Unmarshal(body, &parsedJSON)
	if err != nil {
		log.Fatalln(err)
	}

	// defer closing of Body stream of bytes until the function returns
	defer res.Body.Close()

	fmt.Println(parsedJSON.Weather[0].Main, stringBody)
	// what we send to Discord bot
	weatherReport := template.New("weatherReport")
	weatherReport.Parse(`Temp is {{.Main.Temp}} & Weather is {{$weather := index .Weather 0}}{{$weather.Main}}`)

	// intermediate buffer that implements io.Writer so can be used in weatherReport.Execute
	// (need to allocate with new so that buffer wouldn't point to nil)
	buf := new(bytes.Buffer)
	// finally giving our buffer with info
	err = weatherReport.Execute(buf, parsedJSON)
	if err != nil {
		log.Fatalln(err)
	}

	// fmt.Println(stringBody)
	// handler function which handles command "/get-weather (names must be the same)"
	r.AddFunc("get-weather", func(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponseData {
		return &api.InteractionResponseData{Content: option.NewNullableString(buf.String())}
	})

	// connecting to our bot with unique bot-token
	s := state.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	// permissions and description of actions of our bot
	s.AddInteractionHandler(r)
	s.AddIntents(gateway.IntentGuilds)

	fmt.Println("Bot started")

	if err := cmdroute.OverwriteCommands(s, commands); err != nil {
		log.Fatalln("cannot update commands:", err)
	}

	if err := s.Connect(context.TODO()); err != nil {
		log.Println("cannot connect:", err)
	}
}
