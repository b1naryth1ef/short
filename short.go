package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/handlers"
	"gopkg.in/redis.v3"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var (
	// Flags
	host      = flag.String("h", "localhost", "Bind address to listen on")
	base      = flag.String("b", "http://localhost/", "Base URL for the shortener")
	port      = flag.String("p", "8080", "Port you want to listen on, defaults to 8080")
	redisConn = flag.String("r", "localhost:6379", "Redis Address, defaults to localhost:6379")
	authCode  = flag.String("a", "", "Authorization code")

	// Redis connection
	redisClient *redis.Client
)

type ShortendURL struct {
	LongURL string `json:"long_url"`
	ShortID string `json:"short_id"`

	// On return, the full short url
	ShortURL string `json:"short_url",omitempty`

	// Optionally the view count
	Views int `json:"views",omitempty`

	// Optionally the auth code
	AuthCode string `json:"auth_code",omitempty`
}

func (s *ShortendURL) GetViews() (int, error) {
	val, err := redisClient.Get(fmt.Sprintf("link:%v:views", s.ShortID)).Result()
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(val)
}

func HitShortendURL(id string) error {
	return redisClient.Incr(fmt.Sprintf("link:%v:views", id)).Err()
}

func GetShortendURL(id string) (*ShortendURL, error) {
	val, err := redisClient.Get(fmt.Sprintf("link:%v:url", id)).Result()
	if err != nil {
		return nil, err
	}

	return &ShortendURL{
		LongURL:  val,
		ShortID:  id,
		ShortURL: *base + id,
	}, nil
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	domain, err := GetShortendURL(r.URL.Path[1:])
	if err != nil {
		http.Error(w, "Failed to get long URL from short key", http.StatusBadRequest)
		return
	}

	err = HitShortendURL(r.URL.Path[1:])
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, domain.LongURL, http.StatusFound)
	return
}

func handleList(w http.ResponseWriter, r *http.Request) {
	var (
		keyId string
		links []*ShortendURL
	)

	val, err := redisClient.Keys("link:*:url").Result()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	for _, key := range val {
		keyId = strings.Split(key, ":")[1]

		// Grab the shortend url
		url, err := GetShortendURL(keyId)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// Grab the view count
		url.Views, err = url.GetViews()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		links = append(links, url)
	}

	output, err := json.Marshal(links)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(output)
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	var obj ShortendURL
	create, err := ioutil.ReadAll(r.Body)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = json.Unmarshal(create, &obj)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if obj.AuthCode != *authCode {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}

	err = redisClient.Set(fmt.Sprintf("link:%v:url", obj.ShortID), obj.LongURL, 0).Err()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	obj.AuthCode = ""
	output, err := json.Marshal(obj)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(output)
}

func main() {
	flag.Parse()

	// Connect to redis
	redisClient = redis.NewClient(&redis.Options{
		Addr:     *redisConn,
		PoolSize: 64,
	})

	server := http.NewServeMux()
	server.HandleFunc("/", handleIndex)
	server.HandleFunc("/links", handleList)
	server.HandleFunc("/links/create", handleCreate)

	// If the requests log doesnt exist, make it
	if _, err := os.Stat("requests.log"); os.IsNotExist(err) {
		ioutil.WriteFile("requests.log", []byte{}, 0600)
	}

	// Open the log file in append mode
	logFile, err := os.OpenFile("requests.log", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer logFile.Close()

	// Actually start the server
	loggedRouter := handlers.LoggingHandler(logFile, server)

	err = http.ListenAndServe(*host+":"+*port, loggedRouter)
	if err != nil {
		fmt.Println(err)
	}
}
