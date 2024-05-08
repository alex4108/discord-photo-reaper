package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/drive/v3"
)

// Scans a channel, fetching all messages and processing them
func scanChannel(dg *discordgo.Session, channelId string, driveService *drive.Service) {
	var lastMessageId string
	var wg sync.WaitGroup

	maxConcurrentGoroutines := 5
	maxConcurrentGoroutinesStr := os.Getenv("MAX_CONCURRENT_GOROUTINES")

	if maxConcurrentGoroutinesStr != "" {
		maxConcurrentGoroutines, err := strconv.Atoi(maxConcurrentGoroutinesStr)
		log.Debugf("Max Routines: %d", maxConcurrentGoroutines)
		if err != nil {
			log.Errorf("Failed to parse MAX_CONCURRENT_GOROUTINES: %v", err)
		}
	}

	semaphore := make(chan struct{}, maxConcurrentGoroutines)

	for {
		messages, err := dg.ChannelMessages(channelId, 100, lastMessageId, "", "")
		if err != nil {
			// Handle rate limits by retrying after a delay
			if discordErr, ok := err.(*discordgo.RESTError); ok && discordErr.Response.StatusCode == 429 {
				retryAfter := discordErr.Response.Header.Get("Retry-After")
				waitDuration, err := time.ParseDuration(retryAfter + "ms")
				if err == nil {
					log.Warnf("Rate limit encountered, retrying after %s", waitDuration.String())
					time.Sleep(waitDuration)
					continue
				}
			} else {
				log.Errorf("Failed to fetch messages in channel %s: %v", channelId, err)
				break
			}
		}

		if len(messages) == 0 {
			log.Infof("Completed scan for channel %s", channelId)
			break
		}

		semaphore <- struct{}{}
		wg.Add(1)

		go func(messages []*discordgo.Message, driveService *drive.Service) {
			defer wg.Done()                // Signal completion
			defer func() { <-semaphore }() // Release semaphore slot
			log.Debugf("Start scanner for batch %s %s", channelId, lastMessageId)
			scanMessages(driveService, messages)
		}(messages, driveService)

		lastMessageId = messages[len(messages)-1].ID
	}

	wg.Wait()
}

func initDiscordGo(token string) *discordgo.Session {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		Fail(fmt.Sprintf("error creating Discord session %v", err))
	}
	intents := discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent
	dg.Identify.Intents = intents

	err = dg.Open()
	if err != nil {
		Fail(fmt.Sprintf("error opening connection %v", err))
	}

	return dg
}

func getChannelIds(dg *discordgo.Session, guildId string) []string {
	channels, err := dg.GuildChannels(guildId)
	if err != nil {
		log.Fatalf("Error fetching channels for guild %s: %v", guildId, err)
	}

	channelIds := []string{}

	// Add the channel IDs to our list
	for _, channel := range channels {
		log.Debugf("Got channel %s %s", channel.Name, channel.ID)
		channelIds = append(channelIds, channel.ID)
	}

	log.Infof("Got all channels")
	return channelIds
}

func scanMessages(driveService *drive.Service, messages []*discordgo.Message) {
	start := time.Now()
	for _, message := range messages {
		log.Debugf("Message: %v", message)
		for _, attachment := range message.Attachments {
			log.Debugf("Attachment: %v", attachment)
			log.Debugf("Start download for file %s %s", attachment.URL, attachment.Filename)
			download(
				attachment.URL,
				attachment.Filename,
				attachment.ContentType,
				attachment.Size,
				driveService,
			)
		}
	}
	messagesChecked.Add(float64(len(messages)))
	batchProcessingTime.WithLabelValues().Observe(float64(time.Since(start).Seconds()))
}
