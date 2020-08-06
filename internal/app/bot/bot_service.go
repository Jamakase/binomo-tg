package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/awesomeProject/internal/app/binomo"
	"github.com/awesomeProject/internal/app/bot/message_flow_config"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	"math/rand"

	"strconv"
	"strings"
	"time"
)

const (
	help = `Я знаю следующие комманды:
/help: Выводит информацию о возможных командах бота
/schedule: Начинает процесс выставления таймера в канал
/addConfig: начнет процесс создания нового конфига
/listConfig: выведет все текуще существующие конфиги

во время флоу я знаю следующие команды:
/cancel: остановет процедуру создания конфига
/done: завершит текущий флоу ( создания конфига или выставление таймера )
`
)

type BotService interface {
	Run(ctx context.Context)
}

type botService struct {
	config        *Config
	tgApi         *tgbotapi.BotAPI
	binomoService binomo.Service

	repo    UserStateRepo
	cfgRepo message_flow_config.Repo
}

func New(config Config, service binomo.Service, repo UserStateRepo, cfgRepo message_flow_config.Repo) BotService {
	bot, err := tgbotapi.NewBotAPI(config.Token)

	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	return botService{
		binomoService: service,
		tgApi:         bot,
		config:        &config,
		repo:          repo,
		cfgRepo:       cfgRepo,
	}
}

func (bt botService) Run(ctx context.Context) {
	bot := bt.tgApi

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bt.tgApi.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message updates
			continue
		}

		if len(bt.config.Whitelist) > 0 {
			contains := false
			for _, id := range bt.config.Whitelist {
				if update.Message.Chat.ID == id {
					contains = true
				}
			}
			if !contains {
				continue
			}
		}

		us := bt.repo.FindUserState(ctx, chatId(update.Message.Chat.ID))

		if us == nil {
			us = &userState{
				id: chatId(update.Message.Chat.ID),
			}
		}

		switch update.Message.Command() {
		case "help":
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, help))
		case "listConfig":
			ls := bt.cfgRepo.FindAll(ctx)
			if text, err := json.Marshal(ls); err == nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, string(text)))
			}
		}

		chatId := update.Message.Chat.ID

		// if there is no current state
		if us.flow == nil {
			switch update.Message.Command() {
			case "schedule":
				cfgList := bt.cfgRepo.FindAll(ctx)

				if len(cfgList) == 0 {
					bot.Send(tgbotapi.NewMessage(chatId, "У вас еще нет конфига. Пожалуйста используейте команду /addConfig"))
				} else {
					flow := NewScheduleFlow()
					us.flow = &flow
					bot.Send(tgbotapi.NewMessage(chatId, "Введите айди канала, куда необходимо запостить"))
					us.flow.step = "requested_channel"
				}
			case "addConfig":
				flow := NewConfigFlow()
				us.flow = &flow
				bot.Send(tgbotapi.NewMessage(chatId, "Введите текст. !pair будет заменено на сигнал"))
				us.flow.step = "text_requested"
			}
		} else {
			if !update.Message.IsCommand() && us.flow != nil {
				switch us.flow.kind {
				case "schedule":
					info, ok := (us.flow.info).(scheduleConfig)

					if !ok {
						log.Error("Non possible state met")
					}
					switch us.flow.step {
					case "requested_channel":
						t, err := strconv.Atoi(update.Message.Text)

						if err != nil {
							bot.Send(tgbotapi.NewMessage(chatId, "Неизвестный формат"))
							bot.Send(tgbotapi.NewMessage(chatId, "Введите айди канала, куда необходимо запостить\""))
						} else {
							info.chatId = int64(t)
							bot.Send(tgbotapi.NewMessage(chatId, "Введите время канала, куда необходимо запостить"))
							us.flow.step = "requested_time"
						}
					case "requested_time":
						if _, err := cron.Parse(update.Message.Text); err != nil {
							bot.Send(tgbotapi.NewMessage(chatId, "Неизвестный формат: "+err.Error()))
						} else {
							info.tm = update.Message.Text
							bot.Send(tgbotapi.NewMessage(chatId, "Введите номер конфига"))
							us.flow.step = "requested_config"
						}
					case "requested_config":
						t, err := strconv.Atoi(update.Message.Text)

						if err != nil {
							bot.Send(tgbotapi.NewMessage(chatId, "Неизвестный формат"))
							bot.Send(tgbotapi.NewMessage(chatId, "Введите номер конфига"))
						} else {

							cfg := bt.cfgRepo.Find(ctx, message_flow_config.ConfigId(t))

							if cfg == nil {
								bot.Send(tgbotapi.NewMessage(chatId, "Нет такого конфига. \nВведите другой номер"))
								break
							}

							info.configId = int64(t)
							bot.Send(tgbotapi.NewMessage(chatId, "Подтвердите /done, если все верно:"))
							us.flow.step = "to_confirm"
						}
					}
					us.flow.info = info
				case "config":
					cmds, ok := us.flow.info.([]message_flow_config.Command)
					if !ok {
						log.Error("Non possible state met")
					}

					switch us.flow.step {
					case "text_requested":
						cmd := message_flow_config.Command{
							Text: update.Message.Text,
						}
						cmds = append(cmds, cmd)
						us.flow.info = cmds
						bot.Send(tgbotapi.NewMessage(chatId, "Введите время канала"))
						us.flow.step = "time_requested"
					case "time_requested":
						if t, err := strconv.Atoi(update.Message.Text); err != nil {
							bot.Send(tgbotapi.NewMessage(chatId, "Неверный формат. Введите число"))
						} else {
							cmd := &cmds[len(cmds)-1]
							cmd.TimeAfter = int64(t)
							bot.Send(tgbotapi.NewMessage(chatId, "Введите текст канала"))
							us.flow.step = "text_requested"
						}
					}
				}
			} else {
				switch update.Message.Command() {
				case "cancel":
					us.flow = nil
					bot.Send(tgbotapi.NewMessage(chatId, "Отмена"))
				case "done":
					bt.executeFlow(ctx, chatId, us.flow)
					us.flow = nil
				default:
					bot.Send(tgbotapi.NewMessage(chatId, "Неизвестная команда"))
				}
			}
		}

		bt.repo.Save(ctx, *us)
	}
}

