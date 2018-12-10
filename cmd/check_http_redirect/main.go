package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/jasonhancock/go-nagios"
	"github.com/pkg/errors"
)

func main() {
	p := nagios.NewPlugin("check_http_redirect", flag.CommandLine)

	p.StringFlag("url", "", "The address of the Nomad server")
	p.StringFlag("expected", "", "A comma delimited list of acceptable URL targets to redirect to")
	p.StringFlag("codes", "", "A comma delimited list of acceptable http status codes")
	flag.Parse()

	codes, err := parseCodes(p.OptRequiredString("codes"))
	if err != nil {
		p.Fatal(errors.Wrap(err, "parsing codes"))
	}

	urls, err := parseURLs(p.OptRequiredString("expected"))
	if err != nil {
		p.Fatal(errors.Wrap(err, "parsing expected"))
	}

	source := p.OptRequiredString("url")

	code, location, err := checkRedirect(httpClient(), source, codes, urls)
	if err != nil {
		p.Exit(nagios.CRITICAL, err.Error())
	}

	p.Exit(nagios.OK, fmt.Sprintf("code=%d location=%q", code, location))
}

type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

var errLocationHeaderNotFound = errors.New("location header not found")

func httpClient() *http.Client {
	c := cleanhttp.DefaultClient()
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return c
}

func checkRedirect(client doer, u string, validCodes map[int]struct{}, validDestinations map[string]struct{}) (int, string, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return 0, "", errors.Wrap(err, "constructing request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", errors.Wrap(err, "executing request")
	}
	defer resp.Body.Close()

	if _, ok := validCodes[resp.StatusCode]; !ok {
		return 0, "", errors.Errorf("got unexpected status code %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return 0, "", errLocationHeaderNotFound
	}

	loc, err := url.Parse(location)
	if err != nil {
		return 0, "", errors.Wrap(err, "parsing location header")
	}

	if _, ok := validDestinations[loc.String()]; !ok {
		return 0, "", errors.Errorf("got invalid redirect target %q", loc.String())
	}

	return resp.StatusCode, loc.String(), nil
}

func parseCodes(codes string) (map[int]struct{}, error) {
	valid := make(map[int]struct{})
	pieces := strings.Split(codes, ",")
	for _, v := range pieces {
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return nil, errors.Wrapf(err, "parsing %q as an int", v)
		}
		valid[i] = struct{}{}
	}
	return valid, nil
}

func parseURLs(urls string) (map[string]struct{}, error) {
	valid := make(map[string]struct{})
	pieces := strings.Split(urls, ",")
	for _, v := range pieces {
		u, err := url.Parse(strings.TrimSpace(v))
		if err != nil {
			return nil, errors.Wrapf(err, "parsing %q as a url", v)
		}
		valid[u.String()] = struct{}{}
	}
	return valid, nil
}
