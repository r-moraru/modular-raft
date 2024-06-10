package clients

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type Client struct {
	url        string
	httpClient *http.Client
}

func (l *Client) get(url string, v any) error {
	resp, err := l.httpClient.Get(url)
	if err != nil {
		slog.Error("Failed to get response from " + url)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		slog.Error("Response status: " + resp.Status + " from " + url)
		return errors.New("Response status: " + resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		slog.Error("Failed to decode response from " + url)
		return err
	}
	return nil
}

func (l *Client) post(url string, payload any) error {
	marshalledPayload, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshall payload.")
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(marshalledPayload))
	if err != nil {
		slog.Error("Failed to create POST request for " + url)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		slog.Error("Failed to send POST request to " + url)
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		slog.Error("Response Status: " + resp.Status + " from " + url)
		return errors.New("Resopnse status: " + resp.Status)
	}

	return nil
}

func (l *Client) postWithResp(url string, payload any, respVal any) error {
	marshalledPayload, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshall payload.")
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(marshalledPayload))
	if err != nil {
		slog.Error("Failed to create POST request for " + url)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		slog.Error("Failed to send POST request to " + url)
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		slog.Error("Response Status: " + resp.Status + " from " + url)
		return errors.New("Resopnse status: " + resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(respVal)
	if err != nil {
		slog.Error("Failed to decode response from " + url)
		return err
	}
	return nil
}
