package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/cheekybits/is"
	"github.com/pkg/errors"
)

type fakeDoer struct {
	code     int
	location string
}

func (d *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	r := &http.Response{
		StatusCode: d.code,
		Header:     make(http.Header),
	}

	r.Header.Set("Location", d.location)
	r.Body = ioutil.NopCloser(bytes.NewReader([]byte("hello world")))
	return r, nil
}

func TestCheckRedirect(t *testing.T) {
	var tests = []struct {
		description string
		code        int
		location    string
		validCodes  string
		validUrls   string
		expectedErr error
	}{
		{"normal", 301, "https://www.example.com", "301", "https://www.example.com", nil},
		{"invalid code", 308, "https://www.example.com", "301", "https://www.example.com", errors.New("got unexpected status code 308")},
		{"invalid location", 301, "https://www.example.com", "301", "https://foo.example.com", errors.New(`got invalid redirect target "https://www.example.com"`)},
		{"empty location header", 301, "", "301", "https://foo.example.com", errLocationHeaderNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			is := is.New(t)
			d := &fakeDoer{code: tt.code, location: tt.location}
			codes, err := parseCodes(tt.validCodes)
			is.NoErr(err)
			urls, err := parseURLs(tt.validUrls)
			is.NoErr(err)

			code, loc, err := checkRedirect(d, "", codes, urls)
			if tt.expectedErr != nil {
				is.Err(err)
				is.Equal(err.Error(), tt.expectedErr.Error())
				return
			}
			is.NoErr(err)
			is.Equal(code, tt.code)
			is.Equal(loc, tt.location)
		})
	}

}
