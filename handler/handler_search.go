package handler

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
)

var searchGithubRe = regexp.MustCompile(`https:\/\/github\.com\/(\w+)\/(\w+)`)

//uses im feeling lucky and grabs the "Location"
//header from the 302, which contains the IMDB ID
func searchGoogle(phrase string) (user, project string, err error) {
	phrase += " site:github.com"
	log.Printf("google search for '%s'", phrase)
	v := url.Values{}
	v.Set("btnI", "") //I'm feeling lucky
	v.Set("q", phrase)
	urlstr := "https://www.google.com/search?" + v.Encode()
	req, err := http.NewRequest("GET", urlstr, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "*/*")
	//I'm a browser... :)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.122 Safari/537.36")
	//roundtripper doesn't follow redirects
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", "", fmt.Errorf("request failed: %s", err)
	}
	resp.Body.Close()
	//assume redirection
	if resp.StatusCode != 302 {
		return "", "", fmt.Errorf("non-redirect response: %d", resp.StatusCode)
	}
	//extract Location header URL
	loc := resp.Header.Get("Location")
	m := searchGithubRe.FindStringSubmatch(loc)
	if len(m) == 0 {
		return "", "", fmt.Errorf("github url not found in redirect: %s", loc)
	}
	user = m[1]
	project = m[2]
	return user, project, nil
}
