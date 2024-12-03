package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/mdp/qrterminal/v3"
	"github.com/openai/openai-go"
	"go.mau.fi/util/exmime"
	"go.mau.fi/util/ffmpeg"
	"go.mau.fi/util/ptr"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"transcribe-wa/tdb"
)

var cli *whatsmeow.Client
var log waLog.Logger

var openAIClient *openai.Client
var appLogger waLog.Logger

var db *tdb.TranscriptionDB

var logLevel = "INFO"
var debugLogs = flag.Bool("debug", false, "Enable debug logs?")

var pairRejectChan = make(chan bool, 1)

var prompt = `
This is a voice recording from a phone, make sure everything is grammatically correct. Remove or clean any filler words. Anything duplicate, make sure is cut out. Summarize it as if the speaker is writing a text message and keep it really short.
`

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Errorf("Error loading .env file: %s", err)
	}

	openAIClient = openai.NewClient()

	appLogger = waLog.Stdout("Application", logLevel, true)

	dbLogger := waLog.Stdout("Database", logLevel, true)

	db, err = tdb.NewTranscriptionDB(os.Getenv("DB_DIALECT"), os.Getenv("DB_ADDRESS"), dbLogger)
	if err != nil {
		appLogger.Errorf("Error opening database: %s", err)
	}
	err = db.Upgrade()
	if err != nil {
		appLogger.Errorf("Error upgrading DB: %s", err)
	}

	flag.Parse()

	if *debugLogs {
		logLevel = "DEBUG"
	}
	log = waLog.Stdout("Main", logLevel, true)

	dbLog := waLog.Stdout("Database", logLevel, true)
	storeContainer, err := sqlstore.New(os.Getenv("DB_DIALECT"), os.Getenv("DB_ADDRESS"), dbLog)
	if err != nil {
		log.Errorf("Failed to connect to database: %v", err)
		return
	}
	device, err := storeContainer.GetFirstDevice()
	if err != nil {
		log.Errorf("Failed to get device: %v", err)
		return
	}

	cli = whatsmeow.NewClient(device, waLog.Stdout("Client", logLevel, true))
	var isWaitingForPair atomic.Bool
	cli.PrePairCallback = func(jid types.JID, platform, businessName string) bool {
		isWaitingForPair.Store(true)
		defer isWaitingForPair.Store(false)
		log.Infof("Pairing %s (platform: %q, business name: %q). Type r within 3 seconds to reject pair", jid, platform, businessName)
		select {
		case reject := <-pairRejectChan:
			if reject {
				log.Infof("Rejecting pair")
				return false
			}
		case <-time.After(3 * time.Second):
		}
		log.Infof("Accepting pair")
		return true
	}

	ch, err := cli.GetQRChannel(context.Background())
	if err != nil {
		if !errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
			log.Errorf("Failed to get QR channel: %v", err)
		}
	} else {
		go func() {
			for evt := range ch {
				if evt.Event == "code" {
					qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				} else {
					log.Infof("QR channel result: %s", evt.Event)
				}
			}
		}()
	}

	cli.AddEventHandler(handler)
	err = cli.Connect()
	if err != nil {
		log.Errorf("Failed to connect: %v", err)
		return
	}

	c := make(chan os.Signal, 1)
	input := make(chan string)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		defer close(input)
		scan := bufio.NewScanner(os.Stdin)
		for scan.Scan() {
			line := strings.TrimSpace(scan.Text())
			if len(line) > 0 {
				input <- line
			}
		}
	}()
	for {
		select {
		case <-c:
			log.Infof("Interrupt received, exiting")
			cli.Disconnect()
			return
		case cmd := <-input:
			if len(cmd) == 0 {
				log.Infof("Stdin closed, exiting")
				cli.Disconnect()
				return
			}
			if isWaitingForPair.Load() {
				if cmd == "r" {
					pairRejectChan <- true
				} else if cmd == "a" {
					pairRejectChan <- false
				}
				continue
			}
		}
	}

}

