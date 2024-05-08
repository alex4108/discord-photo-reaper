package main

import (
	"os"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/drive/v3"
)

func main() {
	if os.Getenv("DAEMON") == "1" {
		seconds, err := strconv.ParseInt(os.Getenv("DAEMON_SLEEP_SECONDS"), 10, 0)
		if err != nil {
			log.Fatalf("Invalid DAEMON_SLEEP_SECONDS: %v", err)
		}
		run()

		ticker := time.NewTicker(time.Duration(seconds) * time.Second)
		defer ticker.Stop()
		go func() {
			for range ticker.C {
				log.Debugf("Woke up after %v seconds", os.Getenv("DAEMON_SLEEP_SECS"))
				run()
			}
		}()

		select {}
	} else {
		run()
	}
}

func run() {
	setupLogs()

	token := os.Getenv("DISCORD_BOT_TOKEN")

	dg := initDiscordGo(token)
	log.Info("Discord init'ed")

	driveService := initGDriveSvc(
		os.Getenv("GOOGLE_CREDENTIALS_FILE"),
		os.Getenv("GOOGLE_TOKEN_FILE"),
	)
	log.Info("GDrive init'd")

	processedFilePath = os.Getenv("STATE_FILE")
	urlFile := initProcessedEntities()
	log.Info("State file init'ed")

	initMetrics()
	log.Info("Metrics init'd")

	if os.Getenv("RUN_E2E") == "1" {
		log.Warn("Running E2E")
		validateCanDownloadFile(
			dg,
			driveService,
			os.Getenv("E2E_CHANNEL_ID"),
			os.Getenv("E2E_MESSAGE_ID"),
		)
		log.Info("E2E Completed")
		os.Exit(0)
	}

	channel_ids := getChannelIds(dg, os.Getenv("DISCORD_GUILD_ID"))

	for _, channel_id := range channel_ids {
		scanChannel(dg, channel_id, driveService)
	}

	log.Infof("All files downloaded.")

	defer dg.Close()
	defer urlFile.Close()
	lastRunSuccess.WithLabelValues().Set(1)
	log.Infof("The application completed successfully.")
}

func validateCanDownloadFile(dg *discordgo.Session, driveService *drive.Service, channelID string, messageID string) error {
	msgs, err := dg.ChannelMessages(channelID, 1, "", "", messageID)
	if err != nil {
		log.Fatalf("error fetching message with ID %s: %v", messageID, err)
	}
	log.Debugf("E2E: Got Message %v", msgs)
	msg := msgs[0]

	// Check if the message has attachments
	if len(msg.Attachments) == 0 {
		log.Fatalf("no attachments found in message with ID %s", messageID)
	}

	var messages []*discordgo.Message
	messages = append(messages, msg)
	log.Debugf("Scanning....")
	scanMessages(driveService, messages)
	log.Debugf("Scan completed")
	return nil
}
