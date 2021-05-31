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
	"bloodreqbot":     "medical",
	"thesneakerheist": "too spammy",
	"PayanSneakers":   "too spammy",
	"malikiumar085":   "too spammy",
	"covidinfoleads":  "medical",
	"ITFobOffBot":     "joke",
}
var hashMute = map[string]string{
	"BloodAid":      "medical",
	"BloodMatters":  "medical",
	"PayanSneakers": "too spammy",
}
var queryString = `("my number") OR ("call me") OR ("phone number") OR ("call") OR ("reach out") OR ("text") -WhatsApp`

func getTweets(bearerString string) []twitter.Tweet {
	urlQuery := "https://api.twitter.com/1.1/search/tweets.json?q=" + url.QueryEscape(queryString) + "&result_type=recent&tweet_mode=extended&count=100&include_entities=true&lang=en"

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

func shouldPostTweet(currentTweet twitter.Tweet) bool {
	shouldPost := true
	if !phoneNumberRegex.MatchString(currentTweet.FullText) {
		shouldPost = false
	}
	_, blocked := blockList[currentTweet.User.ScreenName]
	if shouldPost && blocked {
		shouldPost = false
	}
	if shouldPost {
		for i := range currentTweet.Entities.Hashtags {
			_, mutedHashtag := hashMute[currentTweet.Entities.Hashtags[i].Text]
			if mutedHashtag {
				shouldPost = false
				break
			}
		}
	}
	if shouldPost && currentTweet.RetweetedStatus != nil {
		_, rtblocked := blockList[currentTweet.RetweetedStatus.User.ScreenName]
		if rtblocked {
			shouldPost = false
		}
		for i := range currentTweet.RetweetedStatus.Entities.Hashtags {
			_, mutedRTHashtag := hashMute[currentTweet.RetweetedStatus.Entities.Hashtags[i].Text]
			if mutedRTHashtag {
				shouldPost = false
				break
			}
		}
	}

	return shouldPost
}

func main() {
	fileBytes, _ := ioutil.ReadFile("TophandourNumberBot.config")
	configObject := config.Configuration{}
	json.Unmarshal(fileBytes, &configObject)
	discord, _ := discordgo.New("Bot " + configObject.BotSecretString)
	discord.Open()

	tweetList := getTweets(configObject.TweetBearerString)
	for i := range tweetList {
		if shouldPostTweet(tweetList[i]) {
			postDiscord(discord, ">>> https://twitter.com/"+url.QueryEscape(tweetList[i].User.ScreenName)+"/status/"+tweetList[i].IDStr, configObject.ChannelIDString)
		}
	}
	discord.Close()
}
