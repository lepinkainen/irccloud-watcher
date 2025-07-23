package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"irccloud-watcher/internal/api"
	"irccloud-watcher/internal/config"
	"irccloud-watcher/internal/storage"
	"irccloud-watcher/internal/summary"

	"github.com/alecthomas/kong"
	"github.com/robfig/cron/v3"
)

type CLI struct {
	Config          string `help:"Path to the configuration file" default:"config.yaml"`
	GenerateSummary bool   `help:"Generate a summary and exit"`
	Debug           bool   `help:"Print raw received messages to stdout in addition to formatted messages"`
}

func main() {
	var cli CLI
	kong.Parse(&cli)

	cfg, err := config.LoadConfig(cli.Config)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := storage.NewDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	if cli.GenerateSummary {
		summaryGenerator := summary.NewGenerator(cfg)
		if summaryErr := summaryGenerator.GenerateDailySummary(db, cfg.SummaryOutputPath); summaryErr != nil {
			db.Close()
			log.Fatalf("Failed to generate summary: %v", summaryErr)
		}
		db.Close()
		os.Exit(0)
	}

	defer db.Close()

	client := api.NewIRCCloudClient(db)
	client.SetConnectionConfig(&cfg.Connection)
	client.SetDebugMode(cli.Debug)
	if connectErr := client.Connect(cfg.Email, cfg.Password); connectErr != nil {
		log.Fatalf("Failed to connect to IRCCloud: %v", connectErr)
	}
	defer client.Close()

	c := cron.New()
	_, err = c.AddFunc(cfg.SummaryTime, func() {
		summaryGenerator := summary.NewGenerator(cfg)
		if cronErr := summaryGenerator.GenerateDailySummary(db, cfg.SummaryOutputPath); cronErr != nil {
			log.Printf("Failed to generate summary: %v", cronErr)
		}
	})
	if err != nil {
		log.Fatalf("Failed to schedule summary generation: %v", err)
	}
	c.Start()
	defer c.Stop()

	log.Println("🚀 IRCCloud watcher started successfully!")
	if len(cfg.Channels) > 0 {
		log.Printf("📺 Monitoring channels: %v", cfg.Channels)
	} else {
		log.Println("📺 Monitoring all channels")
	}
	if len(cfg.IgnoredChannels) > 0 {
		log.Printf("🚫 Ignoring channels: %v", cfg.IgnoredChannels)
	}
	log.Printf("💾 Database: %s", cfg.DatabasePath)
	log.Printf("📊 Summary schedule: %s", cfg.SummaryTime)

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Run the client in a goroutine
	go client.Run(cfg.Channels, cfg.IgnoredChannels, &cfg.Connection)

	// Wait for shutdown signal
	<-quit
	log.Println("🛑 Shutting down gracefully...")

	// Cleanup is handled by defer statements
}
