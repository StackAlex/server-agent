package monitor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	networkMu        sync.RWMutex
	lastNetworkSpeed NetworkSpeed
	lastNetworkCheck time.Time
)

func SetNetworkSpeed(speed NetworkSpeed) {
	networkMu.Lock()
	defer networkMu.Unlock()

	lastNetworkSpeed = speed
	lastNetworkCheck = time.Now().UTC()
}

func GetNetworkSpeed() NetworkSpeed {
	networkMu.RLock()
	defer networkMu.RUnlock()

	return lastNetworkSpeed
}

type NetworkSpeed struct {
	DownloadMbps float64 `json:"download_mbps"`
	UploadMbps   float64 `json:"upload_mbps"`
	LatencyMs    float64 `json:"latency_ms"`
}

const (
	testURL       = "https://speed.cloudflare.com/__down?bytes=10000000"
	uploadURL     = "https://speed.cloudflare.com/__up"
	downloadBytes = 10_000_000
	uploadBytes   = 5_000_000
)

func MeasureNetworkSpeed(ctx context.Context) (*NetworkSpeed, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	result := &NetworkSpeed{}

	// Измеряем задержку соединения.
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://speed.cloudflare.com/", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network latency test failed: %w", err)
	}

	resp.Body.Close()
	result.LatencyMs = float64(time.Since(start).Microseconds()) / 1000

	// Измеряем скорость скачивания.
	start = time.Now()

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download test failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("download test returned HTTP %d", resp.StatusCode)
	}

	downloaded, err := io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	elapsed := time.Since(start).Seconds()
	if elapsed > 0 && downloaded > 0 {
		result.DownloadMbps = float64(downloaded*8) / elapsed / 1_000_000
	}

	// Измеряем скорость отдачи.
	data := bytes.Repeat([]byte("a"), uploadBytes)

	start = time.Now()

	req, err = http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload test failed: %w", err)
	}

	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upload test returned HTTP %d", resp.StatusCode)
	}

	elapsed = time.Since(start).Seconds()
	if elapsed > 0 {
		result.UploadMbps = float64(uploadBytes*8) / elapsed / 1_000_000
	}

	return result, nil
}
func StartNetworkMonitor(ctx context.Context) {
	go func() {
		measure := func() {
			speed, err := MeasureNetworkSpeed(ctx)
			if err != nil {
				log.Printf("Network speed measurement failed: %v", err)
				return
			}

			SetNetworkSpeed(*speed)

			log.Printf(
				"Network speed: download=%.2f Mbps, upload=%.2f Mbps, latency=%.2f ms",
				speed.DownloadMbps,
				speed.UploadMbps,
				speed.LatencyMs,
			)
		}

		// Первое измерение сразу после запуска.
		measure()

		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				measure()
			}
		}
	}()
}
