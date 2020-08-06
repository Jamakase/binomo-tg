package bot

import "github.com/awesomeProject/internal/app/bot/message_flow_config"

type Flow struct {
	kind string
	step string
	info interface{}
}

type scheduleConfig struct {
	chatId   int64
	configId int64
	tm       string
}

func NewScheduleFlow() Flow {
	return Flow{
		kind: "schedule",
		step: "initial",
		info: scheduleConfig{},
	}
}

func NewConfigFlow() Flow {
	return Flow{
		kind: "config",
		step: "initial",
		info: []message_flow_config.Command{},
	}
}
