package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"wa-bot/internal/config"
	"wa-bot/internal/handler"
	"wa-bot/internal/utils"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func main() {
	startTime := time.Now()
	rand.Seed(time.Now().UnixNano())

  config.LoadOwnerConfig()
	config.LoadGroupConfig()

	dbLog := waLog.Stdout("Database", "ERROR", true)
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:session.db?_foreign_keys=on", dbLog)
	if err != nil { panic(err) }

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil { panic(err) }

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	botHandler := handler.NewBotHandler(client, startTime)
	client.AddEventHandler(botHandler.EventHandler)

	if client.Store.ID == nil {
		fmt.Println(utils.ColorCyan + "=== LOGIN PAIRING ===" + utils.ColorReset)
		fmt.Print("Nomor HP (62xxx): ")
		var phone string
		fmt.Scanln(&phone)

		if !strings.HasPrefix(phone, "62") {
			fmt.Println("❌ Awal nomor harus 62")
			return
		}

		client.Connect()
		time.Sleep(3 * time.Second)

		code, err := client.PairPhone(context.Background(), phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
		if err != nil { panic(err) }
		fmt.Println(utils.ColorGreen + "✅ CODE: " + code + utils.ColorReset)
	} else {
		client.Connect()
	}

	fmt.Println(utils.ColorGreen + "\n✅ BOT ONLINE MODULAR" + utils.ColorReset)
	
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	client.Disconnect()
}
