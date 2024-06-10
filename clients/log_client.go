package clients

import (
	"net/http"
	"time"

	"github.com/r-moraru/modular-raft/proto/entries"
)

const (
	LastLogPath = "/last_log"
	LengthPath  = "/length"
	EntryPath   = "/entry"
	TermPath    = "/term"
	InsertPath  = "/insert"
	AppendPath  = "/append"
)

type GetLastIndexResponse struct {
	LastIndex uint64 `json:"last_index"`
}

type GetLengthResponse struct {
	Length uint64 `json:"length"`
}

type GetTermRequest struct {
	Index uint64 `json:"index"`
}

type GetTermResponse struct {
	Term uint64 `json:"term"`
}

type GetEntryRequest struct {
	Index uint64 `json:"index"`
}

type LogClient Client

func NewLogClient(url string, timeout uint64) *LogClient {
	return &LogClient{
		url: url,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Millisecond,
		},
	}
}

func (l *LogClient) GetLastIndex() uint64 {
	getLastIndexResponse := new(GetLastIndexResponse)
	err := (*Client)(l).get(l.url+LastLogPath, getLastIndexResponse)
	if err != nil {
		return 0
	}
	return getLastIndexResponse.LastIndex
}

func (l *LogClient) GetLength() uint64 {
	getLengthResponse := new(GetLengthResponse)
	err := (*Client)(l).get(l.url+LengthPath, getLengthResponse)
	if err != nil {
		return 0
	}
	return getLengthResponse.Length
}

func (l *LogClient) GetEntry(index uint64) (*entries.LogEntry, error) {
	getEntryRequest := &GetEntryRequest{Index: index}
	getEntryResponse := new(entries.LogEntry)
	err := (*Client)(l).postWithResp(
		l.url+LengthPath,
		getEntryRequest,
		getEntryResponse,
	)
	if err != nil {
		return nil, err
	}
	return getEntryResponse, nil
}

func (l *LogClient) GetTermAtIndex(index uint64) (uint64, error) {
	getTermRequest := &GetTermRequest{Index: index}
	getTermResponse := new(GetTermResponse)
	err := (*Client)(l).postWithResp(
		l.url+TermPath,
		getTermRequest,
		getTermResponse,
	)
	if err != nil {
		return 0, err
	}
	return getTermResponse.Term, nil
}

func (l *LogClient) InsertLogEntry(entry *entries.LogEntry) error {
	return (*Client)(l).post(l.url+InsertPath, entry)
}

func (l *LogClient) AppendEntry(term uint64, clientID string, serializationID uint64, entry string) error {
	logEntry := &entries.LogEntry{
		Term:            term,
		ClientID:        clientID,
		SerializationID: serializationID,
		Entry:           entry,
	}
	return (*Client)(l).post(l.url+AppendPath, logEntry)
}
