/*
Package relay implments a simple client to report errors using the Mailgun mailing service
*/
package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"time"
)

var ErrBadConfig = errors.New("the config does not contain the necessary information")
var ErrBadRequest = errors.New("the request generated by relay is invalid")
var ErrMailgunDown = errors.New("something seems to be wrong with Mailgun servers")
var ErrUnknown = errors.New("something undefined happened")
var ErrNoConfig = errors.New("no Config object provided and config.json does not exist")

// Relay is a client to send error messages with
type Relay struct {
	c      *http.Client
	domain string
	to     string
	from   string
	key    string
}

// Config contains the information used to initialize a Relay
type Config struct {
	Domain string `json:"domain"`
	To     string `json:"to"`
	From   string `json:"from"`
	Key    string `json:"api_key"`
}

// New is used to generate a new Relay. If called with argument nil, it
// reads from config.json
func New(c *Config) (*Relay, error) {
	// make a new relay
	r := &Relay{
		c: &http.Client{},
	}

	// if they gave us a config, use it
	if c != nil {
		r.to = c.To
		r.from = c.From
		r.key = c.Key
		r.domain = c.Domain
	} else { // otherwise read config.json
		infile, err := os.Open("config.json")
		defer infile.Close()
		if err != nil {
			return nil, ErrNoConfig
		}

		dec := json.NewDecoder(infile)

		config := new(Config)
		err = dec.Decode(config)
		if err != nil {
			return nil, ErrBadConfig
		}

		r.to = config.To
		r.from = config.From
		r.key = config.Key
		r.domain = config.Domain
	}

	// ensure all necessary fields are set
	if r.to == "" || r.from == "" || r.key == "" || r.domain == "" {
		return nil, ErrBadConfig
	} else {
		return r, nil
	}
}

// Send sends the error 'err' with a timestamp the supplied subject
func (r *Relay) Send(subject string, err error) error {
	// Collect the information we want to send
	text := time.Now().Format(time.RFC1123) + ":\n" + err.Error()
	val := make(url.Values)
	val.Add("from", r.from)
	val.Add("to", r.to)
	val.Add("subject", subject)
	val.Add("text", text)

	// Set up the request
	req, err := http.NewRequest("POST",
		"https://api.mailgun.net/v2/"+r.domain+"/messages",
		bytes.NewReader([]byte(val.Encode())))

	if err != nil {
		return ErrBadRequest
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.SetBasicAuth("api", r.key)

	// Do the request
	res, err := r.c.Do(req)

	if err != nil {
		return err
	}

	// Handle the errors
	if res.StatusCode == 200 {
		return nil
	}
	if res.StatusCode >= 400 && res.StatusCode < 500 {
		return ErrBadRequest
	}
	if res.StatusCode >= 500 {
		return ErrMailgunDown
	}

	return ErrUnknown
}
