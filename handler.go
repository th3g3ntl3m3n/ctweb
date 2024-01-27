package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
)

type BatchData []*LogRequest

type HttpHandler struct {
	Size        int
	Interval    int
	PostRequest string

	logChan   chan LogRequest
	batchData BatchData
	timout    <-chan time.Time
	totalData int
	log       *zap.Logger
}

func NewLogHandler(size int, interval int, url string) *HttpHandler {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	h := &HttpHandler{
		Size:        size,
		Interval:    interval,
		PostRequest: url,
		logChan:     make(chan LogRequest),
		batchData:   make(BatchData, 0),
		timout:      time.Tick(time.Duration(interval) * time.Millisecond),
		log:         logger,
	}

	go h.processHandler()

	return h
}

func (h *HttpHandler) WithLogger(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.log.Info("Got new request", zap.String("path", r.URL.Path))
		next.ServeHTTP(w, r)
	})
}

func (handler *HttpHandler) processHandler() {
	for {
		select {
		case lReq, ok := <-handler.logChan:
			if !ok {
				handler.log.Error("can't read anything from channel bye")
				return
			}
			handler.batchData = append(handler.batchData, &lReq)
			if len(handler.batchData) == handler.Size {
				go postUrl(handler.batchData, handler.PostRequest, 0, handler.log)
				handler.batchData = make(BatchData, 0)
			}
		case <-handler.timout:
			go postUrl(handler.batchData, handler.PostRequest, 0, handler.log)
			handler.batchData = make(BatchData, 0)
		}
	}
}

func postUrl(data BatchData, url string, retry int, log *zap.Logger) {
	if retry == 3 {
		log.Error("Error reaching to url, exiting")
		os.Exit(1)
		return
	}

	t := time.Now()
	// can check error but this data is already check when request came
	jsonString, _ := json.Marshal(data)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonString))
	if err != nil {
		log.Error("error sending data to external api ", zap.String("url", url), zap.Duration("time taken", time.Since(t)))

		return
	}
	// any case response server failed to reach out or something hapapn
	if resp.StatusCode >= 500 {
		time.Sleep(2 * time.Second)
		postUrl(data, url, retry+1, log)
	}
	log.Info("data sent to external api ", zap.String("url", url), zap.Duration("time taken", time.Since(t)))
}

func sendResponse(w http.ResponseWriter, code int, response string) {
	w.WriteHeader(code)
	fmt.Fprintln(w, response)
}

func (h *HttpHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sendResponse(w, http.StatusOK, "OK")
}

func (h *HttpHandler) LogRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var lRequest = LogRequest{}

	if err := json.NewDecoder(r.Body).Decode(&lRequest); err != nil {
		sendResponse(w, http.StatusBadRequest, fmt.Sprintf("bad request %v", err))
		return
	}

	h.logChan <- lRequest

	sendResponse(w, http.StatusOK, "Got it, Thanks")
}

// PostLogRequest is a test endpoint for the program itself
func (h *HttpHandler) PostLogRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var lRequest = []*LogRequest{}

	if err := json.NewDecoder(r.Body).Decode(&lRequest); err != nil {
		sendResponse(w, http.StatusBadRequest, fmt.Sprintf("bad request %v", err))
		return
	}

	var ra = rand.Intn(10-1) + 1

	if ra%4 == 0 {
		sendResponse(w, http.StatusOK, "Got it, Thanks")
		return
	}

	sendResponse(w, http.StatusInternalServerError, "something broke in me")
}
