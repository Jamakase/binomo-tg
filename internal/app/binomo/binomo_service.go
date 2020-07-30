package binomo

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
)

type Service interface {
	GetLastValue() Value
}

type service struct {
	bot *tgbotapi.BotAPI
}

type Value struct {
	Text string
}

func New(config Config) Service {
	bot, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		log.Panic(err)
	}

	return service{
		bot: bot,
	}
}

func (s service) GetLastValue() Value {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := s.bot.GetUpdates(u)

	if err != nil {
		log.Fatal(err)
	}

	if len(updates) == 0 {
		return Value{}
	}

	lastUpdate := updates[len(updates)-1]

	text := lastUpdate.ChannelPost.Text

	return Value{
		text,
	}
}
