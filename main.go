package main

import (
	//	"bytes"

	"context"
	"log"
	"math/big"
	"os"

	//"strconv"
	//	"strings"
	passport "github.com/MoonSHRD/IKY-telegram-bot/artifacts/TGPassport"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var tgApiKey, err = os.ReadFile(".secret")
var bot, error1 = tgbotapi.NewBotAPI(string(tgApiKey))

type user struct {
	DialogStatus int64
	ChatID       int64
}

var userDatabase = make(map[int64]user)

var buySellKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Buy"),
		tgbotapi.NewKeyboardButton("Sell"),
	),
)

var nullAddress common.Address = common.HexToAddress("0x0000000000000000000000000000000000000000")

func main() {

	_ = godotenv.Load()
	//ctx := context.Background()
	pk := os.Getenv("PK") // load private key from env
	gateway := os.Getenv("GATEWAY_GOERLI_WS")

	passportAddress := os.Getenv("PASSPORT_ADDRESS")

	// setting up private key in proper format
	privateKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		log.Fatal(err)
	}

	// Creating an auth transactor
	auth, _ := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(5))
	if err != nil {
		log.Fatalf("could not connect to auth gateway: %v\n", err)
	}

	bot, err = tgbotapi.NewBotAPI(string(tgApiKey))
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	client, err := ethclient.Dial(gateway) // load from local .env file
	if err != nil {
		log.Fatalf("could not connect to Ethereum gateway: %v\n", err)
	}
	defer client.Close()

	passportCaller, err := passport.NewPassportCaller(common.HexToAddress(passportAddress), client)
	if err != nil {
		log.Fatalf("Failed to instantiate a Passport contract: %v", err)
	}

	updates := bot.GetUpdatesChan(u)

	for update := range updates {

		if _, ok := userDatabase[update.Message.Chat.ID]; !ok {

			userDatabase[update.Message.Chat.ID] = user{0, update.Message.Chat.ID}

			isRegistered := checkUser(auth, passportCaller, update.Message.Chat.ID)

			if isRegistered {
				updateDb := userDatabase[update.Message.Chat.ID]

				//updateDb = restoreUserViaJson(tgdb, update.Message.Chat.ID) //if the chat is registered, it is present in our db

				updateDb.DialogStatus = 1

				himsg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Hello! This bot is designed for buying/selling NFT.")
				bot.Send(himsg)

				msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Found you in our database! Use the provided keyboard to operate the bot.")
				msg.ReplyMarkup = buySellKeyboard
				bot.Send(msg)
				userDatabase[update.Message.Chat.ID] = updateDb

			} else {
				himsg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Hello! This bot is designed for buying/selling NFT.")
				bot.Send(himsg)

				msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Please, go to <IKYBOTLINK> for registration, it takes only a couple of minutes.")
				bot.Send(msg)
				delete(userDatabase, update.Message.Chat.ID)
			}

		} else if userDatabase[update.Message.Chat.ID].DialogStatus == 1 {

		}
	}
}

func checkUser(auth *bind.TransactOpts, pc *passport.PassportCaller, Tgid int64) bool {
	registration, err := pc.TgIdToAddress(&bind.CallOpts{
		From:    auth.From,
		Context: context.Background(),
	}, Tgid)

	log.Println(registration)

	if err != nil {
		log.Print(err)
	}

	if registration == nullAddress {
		return false
	} else {
		return true
	}
}
