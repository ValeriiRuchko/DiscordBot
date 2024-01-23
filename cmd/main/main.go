package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"text/template"

	"github.com/bwmarrin/discordgo"
)

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
	// to further parse and unparse as JSON) and after use close with "defer res.Body.close()"
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

	// stringifiedBody := string(body)

	var parsedJSON ParsedBody

	err = json.Unmarshal(body, &parsedJSON)
	if err != nil {
		log.Fatalln(err)
	}

	// defer closing of Body stream of bytes until the function returns
	defer res.Body.Close()

	// what we send to Discord bot
	weatherReport := template.New("weatherReport")

	weatherReport.Parse(`Temp is {{.Main.Temp}} & Weather is {{$weather := index .Weather 0}}{{$weather.Main}}`)

	// intermediate buffer that implements io.Writer so can be used in weatherReport.Execute
	// (need to allocate with "new" so that buffer wouldn't point to nil)
	buf := new(bytes.Buffer)
	// finally giving our buffer with info
	err = weatherReport.Execute(buf, parsedJSON)
	if err != nil {
		log.Fatalln(err)
	}

	// bot initialization
	sess, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		log.Fatalln("Initialization of bot failed", err)
	}

	// ----- ----- ----- ----- ----- -----
	// NECESSARY ACTIONS TO ADD NEW "SLASH COMMAND"
	command := discordgo.ApplicationCommand{Name: "get-weather", Description: "Get current weather information"}

	getHandler := func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: buf.String(),
			},
		})
		fmt.Println("Handled")
	}
	// register interaction handler
	sess.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		getHandler(s, i)
	})
	// register interaction
	sess.ApplicationCommandCreate(os.Getenv("APP_ID"), "", &command)

	// ----- ----- ----- ----- ----- -----

	err = sess.Open()
	if err != nil {
		log.Fatalln("Couldn't open the session", err)
	}

	defer sess.Close()

	// so the program won't close, wtf
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	fmt.Println("Bot is up and running")
}
