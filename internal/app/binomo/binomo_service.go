package binomo

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"strings"
)

type Service interface {
	GetLastValue() []Pair
}

type service struct {
	bot *tgbotapi.BotAPI
}

type Pair struct {
	PairName string
	Type     string
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

func (s service) GetLastValue() []Pair {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := s.bot.GetUpdates(u)

	if err != nil {
		log.Fatal(err)
	}

	if len(updates) == 0 {
		return []Pair{}
	}

	lastUpdate := updates[len(updates)-1]

	text := lastUpdate.ChannelPost.Text

	return parsePairs(text)
}

func parsePairs(str string) []Pair {
	rows := strings.Split(str, "\n")
	pairs := make([]Pair, len(rows))

	for i, row := range rows {
		v := strings.Split(row, "|")
		pairs[i] = Pair{
			PairName: v[0],
			Type:     v[1],
		}
	}

	return pairs
}
