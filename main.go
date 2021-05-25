//main
package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/dghubble/go-twitter/twitter"
)

var phoneNumberRegex = regexp.MustCompile(`\(?\d{3}\)?[-\.]? *\d{3}[-\.]? *[-\.]?\d{4}`)

func getTweets(bearerString string) []twitter.Tweet {
	urlQuery := "https://api.twitter.com/1.1/search/tweets.json?q=" + url.QueryEscape(`("my number") OR ("call me") OR ("phone number") OR ("call") OR ("reach out")`) + "&result_type=recent&tweet_mode=extended&count=100&lang=en"

	client := &http.Client{}

	req, _ := http.NewRequest("GET", urlQuery, nil)

	req.Header.Add("Authorization", "Bearer "+bearerString)

	resp, _ := client.Do(req)
	defer resp.Body.Close()
	bodybytes, _ := ioutil.ReadAll(resp.Body)
	bodyAsString := string(bodybytes)
	responseObject := twitter.Search{}
	json.Unmarshal([]byte(bodyAsString), &responseObject)
	return responseObject.Statuses
}

func postDiscord(message string, botString string, channelString string) {
	discord, _ := discordgo.New("Bot " + botString)
	discord.Open()
	discord.ChannelMessageSend(channelString, message)
	discord.Close()
}

func main() {
	tweetBearerString := string(os.Args[1])
	botSecretString := string(os.Args[2])
	channelIDString := string(os.Args[3])

	tweetList := getTweets(tweetBearerString)
	for i := range tweetList {
		if phoneNumberRegex.MatchString(tweetList[i].FullText) {
			postDiscord(">>> https://twitter.com/i/web/status/"+tweetList[i].IDStr, botSecretString, channelIDString)
		}
	}
}
