package log

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/r-moraru/modular-raft/proto/entries"
)

const (
	lastLogPath = "last_log/"
	lengthPath  = "length/"
	entryPath   = "entry/"
	termPath    = "term/"
	insertPath  = "insert/"
	appendPath  = "append/"
)

type GetLastIndexResponse struct {
	LastIndex uint64 `json:"last_index"`
}

type GetLengthResponse struct {
	Length uint64 `json:"length"`
}

type GetTermResponse struct {
	Term uint64 `json:"term"`
}

type LogClient struct {
	logUrl string
	client *http.Client
}

func (l *LogClient) NewLogClient(url string, timeout uint64) *LogClient {
	return &LogClient{
		logUrl: url,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Millisecond,
		},
	}
}

func (l *LogClient) getAndProcessRequest(url string, v any) error {
	resp, err := l.client.Get(url)
	if err != nil {
		log.Fatalf("Failed to get response from " + url)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Fatalf("Response status: " + resp.Status + " from " + url)
		return errors.New("Response status: " + resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		log.Fatalf("Failed to decode response from " + url)
		return err
	}
	return nil
}

func (l *LogClient) post(url string, payload any) error {
	marshalledPayload, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshall payload.")
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(marshalledPayload))
	if err != nil {
		log.Fatalf("Failed to create POST request for " + url)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		log.Fatalf("Failed to send POST request to " + url)
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Fatalf("Response Status: " + resp.Status + " from " + url)
		return errors.New("Resopnse status: " + resp.Status)
	}

	return nil
}

func (l *LogClient) GetLastIndex() uint64 {
	getLastIndexResponse := new(GetLastIndexResponse)
	err := l.getAndProcessRequest(l.logUrl+lastLogPath, getLastIndexResponse)
	if err != nil {
		return 0
	}
	return getLastIndexResponse.LastIndex
}

func (l *LogClient) GetLength() uint64 {
	getLengthResponse := new(GetLengthResponse)
	err := l.getAndProcessRequest(l.logUrl+lengthPath, getLengthResponse)
	if err != nil {
		return 0
	}
	return getLengthResponse.Length
}

func (l *LogClient) GetEntry(index uint64) (*entries.LogEntry, error) {
	getEntryResponse := new(entries.LogEntry)
	err := l.getAndProcessRequest(
		l.logUrl+lengthPath+strconv.Itoa(int(index)),
		getEntryResponse,
	)
	if err != nil {
		return nil, err
	}
	return getEntryResponse, nil
}

func (l *LogClient) GetTermAtIndex(index uint64) (uint64, error) {
	getTermResponse := new(GetTermResponse)
	err := l.getAndProcessRequest(
		l.logUrl+termPath+strconv.Itoa(int(index)),
		getTermResponse,
	)
	if err != nil {
		return 0, err
	}
	return getTermResponse.Term, nil
}

func (l *LogClient) InsertLogEntry(entry *entries.LogEntry) error {
	return l.post(l.logUrl+insertPath, entry)
}

func (l *LogClient) AppendEntry(term uint64, clientID string, serializationID uint64, entry string) error {
	logEntry := &entries.LogEntry{
		Term:            term,
		ClientID:        clientID,
		SerializationID: int64(serializationID),
		Entry:           entry,
	}
	return l.post(l.logUrl+appendPath, logEntry)
}
