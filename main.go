package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"go.uber.org/zap"
)

func main() {

	batchSize := validateNumber("BATCH_SIZE", os.Getenv("BATCH_SIZE"))
	batchIntvl := validateNumber("BATCH_INTERVAL", os.Getenv("BATCH_INTERVAL"))
	postURL := validateURL(os.Getenv("POST_URL"))
	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = "8080"
	}

	newLogHandler := NewLogHandler(batchSize, batchIntvl, postURL)

	server := http.NewServeMux()

	loggerMiddleware := newLogHandler.WithLogger
	server.HandleFunc("/healthz", loggerMiddleware(http.HandlerFunc(newLogHandler.Healthz)))
	server.HandleFunc("/log", loggerMiddleware(http.HandlerFunc(newLogHandler.LogRequest)))
	server.HandleFunc("/postLog", loggerMiddleware(http.HandlerFunc(newLogHandler.PostLogRequest)))

	newLogHandler.log.Info("APP is running on ", zap.String(":PORT", PORT))
	if err := http.ListenAndServe(":"+PORT, server); err != nil {
		close(newLogHandler.logChan)
		newLogHandler.log.Error("error starting server ", zap.Error(err))
	}
}

func validateNumber(t, size string) int {
	s, err := strconv.Atoi(size)

	if err != nil {
		log.Fatalf("%s is required number, err %v", t, err)
		os.Exit(1)
	}

	return s
}

func validateURL(postURL string) string {
	u, err := url.Parse(postURL)
	if err != nil {
		log.Fatalf("POST_URL is required to be url string, err %v", err)
		os.Exit(1)
	}

	return u.String()
}