func handler(rawEvt interface{}) {
	switch evt := rawEvt.(type) {
	case *events.Connected:
		log.Infof("Connection to WA open")
	case *events.StreamReplaced:
		os.Exit(0)
	case *events.Message:
		ctx := context.Background()
		log.Infof("Received message %s from %s at %v (%s), ", evt.Info.ID, evt.Info.SourceString(), evt.Info.Timestamp, evt.Message)
		var messageText string
		if conversation := evt.Message.Conversation; conversation != nil {
			messageText = *conversation
		} else if extendedText := evt.Message.ExtendedTextMessage; extendedText != nil {
			messageText = *extendedText.Text
		}

		if !evt.Info.IsFromMe {
			return // only my messages allowed (for now)
		}

		chatInitialized := false
		chatEnabled := false
		chatId := evt.Info.Chat.User
		rowEnabled, err := db.GetTranscribeChat(chatId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				chatInitialized = false
				chatEnabled = false
			} else {
				appLogger.Errorf("failed to get transcribe chat from db: %e", err)
			}
		} else {
			chatEnabled = rowEnabled
			chatInitialized = true
		}

		switch messageText {
		case "ping":
			cli.SendMessage(ctx, evt.Info.Chat, &waE2E.Message{
				Conversation: ptr.Ptr("pong"),
			})
		case "tenable", "tdisable":
			if chatInitialized {
				enabled := messageText == "tenable"
				err = db.UpdateChatEnabled(chatId, enabled)
				if err != nil {
					appLogger.Errorf("failed to enable transcription for chat from db: %e", err)
					return
				}
			} else {
				if messageText == "tdisable" {
					return
				}
				err = db.CreateTranscribeChat(chatId, true)
				if err != nil {
					appLogger.Errorf("failed to create transcription for chat in db: %e", err)
					return
				}
			}

			reaction := cli.BuildReaction(evt.Info.Chat, evt.Info.Sender, evt.Info.ID, "👍")
			cli.SendMessage(ctx, evt.Info.Chat, reaction)
		}

		if !chatEnabled {
			return
		}

		if audio := evt.Message.AudioMessage; chatEnabled && audio != nil {
			data, err := cli.Download(audio)

			if !ffmpeg.Supported() {
				log.Errorf("ffmpeg is not supported on this system")
				os.Exit(1)
			}

			/*data, err = ffmpeg.ConvertBytes(ctx, data, ".mp3", []string{}, []string{
				"-acodec", "libmp3lame",
				"-ar", "44100",
				"-ac", "2",
				"-b:a", "192k",
			}, )*/

			mime := audio.GetMimetype()
			transcription, err := openAIClient.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
				Model:    openai.F(openai.AudioModelWhisper1),
				File:     openai.FileParam(bytes.NewReader(data), "file."+exmime.ExtensionFromMimetype(mime), mime),
				Language: openai.F("en"),
				Prompt:   openai.F(prompt),
			})

			if err != nil {
				log.Errorf("Failed to transcribe: %v", err)
			}

			contextInfo := &waE2E.ContextInfo{}
			contextInfo.StanzaID = proto.String(evt.Info.ID)
			contextInfo.Participant = proto.String(evt.Info.Sender.String())
			contextInfo.QuotedMessage = evt.Message

			_, err = cli.SendMessage(ctx, evt.Info.Chat, &waE2E.Message{
				ExtendedTextMessage: &waE2E.ExtendedTextMessage{
					Text:        ptr.Ptr(fmt.Sprintf("*Transcription:*\n%s", transcription.Text)),
					ContextInfo: contextInfo,
				},
			})
			if err != nil {
				log.Errorf("Failed to send message: %v", err)
			}
			return
		}
	case *events.KeepAliveTimeout:
		log.Debugf("Ping timeout event: %+v", evt)
	case *events.KeepAliveRestored:
		log.Debugf("Ping success")
	case *events.OfflineSyncCompleted:
		log.Infof("sync completed")
	}
}
