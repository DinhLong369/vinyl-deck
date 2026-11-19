// Package youtube parses YouTube links and looks up metadata via oEmbed (no API key needed).
package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	idRe          = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	ErrInvalidURL = errors.New("not a valid YouTube link")
)

// ExtractID accepts watch, youtu.be, embed, shorts, live and music links, or a bare 11-char id.
func ExtractID(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if idRe.MatchString(raw) {
		return raw, nil
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", ErrInvalidURL
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	host = strings.TrimPrefix(host, "m.")
	var id string
	switch host {
	case "youtu.be":
		id = strings.Split(strings.Trim(u.Path, "/"), "/")[0]
	case "youtube.com", "music.youtube.com", "youtube-nocookie.com":
		if v := u.Query().Get("v"); v != "" {
			id = v
		} else {
			parts := strings.Split(strings.Trim(u.Path, "/"), "/")
			if len(parts) >= 2 && (parts[0] == "embed" || parts[0] == "shorts" || parts[0] == "live" || parts[0] == "v") {
				id = parts[1]
			}
		}
	}
	if !idRe.MatchString(id) {
		return "", ErrInvalidURL
	}
	return id, nil
}

type Meta struct {
	Title     string
	Author    string
	Thumbnail string
}

type Client struct{ http *http.Client }

func NewClient() *Client { return &Client{http: &http.Client{Timeout: 8 * time.Second}} }

func ThumbnailURL(id string) string {
	return fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", id)
}

// Lookup fetches title/author. A 401/404 from oEmbed means the video is private,
// removed or not embeddable, which would not play in the iframe player either.
func (c *Client) Lookup(ctx context.Context, id string) (Meta, error) {
	q := url.Values{"url": {"https://www.youtube.com/watch?v=" + id}, "format": {"json"}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.youtube.com/oembed?"+q.Encode(), nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return Meta{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Meta{}, fmt.Errorf("video unavailable or not embeddable (oembed %d)", resp.StatusCode)
	}
	var body struct {
		Title      string `json:"title"`
		AuthorName string `json:"author_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Meta{}, err
	}
	if strings.TrimSpace(body.Title) == "" {
		return Meta{}, errors.New("video unavailable (no metadata from YouTube)")
	}
	return Meta{Title: body.Title, Author: body.AuthorName, Thumbnail: ThumbnailURL(id)}, nil
}
