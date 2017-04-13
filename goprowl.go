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

// ProwlDispatcher defines the methods for interacting with the Prowl API
type ProwlDispatcher interface {
	Push(n Notification) error                   // push the notification
	RequestToken() (*Tokens, error)              // request an access token
	RetrieveAPIKey(token string) (string, error) // retrieve an api key from prowlapp
}

// Notification is a Prowl notification
type Notification struct {
	Application string
	Description string
	Event       string
	Priority    int
	URL         string

	apikeys []string
}

// NewProwlClient creates a new client for interfacing with Prowl
func NewProwlClient(providerKey string) ProwlDispatcher {
	return &ProwlClient{
		ProviderKey: providerKey,
	}
}

// ProwlClient is used to interface with Prowl
type ProwlClient struct {
	ProviderKey string
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

// AddKey appends an API key to the notification list
func (n *Notification) AddKey(key string) error {

	if len(key) != 40 {
		return errors.New("Error, Apikey must be 40 characters long.")
	}

	n.apikeys = append(n.apikeys, key)
	return nil
}

// DelKey removes a key from the notification list
func (n *Notification) DelKey(key string) error {
	for i, value := range n.apikeys {
		if strings.EqualFold(key, value) {
			copy(n.apikeys[i:], n.apikeys[i+1:])
			n.apikeys[len(n.apikeys)-1] = ""
			n.apikeys = n.apikeys[:len(n.apikeys)-1]
			return nil
		}
	}
	return errors.New("Error, key not found")
}

// Push a notification to ProwlApp
func (c ProwlClient) Push(n Notification) error {

	keycsv := strings.Join(n.apikeys, ",")

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
			"token":       token,
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

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
