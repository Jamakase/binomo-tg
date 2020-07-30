package bot

import (
	"github.com/awesomeProject/internal/app/binomo"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"time"
)

type BotService interface {
}

type botService struct {
	binomoService binomo.Service
	config        *Config
	tgApi         *tgbotapi.BotAPI
}

func New(config Config, service binomo.Service) BotService {
	bot, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	bs := botService{
		binomoService: service,
		tgApi:         bot,
		config:        &config,
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message updates
			continue
		}

		if !update.Message.IsCommand() { // ignore any non-command Messages
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		switch update.Message.Command() {
		case "help":
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "I know next commands:"))
		case "status":
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "I'm ok."))
		case "test":
			go bs.scheduleExecution(update.Message)
		}
	}

	return bs
}

type cmd interface {
	Execute()
	Duration() time.Duration
}

type messageCommand struct {
	msg   tgbotapi.Chattable
	d     time.Duration
	tgApi *tgbotapi.BotAPI
}

func (m messageCommand) Duration() time.Duration {
	return m.d
}

func (m messageCommand) Execute() {
	m.tgApi.Send(m.msg)
}

type customCommand struct {
	d time.Duration
	f func()
}

func (c customCommand) Execute() {
	c.f()
}

func (c customCommand) Duration() time.Duration {
	return c.d
}

func (bt botService) scheduleExecution(msg *tgbotapi.Message) {
	bt.tgApi.Send(tgbotapi.NewMessage(msg.Chat.ID, "Scheduled timer"))

	commands := []cmd{
		messageCommand{
			d:     20 * time.Second,
			msg:   tgbotapi.NewMessage(msg.Chat.ID, "Мы запостим информацию через час"),
			tgApi: bt.tgApi,
		},
		messageCommand{
			d:      10 * time.Second,
			msg:   tgbotapi.NewMessage(msg.Chat.ID, "Мы запостим информацию через 10 минут"),
			tgApi: bt.tgApi,
		},
		messageCommand{
			d:     3 * time.Second,
			msg:   tgbotapi.NewMessage(msg.Chat.ID, "Мы запостим информацию через 5 миинут"),
			tgApi: bt.tgApi,
		},
		customCommand{
			d: 20 * time.Second,
			// TODO: think how loc it here
			f: func() {
				value := bt.binomoService.GetLastValue()
				bt.tgApi.Send(tgbotapi.NewMessage(msg.Chat.ID, value.Text))
			},
		},
	}

	ch := make(chan struct{})

	for _, command := range commands {
		timer := time.AfterFunc(command.Duration(), func() {
			command.Execute()
			ch <- struct{}{}
		})
		<-ch
		timer.Stop()
	}
}
