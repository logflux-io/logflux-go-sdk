package main

import (
	"context"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	logrus_integration "github.com/logflux-io/logflux-go-sdk/pkg/integrations/logrus"
	"github.com/sirupsen/logrus"
)

func main() {
	// Create LogFlux batch client
	batchConfig := config.DefaultBatchConfig()
	batchConfig.MaxBatchSize = 10
	batchConfig.FlushInterval = 2 * time.Second

	batchClient := client.NewBatchUnixClient("/tmp/logflux-agent.sock", batchConfig)

	// Connect to LogFlux agent
	ctx := context.Background()
	if err := batchClient.Connect(ctx); err != nil {
		panic("Failed to connect to LogFlux agent: " + err.Error())
	}
	defer batchClient.Close()

	// Create LogFlux logrus hook
	hook := logrus_integration.NewHook(batchClient, "logrus-example")

	// Add hook to logrus
	logrus.AddHook(hook)
	logrus.SetLevel(logrus.DebugLevel)

	// Use logrus as normal - logs will be sent to LogFlux
	logrus.Info("Application started")
	logrus.WithFields(logrus.Fields{
		"version":   "1.0.0",
		"component": "main",
	}).Info("System initialized")

	logrus.WithFields(logrus.Fields{
		"user_id": 123,
		"action":  "login",
	}).Warn("Login attempt with expired token")

	logrus.WithFields(logrus.Fields{
		"error":       "connection timeout",
		"retry_count": 3,
		"timeout":     "30s",
	}).Error("Database connection failed")

	// Give time for batch to flush
	time.Sleep(3 * time.Second)
}
