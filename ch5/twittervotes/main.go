package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/nsqio/go-nsq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	ConsumerKey    string
	ConsumerSecret string
	AccessToken    string
	AccessSecret   string
	BearerToken    string
}

var (
	conn net.Conn
)

type poll struct {
	Options []string
}
type tweet struct {
	Text string
}

var reader io.ReadCloser

func closeConn() {
	if conn != nil {
		conn.Close()
	}
	if reader != nil {
		reader.Close()
	}
}

func dial(netw, addr string) (net.Conn, error) {
	if conn != nil {
		conn.Close()
		conn = nil
	}
	netc, err := net.DialTimeout(netw, addr, 5*time.Second)
	if err != nil {
		return nil, err
	}
	conn = netc
	return netc, nil
}

func main() {

	// 環境変数の読み込む
	err := godotenv.Load("../../.env")
	if err != nil {
		fmt.Printf("読み込み出来ませんでした: %v", err)
	}

	config := &Config{
		ConsumerKey:    os.Getenv("SP_TWITTER_KEY"),
		ConsumerSecret: os.Getenv("SP_TWITTER_SECRET"),
		AccessToken:    os.Getenv("SP_TWITTER_ACCESSTOKEN"),
		AccessSecret:   os.Getenv("SP_TWITTER_ACCESSSECRET"),
		BearerToken:    os.Getenv("BEARER_TOKEN"),
	}
	if config.ConsumerKey == "" {
		log.Fatal("Missing Twitter Consumer Key")
	}
	if config.ConsumerSecret == "" {
		log.Fatal("Missing Twitter Consumer Secret")
	}
	if config.AccessToken == "" {
		log.Fatal("Missing Twitter User Access Token")
	}
	if config.AccessSecret == "" {
		log.Fatal("Missing Twitter User Access Secret")
	}
	if config.BearerToken == "" {
		log.Fatal("Missing BearerToken")
	}

	// creds := &clientcredentials.Config{
	// 	ClientID:     config.AccessToken,
	// 	ClientSecret: config.AccessSecret,
	// }

	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial: dial,
		},
	}
	// twitterを止める用のチャンネルの作成
	twitterStopChan := make(chan struct{}, 1)
	// publisherを止める用のチャンネルの作成
	publisherStopChan := make(chan struct{}, 1)
	stop := false
	signalChan := make(chan os.Signal, 1)
	go func() {
		<-signalChan
		stop = true
		log.Println("Stopping...")
		closeConn()
	}()
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	votes := make(chan string)
	go func() {
		pub, _ := nsq.NewProducer("localhost:4150", nsq.NewConfig())
		for vote := range votes {
			pub.Publish("votes", []byte(vote))
		}
		log.Println("Publisher: Stopping")
		pub.Stop()
		log.Println("Publisher: Stopped")
		publisherStopChan <- struct{}{}
	}()
	go func() {
		defer func() {
			twitterStopChan <- struct{}{}
		}()
		for {
			if stop {
				log.Println("twitter: Stopped")
				return
			}
			time.Sleep(2 * time.Second)

			var opts []string
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			//TODO: ApplyURIの部分
			client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
			if err != nil {
				log.Fatal(err)
			}
			cur, err := client.Database("ballots").Collection("polls").Find(ctx, nil)
			if err != nil {
				log.Fatal(err)
			}
			var p poll
			for cur.Next(ctx) {
				if err = cur.Decode(&p); err != nil {
					log.Fatal(err)
				}
				opts = append(opts, p.Options...)
			}
			defer cur.Close(ctx)
			// defer client.Disconnect(ctx)
			defer func() { _ = client.Disconnect(ctx) }()

			hashtags := make([]string, len(opts))
			for i := range opts {
				hashtags[i] = "#" + strings.ToLower(opts[i])
			}

			form := url.Values{"track": {strings.Join(hashtags, ",")}}
			formEnc := form.Encode()

			u, _ := url.Parse("https://stream.twitter.com/1.1/statuses/filter.json")
			req, err := http.NewRequest("POST", u.String(), strings.NewReader(formEnc))
			if err != nil {
				log.Println("creating filter request failed:", err)
			}

			// req.Header.Set("Authorization", authClient.AuthorizationHeader(creds, "POST", u, form))
			var bearer = "Bearer " + config.BearerToken
			req.Header.Set("Authorization", bearer)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Content-Length", strconv.Itoa(len(formEnc)))

			resp, err := httpClient.Do(req)
			if err != nil {
				log.Println("Error getting response:", err)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				s := bufio.NewScanner(resp.Body)
				s.Scan()
				log.Println(s.Text())
				log.Println(hashtags)
				log.Println("StatusCode =", resp.StatusCode)
				continue
			}
			reader = resp.Body
			decoder := json.NewDecoder(reader)

			for {
				var t tweet
				if err := decoder.Decode(&t); err == nil {
					for _, option := range opts {
						if strings.Contains(
							strings.ToLower(t.Text),
							strings.ToLower(option),
						) {
							log.Println("vote:", option)
							votes <- option
						}
					}
				} else {
					break
				}
			}
		}
	}()

	go func() {
		for {
			time.Sleep(1 * time.Minute)
			closeConn()
			if stop {
				break
			}
		}
	}()
	<-twitterStopChan
	close(votes)
	<-publisherStopChan
}
