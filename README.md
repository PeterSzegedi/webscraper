#### Webscraper Exercise

This solution is created to collect the links from a website and present it in a simple way. The program is using goroutines to collect the URLs asynchronously. The amount of concurrent routines are rate limited. The output is printed as a prettified JSON 

### Quick start for testing
- Unzip the files 
- go get the required modules (see go.mod)
- Run the webscraper.go. Example:

`go run webscraper.go --url "http://www.deadlinkcity.com/" --maxrate 5 --timeout 1`

Disclaimer: http://www.deadlinkcity.com/ is not owned by me, so please try to keep the testing at the minimum/non intrusive.

### Options

```
  --url string
        The URL of the site to scrape
  --maxrate int
        Maximum number of goroutines to use per second for scraping the URLs (default 5)
  --timeout int
        The timeout for getting the URLs (default 5)
```

### Testing
Run `go test` to run the tests. Ideally this should be part of a pipeline

### What is missing
- The webscraper does not care with the robots file
- It cannot use mTLS authentication to access restricted sites or basic auth
- It does not check the size of the pages before reading them
- It removes the anchors as a hardcoded behaviour
- It is using a hardcoded User Agent
- It will try to handle all protocols found in the links as HTTP related protocols
- Fail-safes to stop scraping after an X amount links (although this might be just overly cautious, depending on the usage)

### License
This project is licensed under the MIT License - see the LICENSE.md file for details
