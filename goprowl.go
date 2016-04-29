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

// Notification is a Prowl notification
type Notification struct {
	Application string
	Description string
	Event       string
	Priority    int
	URL         string
}

// NewProwlClient creates a new client for interfacing with Prowl
func NewProwlClient(providerKey string) ProwlClient {
	return ProwlClient{
		ProviderKey: providerKey,
	}
}

// ProwlClient is used to interface with Prowl
type ProwlClient struct {
	ProviderKey string
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
func (c *ProwlClient) RegisterKey(key string) error {

	if len(key) != 40 {
		return errors.New("Error, Apikey must be 40 characters long.")
	}

	c.apikeys = append(c.apikeys, key)
	return nil
}

// DelKey removes a key from the notification list
func (c *ProwlClient) DelKey(key string) error {
	for i, value := range c.apikeys {
		if strings.EqualFold(key, value) {
			copy(c.apikeys[i:], c.apikeys[i+1:])
			c.apikeys[len(c.apikeys)-1] = ""
			c.apikeys = c.apikeys[:len(c.apikeys)-1]
			return nil
		}
	}
	return errors.New("Error, key not found")
}

// Push a notification to ProwlApp
func (c ProwlClient) Push(n Notification) (error) {

	keycsv := strings.Join(c.apikeys, ",")

	vals := url.Values{
		"apikey":      []string{keycsv},
		"application": []string{n.Application},
		"description": []string{n.Description},
		"event":       []string{n.Event},
		"priority":    []string{string(n.Priority)},
	}

	if n.URL != "" {
		vals["url"] = []string{n.URL}
	}

	if c.ProviderKey != "" {
		vals["providerkey"] = []string{c.ProviderKey}
	}

	r, err := http.PostForm(apiURL+"/add", vals)

	if err != nil {
		return err
	}

	defer r.Body.Close()

	if r.StatusCode != 200 {
		err = decodeError(r.Status, r.Body)
	}

	return err
}

// RequestToken retrieves a token from the ProwlApp API.
// Tokens are used to authenticate a user and generate their API key with prowlapp
func (c ProwlClient) RequestToken() (*Tokens, error) {
	body, err := makeHTTPRequestToURL(
		"GET",
		apiURL+"/retrieve/token",
		map[string]string{
			"providerkey": c.ProviderKey,
		},
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
func (c ProwlClient) RetrieveAPIKey(token string) (string, error) {
	body, err := makeHTTPRequestToURL(
		"GET",
		apiURL+"/retrieve/apikey",
		map[string]string{
			"providerkey": c.ProviderKey,
			"token": token,
		},
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

func makeHTTPRequestToURL(requestType, url string, params map[string]string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(requestType, url, body)

	if params != nil {
		q := req.URL.Query()

		for key, val := range params {
			q.Add(key, val)
		}

		req.URL.RawQuery = q.Encode()
	}

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
