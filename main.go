//main
package main

import (
	"TophandourNumberBot/config"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/coreos/pkg/flagutil"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"

	"github.com/bwmarrin/discordgo"
)

var phoneNumberRegex = regexp.MustCompile(`(\+?1{1}[-.● ]?)?[(]?([2-9]{1}[0-9]{2})\)?[-.● ]?([1-9]{1}[0-9]{2})[-.● ]?([0-9]{4})`)
var blockList = map[string]string{
	"BloodAid":        "medical",
	"bloodreqbot":     "medical",
	"thesneakerheist": "too spammy",
	"PayanSneakers":   "too spammy",
	"malikiumar085":   "too spammy",
	"covidinfoleads":  "medical",
	"ITFobOffBot":     "joke",
	"Ericacl50473675": "spam",
	"TrevorProject":   "Humanitarian",
	"TwitterSupport":  "irrelevant",
}
var hashMute = map[string]string{
	"BloodAid":      "medical",
	"BloodMatters":  "medical",
	"PayanSneakers": "too spammy",
}

func postDiscord(discord *discordgo.Session, message string, channelString string) {
	discord.ChannelMessageSend(channelString, message)
}

func shouldPostTweet(currentTweet twitter.Tweet) bool {
	shouldPost := true
	if shouldPost && currentTweet.RetweetedStatus != nil {
		//fmt.Println("~~~~blocked RTd~~~~")
		shouldPost = false
	}
	if shouldPost && currentTweet.Place != nil && currentTweet.Place.CountryCode != "" &&
		currentTweet.Place.CountryCode != "US" && currentTweet.Place.CountryCode != "CA" &&
		currentTweet.Place.CountryCode != "EN" {
		//fmt.Println("~~~~failed country code~~~~")
		shouldPost = false
	}
	if shouldPost && ((currentTweet.ExtendedTweet == nil &&
		strings.Contains(strings.ToLower(currentTweet.Text), "+91")) ||
		(currentTweet.ExtendedTweet != nil && strings.Contains(strings.ToLower(currentTweet.ExtendedTweet.FullText), "+91"))) {
		//fmt.Println("~~~~failed India area code~~~~")
		shouldPost = false
	}
	if shouldPost && ((currentTweet.ExtendedTweet == nil &&
		strings.Contains(strings.ToLower(currentTweet.Text), "whatsapp")) ||
		(currentTweet.ExtendedTweet != nil && strings.Contains(strings.ToLower(currentTweet.ExtendedTweet.FullText), "whatsapp"))) {
		//fmt.Println("~~~~failed watsapp regex~~~~")
		shouldPost = false
	}
	if shouldPost && ((currentTweet.ExtendedTweet == nil &&
		!phoneNumberRegex.MatchString(currentTweet.Text)) ||
		(currentTweet.ExtendedTweet != nil && !phoneNumberRegex.MatchString(currentTweet.ExtendedTweet.FullText))) {
		//fmt.Println("~~~~failed regex~~~~")
		shouldPost = false
	}
	_, blocked := blockList[currentTweet.User.ScreenName]
	if shouldPost && blocked {
		//fmt.Println("~~~~blocked~~~~")
		shouldPost = false
	}
	if shouldPost {
		for i := range currentTweet.Entities.Hashtags {
			_, mutedHashtag := hashMute[currentTweet.Entities.Hashtags[i].Text]
			if mutedHashtag {
				//fmt.Println("~~~~blocked #~~~~")
				shouldPost = false
				break
			}
		}
	}
	if shouldPost && currentTweet.RetweetedStatus != nil {
		_, rtblocked := blockList[currentTweet.RetweetedStatus.User.ScreenName]
		if rtblocked {
			//fmt.Println("~~~~blocked RT~~~~")
			shouldPost = false
		}
		for i := range currentTweet.RetweetedStatus.Entities.Hashtags {
			_, mutedRTHashtag := hashMute[currentTweet.RetweetedStatus.Entities.Hashtags[i].Text]
			if mutedRTHashtag {
				//fmt.Println("~~~~blocked #RT~~~~")
				shouldPost = false
				break
			}
		}
	}

	return shouldPost
}

func tweetStream(discord *discordgo.Session, configObject config.Configuration) {
	flags := flag.NewFlagSet("user-auth", flag.ExitOnError)
	consumerKey := flags.String("consumer-key", configObject.TwitterAPI, "Twitter Consumer Key")
	consumerSecret := flags.String("consumer-secret", configObject.TwitterAPISecret, "Twitter Consumer Secret")
	accessToken := flags.String("access-token", configObject.TwitterAccess, "Twitter Access Token")
	accessSecret := flags.String("access-secret", configObject.TwitterAccessSecret, "Twitter Access Secret")
	flags.Parse(os.Args[1:])
	flagutil.SetFlagsFromEnv(flags, "TWITTER")

	if *consumerKey == "" || *consumerSecret == "" || *accessToken == "" || *accessSecret == "" {
		log.Fatal("Consumer key/secret and Access token/secret required")
	}

	config := oauth1.NewConfig(*consumerKey, *consumerSecret)
	token := oauth1.NewToken(*accessToken, *accessSecret)
	// OAuth1 http.Client will automatically authorize Requests
	httpClient := config.Client(oauth1.NoContext, token)

	// Twitter Client
	client := twitter.NewClient(httpClient)

	// Convenience Demux demultiplexed stream messages
	demux := twitter.NewSwitchDemux()
	demux.Tweet = func(tweet *twitter.Tweet) {
		//fmt.Println(tweet.Text)
		if shouldPostTweet(*tweet) {
			fmt.Println("posting >>> https://twitter.com/" + url.QueryEscape(tweet.User.ScreenName) + "/status/" + tweet.IDStr)
			if (tweet.Entities != nil && tweet.Entities.Media != nil && len(tweet.Entities.Media) > 0) ||
				(tweet.ExtendedEntities != nil && tweet.ExtendedEntities.Media != nil && len(tweet.ExtendedEntities.Media) > 0) ||
				(tweet.ExtendedTweet != nil && tweet.ExtendedTweet.Entities != nil && len(tweet.ExtendedTweet.Entities.Media) > 0) {
				postDiscord(discord, ">>> https://twitter.com/"+url.QueryEscape(tweet.User.ScreenName)+"/status/"+tweet.IDStr, configObject.MediaChannelIDString)
			} else {
				postDiscord(discord, ">>> https://twitter.com/"+url.QueryEscape(tweet.User.ScreenName)+"/status/"+tweet.IDStr, configObject.ChannelIDString)
			}
		}
	}

	fmt.Println("Starting Stream...")

	// FILTER
	filterParams := &twitter.StreamFilterParams{
		Track:         []string{"my number", "call me", "phone number", "call", "reach out", "text", "reach me", "-whatsapp -is:retweet"},
		Language:      []string{"en"},
		StallWarnings: twitter.Bool(true),
	}
	stream, err := client.Streams.Filter(filterParams)
	if err != nil {
		log.Fatal(err)
	}

	// Receive messages until stopped or stream quits
	go demux.HandleChan(stream.Messages)

	// Wait for SIGINT and SIGTERM (HIT CTRL-C)
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	fmt.Println("Stopping Stream...")
	stream.Stop()
}

func main() {
	fileBytes, _ := ioutil.ReadFile("TophandourNumberBot.config")
	configObject := config.Configuration{}
	error := json.Unmarshal(fileBytes, &configObject)
	if error != nil {
		fmt.Println(error.Error())
	}
	discord, _ := discordgo.New("Bot " + configObject.BotSecretString)
	discord.Open()
	tweetStream(discord, configObject)
	discord.Close()
}
