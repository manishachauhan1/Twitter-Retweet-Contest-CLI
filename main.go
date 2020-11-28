package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

func main() {

	var (
		twitterid  string
		keyfile    string
		usersFile  string
		numWinners int
	)

	flag.StringVar(&keyfile, "keyfile", "keys.json", "Enter your key file name ")
	// dfault tweetid = 1267788745492791296
	flag.StringVar(&twitterid, "id", "1267788745492791296", "Enter your twitter tweet id ")
	flag.StringVar(&usersFile, "user", "users.csv", "The file where users have retweeted the tweet are stored. This will be created if dont exists")
	flag.IntVar(&numWinners, "winners", 0, "The number of winners to pick for the contest.")
	flag.Parse()

	_, _, token, err := keys(keyfile)
	if err != nil {
		panic(err)
	}

	// can obtain bearer token like this too
	/*Client, err := twitterClient(key, secret)
	if err!= nil{
		panic(err)
	}*/
	//obtaining bearer token
	var conf oauth2.Config
	client := conf.Client(context.Background(), &token)

	// getting username of those who retweeted
	newUsernames, err := Retweeters(client, twitterid)
	if err != nil {
		panic(err)
	}
	existUsernames := existing(usersFile)
	//fmt.Println(existUsernames)
	allUsernames := merge(newUsernames, existUsernames)
	err = writeUsers(usersFile, allUsernames)
	if err != nil {
		panic(err)
	}
	//fmt.Println(allUsernames)
	if numWinners == 0 {
		return
	}
	winners := pickWinners(existUsernames, numWinners)
	fmt.Println("The winners are:")
	for _, username := range winners {
		fmt.Printf("\t%s\n", username)
	}

}

func keys(keyfile string) (string, string, oauth2.Token, error) {
	var keys struct {
		Key    string `json:"consumer_key"`
		Secret string `json:"consumer_secret"`
		BToken string `json:"bearer_token"`
	}

	f, err := os.Open("keys.json")
	if err != nil {
		panic(err)
	}

	defer f.Close()
	dec := json.NewDecoder(f)
	dec.Decode(&keys)

	var token oauth2.Token
	token.AccessToken = keys.BToken
	return keys.Key, keys.Secret, token, nil
}

// Authentication part
// this is to optain bearer token as we have already our bearer token we can also use it directly
// this is to obtain bearer token which can be ignored
func twitterClient(key, secret string) (*http.Client, error) {
	req, err := http.NewRequest("POST", "https://api.twitter.com/oauth2/token", strings.NewReader("grant_type=client_credentials"))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(key, secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")

	var client http.Client
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	//io.Copy(os.Stdout, res.Body)
	var token oauth2.Token
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&token)
	if err != nil {
		return nil, err
	}
	var conf oauth2.Config
	return conf.Client(context.Background(), &token), nil
}
func Retweeters(client *http.Client, tweetid string) ([]string, error) {

	url := fmt.Sprintf("https://api.twitter.com/1.1/statuses/retweets/%s.json", tweetid)
	// making request to twitter retweet API
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	//io.Copy(os.Stdout, res2.Body)
	var retweets []struct {
		User struct {
			ScreenName string `json:"screen_name"`
		} `json:"user"`
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&retweets)
	if err != nil {
		return nil, err
	}

	usernames := make([]string, 0, len(retweets))

	for _, retweet := range retweets {
		usernames = append(usernames, retweet.User.ScreenName)
	}

	return usernames, nil
}

func existing(usersFile string) []string {
	f, err := os.Open(usersFile)
	if err != nil {
		return []string{}
	}
	defer f.Close()
	r := csv.NewReader(f)
	lines, err := r.ReadAll()
	users := make([]string, 0, len(lines))
	for _, line := range lines {
		users = append(users, line[0])
	}
	return users
}

func merge(a, b []string) []string {
	uniq := make(map[string]struct{}, 0)
	for _, user := range a {
		uniq[user] = struct{}{}
	}
	for _, user := range b {
		uniq[user] = struct{}{}
	}
	ret := make([]string, 0, len(uniq))
	for user := range uniq {
		ret = append(ret, user)
	}
	return ret
}

func writeUsers(usersFile string, users []string) error {
	f, err := os.OpenFile(usersFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	for _, user := range users {
		if err := w.Write([]string{user}); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return nil
}

func pickWinners(users []string, numWinners int) []string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	perm := r.Perm(len(users))
	winners := perm[:numWinners]
	ret := make([]string, 0, numWinners)
	for _, idx := range winners {
		ret = append(ret, users[idx])
	}
	return ret
}
