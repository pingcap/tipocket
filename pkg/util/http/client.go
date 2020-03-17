package http

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pingcap/errors"
)

// Client is a wrapper for http.Client.
type Client struct {
	*http.Client
}

// NewHTTPClient creates a HTTP Client.
func NewHTTPClient(c *http.Client) *Client {
	return &Client{Client: c}
}

// Get sends a HTTP GET request to the specified URL.
func (c *Client) Get(url string) ([]byte, error) {
	resp, err := c.httpRequest(url, http.MethodGet, "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("GET request \"%s\", got %v", url, resp.StatusCode))
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Read GET response failed")
	}
	return body, nil
}

// Post sends a HTTP POST request to the specified URL.
func (c *Client) Post(url string, bodyType string, body io.Reader) error {
	resp, err := c.httpRequest(url, http.MethodPost, bodyType, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		res, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Read POST response failed")
		}
		return errors.New(fmt.Sprintf("POST request \"%s\", got %v %s", url, resp.StatusCode, string(res)))
	}
	return nil
}

// Delete sends a HTTP DELETE request to the specified URL.
func (c *Client) Delete(url string) error {
	resp, err := c.httpRequest(url, http.MethodDelete, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		res, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Read DELETE response failed")
		}
		return errors.New(fmt.Sprintf("DELETE request \"%s\", got %v %s", url, resp.StatusCode, string(res)))
	}
	return nil
}

func (c *Client) httpRequest(url string, method string, bodyType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP request failed")
	}
	if bodyType != "" {
		req.Header.Set("Content-Type", bodyType)
	}
	return c.Do(req)
}
