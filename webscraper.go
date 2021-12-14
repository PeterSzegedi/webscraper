package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(5, 1)

type Link struct {
	SelfURL   string
	Visited   bool
	Errored   bool
	ChildURLs []string
}

// Look up a URL in links
func LinksContain(links []Link, url string) bool {
	for i := 0; i < len(links); i++ {
		if links[i].SelfURL == url {
			return true
		}
	}
	return false
}

// Function to test if the scraped URL is an absolute URL
func CheckIfURLAbsolute(element string) bool {
	if strings.Index(element, "://") < strings.Index(element, ".") && strings.Contains(element, "://") {
		return true
	}
	return false

}

// Function to test if the top level domain matches the absolute URL's domain
func CheckTLDMatch(tld string, currentURL string) bool {
	tldURLHost, err := neturl.Parse(tld)
	if err != nil {
		log.Warnf("Cannot parse URL, excluding from the list %s", tld)
		return false
	}

	currentURLHost, err := neturl.Parse(currentURL)
	if err != nil {
		log.Warnf("Cannot parse URL, excluding from the list %s", currentURL)
		return false
	}
	if tldURLHost.Host == currentURLHost.Host {
		return true
	}
	return false

}

// Trim anchors to avoid duplicated checks
func trimAnchor(href string) string {
	if strings.Contains(href, "#") {
		href = href[0:strings.Index(href, "#")]
	}
	return href
}

// Retrieve URLs from the response and return it
func GetUrls(tld string, url string, timeout int, wg *sync.WaitGroup, result chan Link) {

	defer wg.Done()
	var resultLink = Link{}
	resultLink.SelfURL = url
	resultLink.ChildURLs = []string{}

	// explicit client for setting the timeout
	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	// Explicit request for setting the useragent
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		log.Warnf("Error creating new request. %s ", err)
		resultLink.Visited = true
		resultLink.Errored = true

	}
	err = limiter.Wait(req.Context())
	if err != nil {
		log.Warnf("Error with limiting execution: %s", err)

	}
	req.Header.Set("User-Agent", "crawler_exercise")
	log.Infof("fetching url:%s", url)
	response, err := client.Do(req)

	if response == nil || err != nil {
		resultLink.Visited = true
		resultLink.Errored = true

	} else {
		defer response.Body.Close()
	}

	resultLink.Visited = true

	if response != nil {
		document, err := goquery.NewDocumentFromReader(response.Body)
		if err != nil {
			log.Warnf("Error loading document from response body. %s ", err)
			resultLink.Visited = true
			resultLink.Errored = true
		} else {
			if response.StatusCode != http.StatusOK {
				log.Debugf("HTTP statuscode not OK: %d", response.StatusCode)
				resultLink.Visited = true
				resultLink.Errored = true

			}
		}

		document.Find("a").Each(func(i int, element *goquery.Selection) {
			href, exists := element.Attr("href")
			if exists {
				if !CheckIfURLAbsolute(href) {

					href = strings.Trim(href, "/")
					tld = strings.TrimRight(tld, "/")

					var assembledURL = tld + "/" + href
					assembledURL = trimAnchor(assembledURL)

					if href != "" {
						resultLink.ChildURLs = append(resultLink.ChildURLs, assembledURL)
					}
				} else if CheckTLDMatch(tld, href) {
					var href = trimAnchor(href)
					resultLink.ChildURLs = append(resultLink.ChildURLs, href)
				}

			}

		})
	} else {
		resultLink.Visited = true
		resultLink.Errored = true
	}
	result <- resultLink
}

func main() {

	var url string
	var timeout int
	var maxRate int
	const (
		defaultURL              = ""
		urlUsage                = "The URL to scrape"
		defaultTimeout          = 5
		timeoutUsage            = "The timeout of the individual requests"
		defaultMaxRatePerSecond = 5
		maxRatePerSecondUsage   = "Request per second"
	)
	flag.StringVar(&url, "url", defaultURL, urlUsage)
	flag.IntVar(&timeout, "timeout", defaultTimeout, timeoutUsage)
	flag.IntVar(&maxRate, "maxrate", defaultMaxRatePerSecond, maxRatePerSecondUsage)
	flag.Parse()
	if url == "" {
		log.Fatalf("Please supply a URL to scrape")
	}

	parsedURL, err := neturl.Parse(url)
	if err != nil {
		log.Fatalf("Cannot parse TLD from main URL %s", url)

	}
	var tld string

	if parsedURL.User.String() == "" {
		tld = parsedURL.Scheme + "://" + parsedURL.Host

	} else {
		tld = parsedURL.Scheme + "://" + parsedURL.User.String() + "@" + parsedURL.Host
	}
	limiter = rate.NewLimiter(rate.Limit(maxRate), 1)

	var wg sync.WaitGroup

	// Get initial results for the TLD

	wg.Add(1)
	result := make(chan Link, 1)
	go GetUrls(tld, url, timeout, &wg, result)
	wg.Wait()

	resultLink := <-result
	close(result)

	var links = []Link{resultLink}
	log.Debugf("Link: %s, visited: %t, errored: %t, child URLs: %s\r\n", resultLink.SelfURL, resultLink.Visited, resultLink.Errored, resultLink.ChildURLs)

	// Loop until we retrieve all links from the childurls
	for {

		for i := 0; i < len(links); i++ {
			urlsToScrape := []string{}
			for j := 0; j < len(links[i].ChildURLs); j++ {

				if !LinksContain(links, links[i].ChildURLs[j]) {
					urlsToScrape = append(urlsToScrape, links[i].ChildURLs[j])
				}
			}
			// Make a buffered channel for the results
			result = make(chan Link, len(urlsToScrape))

			// Do the scraping for the collected childurls
			for k := 0; k < len(urlsToScrape); k++ {

				wg.Add(1)
				go GetUrls(tld, urlsToScrape[k], timeout, &wg, result)
			}
			wg.Wait()

			// Retrieve results and store it
			for i := 0; i < cap(result); i++ {
				resultLink := <-result
				links = append(links, resultLink)
				log.Debugf("Link: %s, visited: %t, errored: %t, child URLs: %s\r\n", resultLink.SelfURL, resultLink.Visited, resultLink.Errored, resultLink.ChildURLs)
			}
			close(result)
		}

		// Look if we need another loop
		flagUrls := []string{}
		for i := 0; i < len(links); i++ {
			for j := 0; j < len(links[i].ChildURLs); j++ {
				if !LinksContain(links, links[i].ChildURLs[j]) {
					flagUrls = append(flagUrls, links[i].ChildURLs[j])
				}
			}
		}
		if len(flagUrls) == 0 {
			break
		} else {
			for i := 0; i < len(flagUrls); i++ {
				log.Debugf("need another loop for:%s", flagUrls[i])

			}
		}

	}

	// Simply print the output in a prettified JSON
	outputJson, err := json.MarshalIndent(links, "", "  ")

	if err != nil {
		log.Warnf("Cannot encode to JSON %s", err)
	}

	fmt.Printf("{ \"links\":%s }\n", outputJson)

	// Log the output without formatting
	logJson, err := json.Marshal(links)

	if err != nil {
		log.Warnf("Cannot encode to JSON %s", err)
	}

	log.Debugf("{ \"links\":%s }", logJson)

}
