package clients

import (
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

type getLastIndexResponse struct {
	LastIndex uint64 `json:"last_index"`
}

type getLengthResponse struct {
	Length uint64 `json:"length"`
}

type getTermResponse struct {
	Term uint64 `json:"term"`
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
	getLastIndexResponse := new(getLastIndexResponse)
	err := (*Client)(l).get(l.url+lastLogPath, getLastIndexResponse)
	if err != nil {
		return 0
	}
	return getLastIndexResponse.LastIndex
}

func (l *LogClient) GetLength() uint64 {
	getLengthResponse := new(getLengthResponse)
	err := (*Client)(l).get(l.url+lengthPath, getLengthResponse)
	if err != nil {
		return 0
	}
	return getLengthResponse.Length
}

func (l *LogClient) GetEntry(index uint64) (*entries.LogEntry, error) {
	getEntryResponse := new(entries.LogEntry)
	err := (*Client)(l).get(
		l.url+lengthPath+strconv.Itoa(int(index)),
		getEntryResponse,
	)
	if err != nil {
		return nil, err
	}
	return getEntryResponse, nil
}

func (l *LogClient) GetTermAtIndex(index uint64) (uint64, error) {
	getTermResponse := new(getTermResponse)
	err := (*Client)(l).get(
		l.url+termPath+strconv.Itoa(int(index)),
		getTermResponse,
	)
	if err != nil {
		return 0, err
	}
	return getTermResponse.Term, nil
}

func (l *LogClient) InsertLogEntry(entry *entries.LogEntry) error {
	return (*Client)(l).post(l.url+insertPath, entry)
}

func (l *LogClient) AppendEntry(term uint64, clientID string, serializationID uint64, entry string) error {
	logEntry := &entries.LogEntry{
		Term:            term,
		ClientID:        clientID,
		SerializationID: serializationID,
		Entry:           entry,
	}
	return (*Client)(l).post(l.url+appendPath, logEntry)
}
