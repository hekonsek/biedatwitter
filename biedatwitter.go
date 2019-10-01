package main

import (
	"encoding/base64"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"os"
	"strings"
)
import (
	"context"
	"github.com/gin-gonic/gin"
	"regexp"
	"time"
)

// Application

func main() {
	err := NewBiedaTwitter().Start()
	if err != nil {
		panic(err)
	}
}

// Application

type biedaTwitter struct {
	srv *http.Server
}

func NewBiedaTwitter() *biedaTwitter {
	return &biedaTwitter{}
}

func (biedaTwitter *biedaTwitter) Start() error {
	tagFinder, err := createTagFinder()
	if err != nil {
		return err
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	mongoAddress := os.Getenv("MONGO_URL")
	if mongoAddress == "" {
		mongoAddress = "mongodb://localhost:27017"
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoAddress))
	if err != nil {
		return err
	}
	mongoTweets := client.Database("biedatwitter").Collection("tweets")

	r := gin.Default()
	r.Group("/", gin.BasicAuth(gin.Accounts{"henry": "secretpass"})).POST(
		"/tweet", func(c *gin.Context) {
			createTweetHandler(c, mongoTweets, tagFinder)
		})

	r.GET("/tweet/:tag", func(c *gin.Context) {
		tagTimelineHandler(c, mongoTweets)
	})

	r.Group("/", gin.BasicAuth(gin.Accounts{"admin": "admin"})).GET(
		"/admin/trending/:from/:to/:tag", func(c *gin.Context) {
			tagTrends(c, mongoTweets)
		})

	biedaTwitter.srv = &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	return biedaTwitter.srv.ListenAndServe()
}

func (biedatwitter *biedaTwitter) Stop() {
	if biedatwitter.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		biedatwitter.srv.Shutdown(ctx)
	}
}

// REST DTOs

type newTweet struct {
	Text string
}

type tweet struct {
	Text    string    `json:"text"`
	Author  string    `json:"author"`
	Created time.Time `json:"created"`
}

// REST helpers

func createTweetHandler(c *gin.Context, mongoTweets *mongo.Collection, tagFinder *regexp.Regexp) {
	tweetBytes, err := c.GetRawData()
	if err != nil {
		errorHandler(c, err)
		return
	}
	var newTweet newTweet
	err = json.Unmarshal(tweetBytes, &newTweet)
	if err != nil {
		errorHandler(c, err)
		return
	}
	tags := findTags(tagFinder, newTweet.Text)

	username, err := resolveUsername(c)
	if err != nil {
		errorHandler(c, err)
		return
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = mongoTweets.InsertOne(ctx, bson.M{
		"text": newTweet.Text, "tags": tags, "created": time.Now(), "author": username,
	})
	if err != nil {
		errorHandler(c, err)
		return
	}

	c.JSON(200, gin.H{
		"status": "success",
		"tags":   tags,
	})
}

func tagTimelineHandler(c *gin.Context, mongoTweets *mongo.Collection) {
	tag := c.Param("tag")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	tweetsCursor, err := mongoTweets.Find(ctx, bson.M{"tags": bson.M{"$in": []string{tag}}},
	// Tag timeline displays last 100 tweets
	options.Find().SetLimit(100).SetSort(bson.M{"created": -1},
	))
	if err != nil {
		errorHandler(c, err)
		return
	}

	var tweets = []tweet{}
	defer tweetsCursor.Close(ctx)
	for tweetsCursor.Next(ctx) {
		var tweetDocument bson.M
		err := tweetsCursor.Decode(&tweetDocument)
		if err != nil {
			errorHandler(c, err)
			return
		}
		tweets = append(tweets, tweet{
			Text:    tweetDocument["text"].(string),
			Author:  tweetDocument["author"].(string),
			Created: tweetDocument["created"].(primitive.DateTime).Time(),
		})
	}

	c.JSON(200, gin.H{"tweets": tweets})
}

func tagTrends(c *gin.Context, mongoTweets *mongo.Collection) {
	tag := c.Param("tag")
	layout := "2006-01-02T15:04:05.000Z"
	from, err := time.Parse(layout, c.Param("from")+"-01-01T11:45:26.371Z")
	if err != nil {
		errorHandler(c, err)
		return
	}
	to, err := time.Parse(layout, c.Param("to")+"-12-31T11:45:26.371Z")
	if err != nil {
		errorHandler(c, err)
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	count, err := mongoTweets.CountDocuments(ctx, bson.M{"tags": bson.M{"$in": []string{tag}}, "created": bson.M{
		"$gte": from, "$lte": to,
	}})
	if err != nil {
		errorHandler(c, err)
		return
	}

	c.JSON(200, gin.H{"count": count})
}

func errorHandler(c *gin.Context, err error) {
	c.JSON(500, gin.H{"error": err.Error()})
}

func resolveUsername(c *gin.Context) (string, error) {
	basicHeader := c.GetHeader("Authorization")
	basicToken := strings.Replace(basicHeader, "Basic ", "", 1)
	decodedBytes, err := base64.StdEncoding.DecodeString(basicToken)
	if err != nil {
		return "", err
	}
	return strings.Split(string(decodedBytes), ":")[0], nil
}

// Tag parsing logic

func createTagFinder() (*regexp.Regexp, error) {
	tagFinder, err := regexp.Compile("#[a-zA-Z0-9_]*")
	if err != nil {
		return nil, err
	}
	return tagFinder, err
}

func findTags(tagFinder *regexp.Regexp, text string) []string {
	tags := tagFinder.FindAllString(text, -1)
	for i, tag := range tags {
		tags[i] = tag[1:] // Remove hash prefix
	}
	return tags
}