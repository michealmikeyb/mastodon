package models

import (
	"time"
)

type Candidate struct {
	StatusUrl 		string `json:"status_url"`
	StatusId 		string `json:"status_id"`
	AccountUrl 	string `json:"account_url"`
	AccountId 		string `json:"account_id"`
	AuthorUrl 		string `json:"author_url"`
	AuthorId 		string `json:"author_id"`
}

type AggregatedCandidate struct {
	AuthorFollowerCount 	int `json:"author_follower_count"`
	Candidate				Candidate `json:"candidate"`
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