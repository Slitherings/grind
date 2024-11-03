package services

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func MakeGoPlusRequest(url string) (*http.Response, error) {
	const (
		MAX_RETRIES = 3
		RETRY_DELAY = 2 * time.Second
	)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var lastErr error
	for i := 0; i < MAX_RETRIES; i++ {
		if i > 0 {
			time.Sleep(RETRY_DELAY)
			log.Printf("Retrying request (%d/%d)...", i+1, MAX_RETRIES)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Check if we need to retry based on status code
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

type GoPlusResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Solana struct {
			LockInfo struct {
				IsLocked     bool    `json:"is_locked"`
				LockedAmount string  `json:"locked_amount"`
				Percentage   float64 `json:"percentage"`
				EndTime      string  `json:"end_time"`
			} `json:"lock_info"`
		} `json:"solana"`
	} `json:"data"`
}
