package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const lokiURL = "http://localhost:3100/loki/api/v1/push"

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type lokiPush struct {
	Streams []lokiStream `json:"streams"`
}

func pushLog(level, msg string) {
	ts := strconv.FormatInt(time.Now().UnixNano(), 10)

	payload := lokiPush{
		Streams: []lokiStream{
			{
				Stream: map[string]string{
					"service": "onboarding_be",
					"env":     "local",
					"level":   level,
				},
				Values: [][]string{
					{ts, msg},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(lokiURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("[loki-push-error] %v\n", err)
		return
	}
	resp.Body.Close()
}

func logInfo(msg string) {
	fmt.Printf("[INFO] %s\n", msg)
	pushLog("info", msg)
}

func logError(msg string) {
	fmt.Printf("[ERROR] %s\n", msg)
	pushLog("error", msg)
}
