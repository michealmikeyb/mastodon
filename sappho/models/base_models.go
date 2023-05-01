package models

import (
	"time"
)

type Candidate struct {
	StatusDomain		string `json:"status_domain"`
	StatusExternalId	string `json:"status_external_id"`
	StatusId 			string `json:"status_id"`
	AccountUrl 			string `json:"account_url"`
	AccountId 			string `json:"account_id"`
	AuthorUsername 		string `json:"author_username"`
	AuthorDomain 		string `json:"author_domain"`
}

type AggregatedCandidate struct {
	Aggregates				map[string]int `json:"aggregates"`
	Candidate				Candidate `json:"candidate"`
}

type RankedCandidate struct {
	Rank					float32 `json:"rank"`
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
	Domain         string	 `json:"domain"`
	Noindex        bool      `json:"noindex"`
	Emojis         []any     `json:"emojis"`
	Roles          []any     `json:"roles"`
	Fields         []struct {
		Name       string `json:"name"`
		Value      string `json:"value"`
		VerifiedAt any    `json:"verified_at"`
	} `json:"fields"`
}

type Status struct {
	ID                 string    `json:"id"`
	CreatedAt          time.Time `json:"created_at"`
	InReplyToID        any       `json:"in_reply_to_id"`
	InReplyToAccountID any       `json:"in_reply_to_account_id"`
	Sensitive          bool      `json:"sensitive"`
	SpoilerText        string    `json:"spoiler_text"`
	Visibility         string    `json:"visibility"`
	Language           string    `json:"language"`
	URI                string    `json:"uri"`
	URL                string    `json:"url"`
	RepliesCount       int       `json:"replies_count"`
	ReblogsCount       int       `json:"reblogs_count"`
	FavouritesCount    int       `json:"favourites_count"`
	EditedAt           any       `json:"edited_at"`
	Content            string    `json:"content"`
	Reblog             any       `json:"reblog"`
	Account            Account `json:"account"`
	MediaAttachments []any `json:"media_attachments"`
	Mentions         []any `json:"mentions"`
	Tags             []Tag `json:"tags"`
	Emojis           []any `json:"emojis"`
	Card             any   `json:"card"`
	Poll             any   `json:"poll"`
}

type Tag struct {
	Name	string `json:"name"`
	Url		string `json:"url"`
}