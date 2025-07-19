package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/robfig/cron/v3"
	"irccloud-watcher/internal/api"
	"irccloud-watcher/internal/config"
	"irccloud-watcher/internal/storage"
	"irccloud-watcher/internal/summary"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to the configuration file")
	generateSummary := flag.Bool("generate-summary", false, "Generate a summary and exit")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := storage.NewDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if *generateSummary {
		summaryGenerator := summary.NewSummaryGenerator()
		if err := summaryGenerator.GenerateDailySummary(db, cfg.SummaryOutputPath); err != nil {
			log.Fatalf("Failed to generate summary: %v", err)
		}
		os.Exit(0)
	}

	client := api.NewIRCCloudClient(db)
	if err := client.Connect(cfg.Email, cfg.Password); err != nil {
		log.Fatalf("Failed to connect to IRCCloud: %v", err)
	}
	defer client.Close()

	c := cron.New()
	_, err = c.AddFunc(cfg.SummaryTime, func() {
		summaryGenerator := summary.NewSummaryGenerator()
		if err := summaryGenerator.GenerateDailySummary(db, cfg.SummaryOutputPath); err != nil {
			log.Printf("Failed to generate summary: %v", err)
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
	go client.Run(cfg.Channels, cfg.IgnoredChannels)

	// Wait for shutdown signal
	<-quit
	log.Println("🛑 Shutting down gracefully...")

	// Cleanup is handled by defer statements
}
