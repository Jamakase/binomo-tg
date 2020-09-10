package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/awesomeProject/internal/app/binomo"
	"github.com/awesomeProject/internal/app/bot/job"
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

/listConfig: выведет все текуще существующие конфиги
/addConfig: начнет процесс создания нового конфига
/schedule: Начинает процесс выставления таймера в канал

/listJobs: выведет все текущие таймеры
/stopJob: поможет остановить выставленный таймер

во время флоу я знаю следующие команды:
/cancel: остановит процедуру создания конфига
/done: завершит текущий флоу ( создания конфига или выставление таймера )
`
)

type BotService interface {
	Run(ctx context.Context)
}

type botService struct {
	config        Config
	logger        Logger
	tgApi         *tgbotapi.BotAPI
	binomoService binomo.Service

	repo      UserStateRepo
	cronStore job.CronStore
	cfgRepo   Repo
}

func New(logger Logger, config Config, service binomo.Service, repo UserStateRepo, cronStore job.CronStore, cfgRepo Repo) BotService {
	bot, err := tgbotapi.NewBotAPI(config.Token)

	if err != nil {
		log.Panic(err)
	}

	logger.Info("Authorized on account", log.Fields{"account": bot.Self.UserName})

	return botService{
		logger:        logger,
		binomoService: service,
		tgApi:         bot,
		config:        config,
		repo:          repo,
		cfgRepo:       cfgRepo,
		cronStore:     cronStore,
	}
}

func (bt botService) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bt.tgApi.GetUpdatesChan(u)

	for update := range updates {
		go bt.processMessage(ctx, update)
	}
}

func (bt botService) processMessage(ctx context.Context, update tgbotapi.Update) {
	if update.Message == nil { // ignore any non-Message updates
		return
	}

	// TODO: add more guys to config
	//if len(bt.config.Whitelist) > 0 {
	//	contains := false
	//	for _, id := range bt.config.Whitelist {
	//		if update.Message.Chat.ID == id {
	//			contains = true
	//		}
	//	}
	//	if !contains {
	//		continue
	//	}
	//}

	us := bt.repo.FindUserState(ctx, chatId(update.Message.Chat.ID))

	if us == nil {
		us = &userState{
			id: chatId(update.Message.Chat.ID),
		}
	}

	chatId := update.Message.Chat.ID

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "help":
			bt.send(update.Message.Chat.ID, help)
		case "listConfig":
			if ls, err := bt.cfgRepo.List(ctx); err == nil {
				if text, err := json.MarshalIndent(ls, "", " "); err == nil {
					bt.send(update.Message.Chat.ID, string(text))
				}
			} else {
				bt.logger.Error(err.Error())
				bt.send(update.Message.Chat.ID, "Не могу получить конфиг. Что-то не так")
			}
		case "listJobs":
			ls := bt.cronStore.List(ctx)
			if parsedList, err := json.MarshalIndent(ls, "", " "); err == nil {
				bt.send(update.Message.Chat.ID, string(parsedList))
			}
		case "schedule":
			if us.flow == nil {
				cfgList, _ := bt.cfgRepo.List(ctx)

				if len(cfgList) == 0 {
					bt.send(chatId, "У вас еще нет конфига. Пожалуйста используейте команду /addConfig")
				} else {
					flow := NewScheduleFlow()
					us.flow = &flow
					bt.send(chatId, "Введите айди канала, куда необходимо запостить")
					us.flow.step = "requested_channel"
				}
			}
		case "addConfig":
			if us.flow == nil {
				flow := NewConfigFlow()
				us.flow = &flow
				bt.send(chatId, "Введите на что необходимо заменить пару в при 1")
				us.flow.step = "grow_text_requested"
			}
		case "stopJob":
			if us.flow == nil {
				jobs := bt.cronStore.List(ctx)
				if len(jobs) == 0 {
					bt.send(chatId, "Нечего останавливать.")
				} else {
					if parsedList, err := json.Marshal(jobs); err == nil {
						bt.send(update.Message.Chat.ID, "Выберите какой надо остановить:\n"+string(parsedList))
						flow := NewStopJob()
						us.flow = &flow
					}
				}
			}
		case "cancel":
			if us.flow != nil {
				us.flow = nil
				bt.send(chatId, "Отмена")
			}
		case "done":
			if us.flow != nil {
				bt.executeFlow(ctx, chatId, us.flow)
				us.flow = nil
			}
		default:
			bt.send(chatId, "Неизвестная команда")
		}
	} else if us.flow != nil {
		// if there is something in process
		switch us.flow.kind {

		//schedule flow
		case Schedule:
			info, ok := (us.flow.info).(scheduleConfig)

			if !ok {
				bt.logger.Error("Non possible state met")
			}
			switch us.flow.step {
			case "requested_channel":
				if t, err := strconv.Atoi(update.Message.Text); err != nil {
					bt.send(chatId, "Неизвестный формат\nВведите айди канала, куда необходимо запостить")
				} else {
					info.chatId = int64(t)
					us.flow.step = "requested_time"
					bt.send(chatId, "Введите в какое время вы хотите запостить. Это может помочь https://www.freeformatter.com/cron-expression-generator-quartz.html")
				}
			case "requested_time":
				if _, err := cron.Parse(update.Message.Text); err != nil {
					bt.send(chatId, "Неизвестный формат: "+err.Error()+"\n Попробуйте https://www.freeformatter.com/cron-expression-generator-quartz.html кроме последнего символа ( только 6 значений, не ввода год )")
				} else {
					info.tm = update.Message.Text
					us.flow.step = "requested_config"
					bt.send(chatId, "Введите номер конфига")
				}
			case "requested_config":
				id := ConfigId(update.Message.Text)
				cfg, err := bt.cfgRepo.Get(ctx, id)

				if err != nil {
					bt.logger.Error(err.Error())
					bt.send(chatId, "Что-то пошло не так. Введите номер")
				} else if cfg == nil {
					bt.send(chatId, "Нет такого конфига. \nВведите другой номер")
				} else {
					info.configId = id
					bt.send(chatId, "Подтвердите /done, если все верно:")
					us.flow.step = "to_confirm"
				}
			}
			us.flow.info = info
		case StopJob:
			if jobId, err := strconv.Atoi(update.Message.Text); err != nil {
				bt.send(chatId, "Неизвестный формат\nВведите айди канала, куда необходимо запостить")
			} else {
				bt.cronStore.Remove(ctx, int64(jobId))
				us.flow = nil
				bt.send(chatId, "Синк остановлен")
				bt.logger.Info("Stopped config", log.Fields{"jobId": jobId})
			}
			// config flow
		case Configuration:
			cfg, ok := us.flow.info.(SpecConfig)
			if !ok {
				bt.logger.Error("Not possible state met")
			}

			commands := cfg.Commands

			switch us.flow.step {
			case "grow_text_requested":
				cfg.UpText = update.Message.Text
				bt.send(chatId, "Введите на что необходимо заменить пару при 0")
				us.flow.step = "low_text_requested"
			case "low_text_requested":
				cfg.LowText = update.Message.Text
				bt.send(chatId, "Введите текст. !pair будет заменено на сигнал")
				us.flow.step = "text_requested"

			case "text_requested":
				cmd := Command{
					Text: update.Message.Text,
				}
				commands = append(commands, cmd)
				if len(commands) == 1 {
					bt.send(chatId, "Введите текст следующего события.")
				} else {
					bt.send(chatId, "Введите через сколько минут от прошлого, вы бы хотели запостить в канал.")
					us.flow.step = "time_requested"
				}

			case "time_requested":
				if t, err := strconv.Atoi(update.Message.Text); err != nil {
					bt.send(chatId, "Неверный формат. Введите число")
				} else {
					cmd := &commands[len(commands)-1]
					cmd.TimeAfter = int64(t)
					bt.send(chatId, "Введите текст следующего события")
					us.flow.step = "text_requested"
				}
			}
			cfg.Commands = commands
			us.flow.info = cfg
		}
	}

	bt.repo.Save(ctx, *us)
}

func (bt botService) executeFlow(ctx context.Context, chatId int64, flow *Flow) {
	switch flow.kind {
	case Schedule:
		scheduleConfig := flow.info.(scheduleConfig)

		cfg, _ := bt.cfgRepo.Get(ctx, scheduleConfig.configId)

		if cfg == nil {
			bt.logger.Error("Something gone wrong. Such config doesnt exist", log.Fields{"cfgId": scheduleConfig.configId})
			bt.send(chatId, "Что-то пошло не так")
		} else {
			if err := bt.scheduleExecution(ctx, cfg.SpecConfig, scheduleConfig); err == nil {
				bt.send(chatId, "Запустили выполнение конфига")
			} else {
				bt.logger.Error("Not able to run config", log.Fields{"err": err})
				bt.send(chatId, "Что-то пошло не так")
			}
		}
	case Configuration:
		spec, ok := flow.info.(SpecConfig)

		if !ok {
			bt.logger.Error("Unable to cast expression to spec config")
		}
		cfg := MessageConfig{
			SpecConfig: spec,
		}
		if err := bt.cfgRepo.Save(ctx, &cfg); err != nil {
			bt.logger.Error(err.Error())
			bt.send(chatId, "Не получилось сохранить конфиг")
		} else {
			bt.send(chatId, "Конфиг сохранен")
		}
	}
}

func (bt botService) send(chatID int64, text string) (tgbotapi.Message, error) {
	msg, err := bt.tgApi.Send(tgbotapi.NewMessage(chatID, text))
	if err != nil {
		bt.logger.Error(err.Error())
	}

	return msg, err
}

func (bt botService) scheduleExecution(ctx context.Context, cfg SpecConfig, info scheduleConfig) error {
	cmds := cfg.Commands
	bt.logger.Info("Scheduling execution of config", log.Fields{
		"cmd": cmds,
		"tm":  info.tm,
	})

	timer := cron.NewWithLocation(time.UTC)
	if err := timer.AddFunc(info.tm, func() {
		for i, cmd := range cmds {
			txt := cmd.Text
			if strings.Contains(cmd.Text, "!pair") {
				value := bt.binomoService.GetLastValue()
				rand.Shuffle(len(value), func(i, j int) { value[i], value[j] = value[j], value[i] })

				rndValue := value[0]

				var pairText string

				if rndValue.Type == "1" {
					pairText = cfg.UpText
				} else {
					pairText = cfg.LowText
				}

				bt.logger.Info("New value for pair", log.Fields{"pair": rndValue.PairName, "value": rndValue.Type})

				txt = strings.Replace(txt, "!pair", fmt.Sprintf("%s %s", rndValue.PairName, pairText), 1)
			}
			if _, sendErr := bt.tgApi.Send(tgbotapi.NewMessage(info.chatId, txt)); sendErr != nil {
				bt.logger.Error("Unable to send message", log.Fields{"err": sendErr})
			}
			if i < len(cmds)-1 {
				nextCmd := cmds[i+1]

				bt.logger.Debug("Sleeping for", log.Fields{"time": nextCmd.TimeAfter})
				time.Sleep(time.Duration(nextCmd.TimeAfter) * time.Minute)
			}
		}
	}); err != nil {
		return err
	}

	bt.cronStore.Add(ctx, info.chatId, timer)

	timer.Start()

	bt.logger.Info("Scheduled timer for channel: ", log.Fields{"chatId": info.chatId})

	return nil
}
