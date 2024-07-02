package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/r-moraru/modular-raft/proto/entries"
	"github.com/r-moraru/modular-raft/state_machine"
)

const (
	ApplyPath       = "/apply"
	LastAppliedPath = "/last_applied"
	GetResultPath   = "/get_result"
)

type GetLastAppliedResponse struct {
	LastApplied uint64 `json:"last_applied"`
}

type GetApplyResultResponse struct {
	ApplyResult string `json:"apply_result"`
}

type GetResultRequest struct {
	ClientID        string `json:"client_id"`
	SerializationID uint64 `json:"serialization_id"`
}

type StateMachineClient Client

func NewStateMachineClient(url string, timeout uint64) *StateMachineClient {
	return &StateMachineClient{
		url: url,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Millisecond,
		},
	}
}

func (s *StateMachineClient) Apply(entry *entries.LogEntry) error {
	return (*Client)(s).post(s.url+ApplyPath, entry)
}

func (s *StateMachineClient) GetLastApplied() uint64 {
	getLastAppliedResponse := new(GetLastAppliedResponse)
	err := (*Client)(s).get(s.url+LastAppliedPath, getLastAppliedResponse)
	if err != nil {
		return 0
	}
	return getLastAppliedResponse.LastApplied
}

func (s *StateMachineClient) WaitForResult(ctx context.Context, clientID string, serializationID uint64) chan state_machine.ApplyResult {
	resultChan := make(chan state_machine.ApplyResult, 1)

	payload := &GetResultRequest{
		ClientID:        clientID,
		SerializationID: serializationID,
	}
	marshalledPayload, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal payload.")
		resultChan <- state_machine.ApplyResult{
			Error: err,
		}
		return resultChan
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.url+GetResultPath, bytes.NewReader(marshalledPayload))
	if err != nil {
		slog.Error("Failed to create POST request.")
		resultChan <- state_machine.ApplyResult{
			Error: err,
		}
		return resultChan
	}
	req.Header.Set("Content-Type", "application/json")

	go func() {
		bareHttpClient := &http.Client{}
		resp, err := bareHttpClient.Do(req)
		if err != nil {
			slog.Error("Get result http request failed.")
			resultChan <- state_machine.ApplyResult{
				Error: err,
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			slog.Error("Response status: " + resp.Status + " from state machine.")
			resultChan <- state_machine.ApplyResult{
				Error: errors.New("received error from state machine"),
			}
			return
		}

		getApplyResultResponse := new(GetApplyResultResponse)
		err = json.NewDecoder(resp.Body).Decode(getApplyResultResponse)
		if err != nil {
			slog.Error("Error decoding response from state machine.")
			resultChan <- state_machine.ApplyResult{
				Error: err,
			}
			return
		}

		resultChan <- state_machine.ApplyResult{
			Result: getApplyResultResponse.ApplyResult,
			Error:  nil,
		}
	}()

	return resultChan
}
