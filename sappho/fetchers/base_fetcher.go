package fetcher

import (
	"log"
	"net/http"
	"encoding/json"
	"time"
 )


type DataStream interface {
	Init(status_url string, account_url string)
	GetChannel() chan interface{}
}

type Account struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Acct           string    `json:"acct"`
	DisplayName    string    `json:"display_name"`
	Locked         bool      `json:"locked"`
	Bot            bool      `json:"bot"`
	Discoverable   bool      `json:"discoverable"`
	Group          bool      `json:"group"`
	CreatedAt      time.Time `json:"created_at"`
	Note           string    `json:"note"`
	URL            string    `json:"url"`
	Avatar         string    `json:"avatar"`
	AvatarStatic   string    `json:"avatar_static"`
	Header         string    `json:"header"`
	HeaderStatic   string    `json:"header_static"`
	FollowersCount int       `json:"followers_count"`
	FollowingCount int       `json:"following_count"`
	StatusesCount  int       `json:"statuses_count"`
	LastStatusAt   string    `json:"last_status_at"`
	Noindex        bool      `json:"noindex"`
	Emojis         []any     `json:"emojis"`
	Roles          []any     `json:"roles"`
	Fields         []struct {
		Name       string `json:"name"`
		Value      string `json:"value"`
		VerifiedAt any    `json:"verified_at"`
	} `json:"fields"`
}

type AccountStream struct {
	account_url string
	channel chan Account
}

func (as AccountStream) Init(status_url string, account_url string) error {
	as.account_url = account_url
	resp, err := http.Get(account_url)
	if err != nil {
		return err
	}

	var account Account
	err = json.NewDecoder(resp.Body).Decode(&account)
	if err != nil {
		return err
	}
	log.Printf("%s", account.Username)
	return nil
}

type DataStreamMap map[string] DataStream

type Fetcher func(x DataStreamMap)

