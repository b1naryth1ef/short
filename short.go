package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"gopkg.in/redis.v3"
	"io/ioutil"
	"net/http"
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

	// Optionally the auth code
	AuthCode string `json:"auth_code",omitempty`
}

// link:id:views - int
// link:id:url - string

func NewShortendURL(obj ShortendURL) error {
	return redisClient.Set(fmt.Sprintf("link:%v:url", obj.ShortID), obj.LongURL, 0).Err()
}

func GetShortendURL(id string) (*ShortendURL, error) {
	err := redisClient.Incr(fmt.Sprintf("link:%v:views", id)).Err()
	if err != nil {
		return nil, err
	}

	val, err := redisClient.Get(fmt.Sprintf("link:%v:url", id)).Result()
	if err != nil {
		return nil, err
	}

	return &ShortendURL{
		LongURL: val,
		ShortID: id,
	}, nil
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	domain, err := GetShortendURL(r.URL.Path[1:])
	if err != nil {
		http.Error(w, "Failed to get long URL from short key", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, domain.LongURL, http.StatusFound)
	return
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

	err = NewShortendURL(obj)
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

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/create", handleCreate)
	err := http.ListenAndServe(*host+":"+*port, nil)
	if err != nil {
		fmt.Println(err)
	}
}
