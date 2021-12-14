package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestCheckIfURLRelative(t *testing.T) {
	absolute := CheckIfURLAbsolute("/results")
	if absolute {
		t.Errorf("URLs relative check was incorrect, got: %t, want: %t for /results", absolute, true)
	}
}

func TestCheckIfURLAbsolute(t *testing.T) {
	absolute := CheckIfURLAbsolute("https://example.com/static/images/icons/extra-features-icon.svg")
	if !absolute {
		t.Errorf("URLs relative check was incorrect, got: %t, want: %t for https://example.com/static/images/icons/extra-features-icon.svg", absolute, false)
	}
}
func TestCheckTLDMatch(t *testing.T) {
	matchingTLD := CheckTLDMatch("https://community.example.com", "https://example.com/static/images/icons/extra-features-icon.svg")
	if matchingTLD {
		t.Errorf("Filtering TLD test was incorrect, got: %t, want: %t for https://community.example.com and https://example.com/static/images/icons/extra-features-icon.svg", matchingTLD, false)
	}
}

func TestLinksContain(t *testing.T) {
	var resultLink = Link{}
	resultLink.SelfURL = "https://example.com"
	resultLink.ChildURLs = []string{"https://community.example.com"}

	links := []Link{resultLink}

	linksDoesContain := LinksContain(links, "https://community.example.com")
	if linksDoesContain {
		t.Errorf("Looking up a link in Links was giving the wrong result, got: %t, want: %t for https://community.example.com", linksDoesContain, false)
	}
}

func TestGetURL(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`Dummy content `))
	}))

	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	result := make(chan Link, 1)
	go GetUrls(server.URL, server.URL, 1, &wg, result)
	wg.Wait()

	resultLink := <-result
	close(result)

	if resultLink.Errored {
		t.Errorf("No result from webserver")
	}
}
