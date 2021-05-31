//main
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"TophandourNumberBot/config"
	"encoding/json"
	"io/ioutil"
	"net/url"
	"regexp"

	"github.com/coreos/pkg/flagutil"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"

	"github.com/bwmarrin/discordgo"
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
	"Ericacl50473675": "spam",
	"TrevorProject":   "Humanitarian",
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
	if (currentTweet.ExtendedTweet != nil && !phoneNumberRegex.MatchString(currentTweet.ExtendedTweet.FullText)) || !phoneNumberRegex.MatchString(currentTweet.Text) {
		fmt.Println("~~~~failed regex~~~~")
		shouldPost = false
	}
	_, blocked := blockList[currentTweet.User.ScreenName]
	if shouldPost && blocked {
		fmt.Println("~~~~blocked~~~~")
		shouldPost = false
	}
	if shouldPost {
		for i := range currentTweet.Entities.Hashtags {
			_, mutedHashtag := hashMute[currentTweet.Entities.Hashtags[i].Text]
			if mutedHashtag {
				fmt.Println("~~~~blocked #~~~~")
				shouldPost = false
				break
			}
		}
	}
	if shouldPost && currentTweet.RetweetedStatus != nil {
		_, rtblocked := blockList[currentTweet.RetweetedStatus.User.ScreenName]
		if rtblocked {
			fmt.Println("~~~~blocked RT~~~~")
			shouldPost = false
		}
		for i := range currentTweet.RetweetedStatus.Entities.Hashtags {
			_, mutedRTHashtag := hashMute[currentTweet.RetweetedStatus.Entities.Hashtags[i].Text]
			if mutedRTHashtag {
				fmt.Println("~~~~blocked #RT~~~~")
				shouldPost = false
				break
			}
		}
	}
	if shouldPost && currentTweet.Retweeted {
		fmt.Println("~~~~blocked RTd~~~~")
		shouldPost = false
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
		fmt.Println(tweet.Text)
		if shouldPostTweet(*tweet) {
			fmt.Println("posting >>> https://twitter.com/" + url.QueryEscape(tweet.User.ScreenName) + "/status/" + tweet.IDStr)
			postDiscord(discord, ">>> https://twitter.com/"+url.QueryEscape(tweet.User.ScreenName)+"/status/"+tweet.IDStr, configObject.ChannelIDString)
		}
	}

	fmt.Println("Starting Stream...")

	// FILTER
	filterParams := &twitter.StreamFilterParams{
		Track:         []string{"my number", "OR call me", "OR phone number", "OR call", "OR reach out", "OR text", "AND -whatsapp", "AND -is:retweet"},
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
