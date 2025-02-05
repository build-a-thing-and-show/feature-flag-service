package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/go-kit/kit/endpoint"
	kitHttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// FeatureFlagService defines the interface for managing feature flags
type FeatureFlagService interface {
	GetFeatureFlag(ctx context.Context, key string) (bool, error)
	SetFeatureFlag(ctx context.Context, key string, value bool) error
}

// featureFlagService is a concrete implementation of FeatureFlagService
type featureFlagService struct {
	flags map[string]bool
	mu    sync.RWMutex
}

func (s *featureFlagService) GetFeatureFlag(ctx context.Context, key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.flags[key]
	if !exists {
		return false, nil
	}
	return val, nil
}

func (s *featureFlagService) SetFeatureFlag(ctx context.Context, key string, value bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flags[key] = value
	return nil
}

// request and response types
type getFeatureFlagRequest struct {
	Key string `json:"key"`
}

type getFeatureFlagResponse struct {
	Value bool `json:"value"`
}

type setFeatureFlagRequest struct {
	Key   string `json:"key"`
	Value bool   `json:"value"`
}

type setFeatureFlagResponse struct {
	Success bool `json:"success"`
}

// Endpoints
func makeGetFeatureFlagEndpoint(svc FeatureFlagService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getFeatureFlagRequest)
		val, _ := svc.GetFeatureFlag(ctx, req.Key)
		return getFeatureFlagResponse{Value: val}, nil
	}
}

func makeSetFeatureFlagEndpoint(svc FeatureFlagService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(setFeatureFlagRequest)
		svc.SetFeatureFlag(ctx, req.Key, req.Value)
		return setFeatureFlagResponse{Success: true}, nil
	}
}

func main() {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	svc := &featureFlagService{flags: make(map[string]bool)}
	getFeatureFlagHandler := kitHttp.NewServer(
		makeGetFeatureFlagEndpoint(svc),
		decodeGetFeatureFlagRequest,
		encodeResponse,
	)

	setFeatureFlagHandler := kitHttp.NewServer(
		makeSetFeatureFlagEndpoint(svc),
		decodeSetFeatureFlagRequest,
		encodeResponse,
	)

	http.Handle("/get", getFeatureFlagHandler)
	http.Handle("/set", setFeatureFlagHandler)

	level.Info(logger).Log("msg", "Starting server on port 10001")
	http.ListenAndServe(":10001", nil)
}

func decodeGetFeatureFlagRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getFeatureFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func decodeSetFeatureFlagRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request setFeatureFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}
