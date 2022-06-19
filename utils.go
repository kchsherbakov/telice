package main

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

func reformatYouTubeUrl(origin string) string {
	if strings.Contains(origin, "youtu.be") {
		u, _ := url.Parse(origin)
		videoId := path.Base(u.Path)

		return fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoId)
	}

	return origin
}
