//main
package main

import (
	"TophandourNumberBot/config"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	//"os"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/dghubble/go-twitter/twitter"
)

var phoneNumberRegex = regexp.MustCompile(`\(?\d{3}\)?[-\.]? *\d{3}[-\.]? *[-\.]?\d{4}`)
var blockList = map[string]string{
	"BloodAid":        "medical",
	"thesneakerheist": "too spammy",
}

func getTweets(bearerString string) []twitter.Tweet {
	urlQuery := "https://api.twitter.com/1.1/search/tweets.json?q=" + url.QueryEscape(`("my number") OR ("call me") OR ("phone number") OR ("call") OR ("reach out")`) + "&result_type=recent&tweet_mode=extended&count=100&lang=en"

	client := &http.Client{}

	req, _ := http.NewRequest("GET", urlQuery, nil)

	req.Header.Add("Authorization", "Bearer "+bearerString)

	resp, _ := client.Do(req)
	defer resp.Body.Close()
	bodybytes, _ := ioutil.ReadAll(resp.Body)
	responseObject := twitter.Search{}
	json.Unmarshal(bodybytes, &responseObject)
	return responseObject.Statuses
}

func postDiscord(discord *discordgo.Session, message string, channelString string) {
	discord.ChannelMessageSend(channelString, message)
}

func main() {
	fileBytes, _ := ioutil.ReadFile("TophandourNumberBot.config")
	configObject := config.Configuration{}
	json.Unmarshal(fileBytes, &configObject)
	discord, _ := discordgo.New("Bot " + configObject.BotSecretString)
	discord.Open()

	tweetList := getTweets(configObject.TweetBearerString)
	for i := range tweetList {
		_, blocked := blockList[tweetList[i].User.ScreenName]
		if phoneNumberRegex.MatchString(tweetList[i].FullText) && !blocked {
			postDiscord(discord, ">>> https://twitter.com/"+url.QueryEscape(tweetList[i].User.ScreenName)+"/status/"+tweetList[i].IDStr, configObject.ChannelIDString)
		}
	}
	discord.Close()
}
