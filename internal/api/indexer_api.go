package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

const (
	APIEndpointBaseRedacted = "https://redacted.ch/ajax.php"
	APIEndpointBaseOrpheus  = "https://orpheus.network/ajax.php"
)

func makeRequest(endpoint, apiKey string, limiter *rate.Limiter, indexer string, target interface{}) error {
	if !limiter.Allow() {
		log.Warn().Msgf("%s: Too many requests", indexer)
		return fmt.Errorf("too many requests")
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Error().Msgf("fetchAPI error: %v", err)
	}
	req.Header.Set("Authorization", apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Msgf("fetchAPI error: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Msgf("fetchAPI error: %v", err)
	}

	if err := json.Unmarshal(respBody, target); err != nil {
		log.Error().Msgf("fetchAPI error: %v", err)
	}

	responseData := target.(*ResponseData)
	if responseData.Status != "success" {
		log.Warn().Msgf("API error from %s: %s", indexer, responseData.Error)
		return fmt.Errorf("API error from %s: %s", indexer, responseData.Error)
	}

	return nil
}

func initiateAPIRequest(id int, action string, apiKey, apiBase, indexer string) (*ResponseData, error) {
	limiter := getLimiter(indexer)
	if limiter == nil {
		return nil, fmt.Errorf("could not get rate limiter for indexer: %s", indexer)
	}

	endpoint := fmt.Sprintf("%s?action=%s&id=%d", apiBase, action, id)
	responseData := &ResponseData{}
	if err := makeRequest(endpoint, apiKey, limiter, indexer, responseData); err != nil {
		return nil, err
	}

	// Log the release information
	if action == "torrent" && responseData.Response.Torrent != nil {
		releaseName := responseData.Response.Torrent.ReleaseName
		uploader := responseData.Response.Torrent.Username
		log.Debug().Msgf("[%s] Checking release: %s - (Uploader: %s) (TorrentID: %d)", indexer, releaseName, uploader, id)
	}

	return responseData, nil
}

func fetchResponseData(requestData *RequestData, data **ResponseData, id int, action string, apiBase string) error {
	// If data is already fetched, do nothing
	if *data != nil {
		return nil
	}

	var apiKey string
	switch requestData.Indexer {
	case "redacted":
		apiKey = requestData.REDKey
	case "ops":
		apiKey = requestData.OPSKey
	default:
		return fmt.Errorf("invalid indexer: %s", requestData.Indexer)
	}

	var err error
	*data, err = initiateAPIRequest(id, action, apiKey, apiBase, requestData.Indexer)
	if err != nil {
		return fmt.Errorf("error fetching %s data: %w", action, err)
	}
	return nil
}
