package main

import (
	//	"bytes"

	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"

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
	DialogStatus        int64
	ChatID              int64
	Currency            string
	Minprice            int64
	Category            string
	IsSingleton         bool
	IsMarketable        bool
	NonSingletonAddress string
	NonSingletonTokenID int64
}

var baseSellURLTGSingleton = "http://localhost:3000/singleton"
var baseSellURLOtherCollection = "http://localhost:3000/nonsingleton"

var currency_query = "?currency="
var minprice_query = "&minprice="
var category_query = "&category="

var address_query = "&address="
var tokenid_query = "&tokenid="

var userDatabase = make(map[int64]user)

var buySellKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Buy"),
		tgbotapi.NewKeyboardButton("Sell"),
	),
)

var yesNoKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Yes"),
		tgbotapi.NewKeyboardButton("No"),
	),
)

var sellKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Telegram Singleton"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Other collection"),
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

			userDatabase[update.Message.Chat.ID] =
				user{0, //status
					update.Message.Chat.ID, //chatid
					"0",                    //currency
					0,                      //minprice
					"default",              //category
					false,                  //isSingleton
					false,                  //isMarketable
					"0x0",                  //NFT address if not TGSingleton
					0}                      //TokenID if not TGSingleton

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

		} else {
			switch userDatabase[update.Message.Chat.ID].DialogStatus {

			//main menu
			case 1:
				switch update.Message.Text {

				case "Sell":
					msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Okay! Was your NFT minted at our wizard-bot (Telegram Singleton) or is it from another collection?")
					msg.ReplyMarkup = sellKeyboard
					bot.Send(msg)

					updateDb := userDatabase[update.Message.Chat.ID]
					updateDb.DialogStatus = 21
					userDatabase[update.Message.Chat.ID] = updateDb

				case "Buy":

				default:
					msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Please use the provided keyboard!")
					msg.ReplyMarkup = buySellKeyboard
					bot.Send(msg)
				}

			//sell sequence
			case 21:

				if update.Message.Text == "Telegram Singleton" {

					msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Cool! What currency you would like to sell your NFT for?")
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(false)
					bot.Send(msg)

					updateDb := userDatabase[update.Message.Chat.ID]
					updateDb.IsSingleton = true
					updateDb.DialogStatus = 41
					userDatabase[update.Message.Chat.ID] = updateDb

				} else if update.Message.Text == "Other collection" {
					msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Neat. Is your NFT marketable?")
					msg.ReplyMarkup = yesNoKeyboard
					bot.Send(msg)

					updateDb := userDatabase[update.Message.Chat.ID]
					updateDb.IsSingleton = false
					updateDb.DialogStatus = 31
					userDatabase[update.Message.Chat.ID] = updateDb

				} else {
					msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Please, specify the collection!")
					msg.ReplyMarkup = sellKeyboard
					bot.Send(msg)
				}

			//extra data gathering for non-TGSingleton items
			case 31:
				if update.Message.Text == "Yes" {
					msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Great! Please, enter your NFT address")
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(false)
					bot.Send(msg)

					updateDb := userDatabase[update.Message.Chat.ID]
					updateDb.DialogStatus = 32
					updateDb.IsMarketable = true
					userDatabase[update.Message.Chat.ID] = updateDb

					//TODO: Non-marketable branch
				} else if update.Message.Text == "No" {
					msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "NON-MARKETABLE BLOCK GOES HERE\n...Conversation branch under construction...")
					msg.ReplyMarkup = yesNoKeyboard
					bot.Send(msg)

					updateDb := userDatabase[update.Message.Chat.ID]
					updateDb.DialogStatus = 31
					userDatabase[update.Message.Chat.ID] = updateDb

				} else {
					msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "That was a Yes/No question🙂")
					msg.ReplyMarkup = yesNoKeyboard
					bot.Send(msg)
				}

			case 32:
				msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Okay, now please send token id")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(false)
				bot.Send(msg)

				updateDb := userDatabase[update.Message.Chat.ID]
				updateDb.DialogStatus = 33
				updateDb.NonSingletonAddress = update.Message.Text
				userDatabase[update.Message.Chat.ID] = updateDb

			case 33:
				msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Cool! What currency you would like to sell your NFT for?")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(false)
				bot.Send(msg)

				updateDb := userDatabase[update.Message.Chat.ID]
				updateDb.DialogStatus = 41
				TokenIDstring, _ := strconv.ParseInt(update.Message.Text, 10, 64)
				updateDb.NonSingletonTokenID = TokenIDstring
				userDatabase[update.Message.Chat.ID] = updateDb

			//sell sequence currency, minprice & category setup
			case 41:
				updateDb := userDatabase[update.Message.Chat.ID]
				updateDb.Currency = update.Message.Text
				updateDb.DialogStatus = 42
				userDatabase[update.Message.Chat.ID] = updateDb
				msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Got it. What's the minimum price you want to sell your NFT for?")
				bot.Send(msg)

			case 42:
				minpriceint, _ := strconv.ParseInt(update.Message.Text, 10, 64)

				updateDb := userDatabase[update.Message.Chat.ID]
				updateDb.Minprice = minpriceint
				updateDb.DialogStatus = 43
				userDatabase[update.Message.Chat.ID] = updateDb

				msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Okay, from what category your NFT is from?")
				bot.Send(msg)

			case 43:
				updateDb := userDatabase[update.Message.Chat.ID]
				updateDb.Category = update.Message.Text
				updateDb.DialogStatus = 1
				userDatabase[update.Message.Chat.ID] = updateDb
				minpriceString := fmt.Sprint(userDatabase[update.Message.Chat.ID].Minprice)

				link := baseSellURLTGSingleton +
					currency_query + userDatabase[update.Message.Chat.ID].Currency +
					minprice_query + minpriceString +
					category_query + userDatabase[update.Message.Chat.ID].Category

				if !userDatabase[update.Message.Chat.ID].IsSingleton {
					tokenIdString := fmt.Sprint(userDatabase[update.Message.Chat.ID].NonSingletonTokenID)
					link = baseSellURLOtherCollection +
						currency_query + userDatabase[update.Message.Chat.ID].Currency +
						minprice_query + minpriceString +
						category_query + userDatabase[update.Message.Chat.ID].Category +
						address_query + userDatabase[update.Message.Chat.ID].NonSingletonAddress +
						tokenid_query + tokenIdString
				}

				msg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, "Here's the link to list your NFT on our marketplace!")
				msg.ReplyMarkup = buySellKeyboard
				bot.Send(msg)

				linkMsg := tgbotapi.NewMessage(userDatabase[update.Message.Chat.ID].ChatID, link)
				bot.Send(linkMsg)
				//end of Sell sequence

			}
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
