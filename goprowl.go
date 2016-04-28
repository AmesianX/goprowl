//
// Copyright (c) 2011, Yanko D Sanchez Bolanos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//     * Redistributions of source code must retain the above copyright
//       notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above copyright
//       notice, this list of conditions and the following disclaimer in the
//       documentation and/or other materials provided with the distribution.
//     * Neither the name of the author nor the
//       names of its contributors may be used to endorse or promote products
//       derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL <COPYRIGHT HOLDER> BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//

package goprowl

import (
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const (
	apiURL = "https://api.prowlapp.com/publicapi"
)

type Notification struct {
	Application string
	Description string
	Event       string
	Priority    string
	Providerkey string
	URL         string
}

type Goprowl struct {
	apikeys []string
}

type errorResponse struct {
	Error struct {
		Code    int    `xml:"code,attr"`
		Message string `xml:",chardata"`
	} `xml:"error"`
}

// Tokens contains the information with which to request an API key.
type Tokens struct {
	Token string `xml:"token,attr"`
	URL   string `xml:"url,attr"`
}

type tokenResponse struct {
	XMLName  xml.Name `xml:"prowl"`
	Retrieve Tokens   `xml:"retrieve"`
}

type apiKeyResponse struct {
	XMLName      xml.Name `xml:"prowl"`
	APIKeyValues struct {
		APIKey string `xml:"apikey,attr"`
	} `xml:"retrieve"`
}

// RegisterKey appends an API key to the notification list
func (gp *Goprowl) RegisterKey(key string) error {

	if len(key) != 40 {
		return errors.New("Error, Apikey must be 40 characters long.")
	}

	gp.apikeys = append(gp.apikeys, key)
	return nil
}

// DelKey removes a key from the notification list
func (gp *Goprowl) DelKey(key string) error {
	for i, value := range gp.apikeys {
		if strings.EqualFold(key, value) {
			copy(gp.apikeys[i:], gp.apikeys[i+1:])
			gp.apikeys[len(gp.apikeys)-1] = ""
			gp.apikeys = gp.apikeys[:len(gp.apikeys)-1]
			return nil
		}
	}
	return errors.New("Error, key not found")
}

// Push a notification to ProwlApp
func (gp *Goprowl) Push(n *Notification) (error) {

	keycsv := strings.Join(gp.apikeys, ",")

	vals := url.Values{
		"apikey":      []string{keycsv},
		"application": []string{n.Application},
		"description": []string{n.Description},
		"event":       []string{n.Event},
		"priority":    []string{n.Priority},
	}

	if n.URL != "" {
		vals["url"] = []string{n.URL}
	}

	if n.Providerkey != "" {
		vals["providerkey"] = []string{n.Providerkey}
	}

	r, err := http.PostForm(apiURL+"/add", vals)

	if err != nil {
		return err
	} else {
		defer r.Body.Close()
		if r.StatusCode != 200 {
			err = decodeError(r.Status, r.Body)
		}
	}
	
	return err
}

// RequestToken retrieves a token from the ProwlApp API.
// Tokens are used to authenticate a user and generate their API key with prowlapp
func RequestToken(providerKey string) (*Tokens, error) {
	body, err := makeHTTPRequestToURL(
		"GET",
		apiURL+"/retrieve/token?providerkey="+providerKey,
		nil,
	)

	token := tokenResponse{}

	err = xml.Unmarshal(body, &token)

	if err != nil {
		return nil, err
	}

	return &token.Retrieve, err
}

// RetrieveAPIKey returns an API key given a token from RequestToken.
// API keys can be added to lists etc
func RetrieveAPIKey(providerKey, token string) (string, error) {
	body, err := makeHTTPRequestToURL(
		"GET",
		apiURL+"/retrieve/apikey?providerkey="+providerKey+"&token="+token,
		nil,
	)

	apiKeyResponse := apiKeyResponse{}

	err = xml.Unmarshal(body, &apiKeyResponse)

	if err != nil {
		return "", err
	}

	return apiKeyResponse.APIKeyValues.APIKey, err
}

func decodeError(def string, r io.Reader) (err error) {
	xres := errorResponse{}
	if xml.NewDecoder(r).Decode(&xres) != nil {
		err = errors.New(def)
	} else {
		err = errors.New(xres.Error.Message)
	}
	return
}

func makeHTTPRequestToURL(requestType, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(requestType, url, body)

	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	resp, err := client.Do(req)

	defer resp.Body.Close()

	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(resp.Body)
}