func (bt botService) executeFlow(ctx context.Context, chatId int64, flow *Flow) {
	switch flow.kind {
	case "schedule":
		scheduleConfig := flow.info.(scheduleConfig)

		cfg := bt.cfgRepo.Find(ctx, message_flow_config.ConfigId(scheduleConfig.configId))

		if cfg == nil {
			log.Error("Something gone wrong. Such config doesnt exist: ", scheduleConfig)
			bt.tgApi.Send(tgbotapi.NewMessage(chatId, "Что-то пошло не так"))
		} else {
			bt.scheduleExecution(ctx, cfg.Cmds, scheduleConfig)
			bt.tgApi.Send(tgbotapi.NewMessage(chatId, "Запустили выполнение конфига"))
		}
	case "config":
		cmds := flow.info.([]message_flow_config.Command)
		cfg := message_flow_config.MessageConfig{
			Id:   1,
			Cmds: cmds,
		}
		bt.cfgRepo.Save(ctx, cfg)
		bt.tgApi.Send(tgbotapi.NewMessage(chatId, "Конфиг сохранен"))
	}
}

func (bt botService) scheduleExecution(ctx context.Context, cmds []message_flow_config.Command, info scheduleConfig) {
	log.Debug("Scheduling execution of config", cmds, "for: ", info.tm)

	cron := cron.New()
	cron.AddFunc(info.tm, func() {
		for i, cmd := range cmds {
			txt := cmd.Text
			if strings.Contains(cmd.Text, "!pair") {
				value := bt.binomoService.GetLastValue()
				rand.Shuffle(len(value), func(i, j int) { value[i], value[j] = value[j], value[i] })

				txt = strings.Replace(txt, "!pair", fmt.Sprintf("%s %s", value[0].PairName, value[0].Type), 1)
			}
			bt.tgApi.Send(tgbotapi.NewMessage(info.chatId, txt))
			if i < len(cmds)-1 {
				nextCmd := cmds[i+1]

				log.Debug("Sleeping for: ", nextCmd.TimeAfter)
				time.Sleep(time.Duration(nextCmd.TimeAfter) * time.Minute)
			}
		}
	})

	cron.Start()

}
