package webrequest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/cixtor/readability"
	"github.com/gocolly/colly/v2"
)

const (
	baseURL          = "https://html.duckduckgo.com/html/?q=%s&no_redirect=1"
	duckDuckGoPrefix = "//duckduckgo.com/l/?uddg="
	maxSitesToVisit  = 7
	urlPattern       = `read:(https?://[^\s]+)` // Extract URL with "read:" prefix.
)

var (
	ErrWebsiteExceedsLimit    = errors.New("error_website_exceeds_limit")
	ErrWebsitesContentExceeds = errors.New("error_websites_content_exceeds")
	ErrFetchWebpage           = errors.New("errorFetch_webpage")
	ErrParseContent           = errors.New("error_parse_content")
	ErrVisitBaseURL           = errors.New("error_visit_base_url")
)

func prepareURL(u string) (string, error) {
	if strings.HasPrefix(u, duckDuckGoPrefix) {
		rawURL := strings.TrimPrefix(u, duckDuckGoPrefix)
		decodedURL, err := url.QueryUnescape(rawURL)
		if err != nil {
			return "", err
		}
		return strings.Split(decodedURL, "&rut=")[0], nil
	}
	return u, nil
}

func removeUselessWhitespaces(s string) string {
	re := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
	return re.ReplaceAllString(strings.TrimSpace(s), " ")
}

func fetchContent(link string) (string, error) {
	res, err := http.Get(link)
	if err != nil {
		return "", ErrFetchWebpage
	}
	defer res.Body.Close()

	r := readability.New()
	parsed, err := r.Parse(res.Body, link)
	if err != nil {
		return "", ErrParseContent
	}

	return removeUselessWhitespaces(parsed.TextContent), nil
}

func containsURL(content string) ([]string, bool) {
	re := regexp.MustCompile(urlPattern)
	matches := re.FindAllStringSubmatch(content, -1)

	var urls []string
	for _, match := range matches {
		if len(match) > 1 {
			urls = append(urls, match[1])
		}
	}

	if len(urls) > 0 {
		return urls, true
	}
	return nil, false
}

func WebRequest(query string) ([]string, error) {
	var res []string

	urlsFound, ok := containsURL(query)
	if ok {
		for _, urlFound := range urlsFound {
			content, err := fetchContent(urlFound)
			if err != nil {
				fmt.Println()
				continue
			}

			res = append(res, content+"\n==========\n")
		}

		if len(res) == 0 {
			res = append(res, "[No content found matching your query]")
		}
		return res, nil
	}

	c := colly.NewCollector()
	var sitesVisited int

	allDone := make(chan bool)

	c.OnHTML(".result", func(e *colly.HTMLElement) {
		if sitesVisited >= maxSitesToVisit {
			return
		}

		linkAttr := e.ChildAttr(".result__title .result__a", "href")
		link, err := prepareURL(linkAttr)
		if err != nil {
			fmt.Println("Error preparing URL:", err)
			return
		}

		content, err := fetchContent(link)
		if err != nil {
			fmt.Println("Error fetching content:", err)
			return
		}

		if len(content) > 1000 {
			content = content[:1000] + "..."
		}

		formattedContent := fmt.Sprintf(
			"Site %d (%s): %s\n==========\n",
			sitesVisited+1,
			link,
			content,
		)

		res = append(res, formattedContent)
		sitesVisited++
	})

	c.OnScraped(func(_ *colly.Response) {
		close(allDone)
	})

	err := c.Visit(fmt.Sprintf(baseURL, url.QueryEscape(query)))
	if err != nil {
		fmt.Println(err)
		return []string{}, ErrVisitBaseURL
	}

	<-allDone

	if len(res) == 0 {
		res = append(res, "[No content found matching your query]")
	}

	return res, nil
}
