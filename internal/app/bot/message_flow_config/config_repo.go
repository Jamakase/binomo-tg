package message_flow_config

import (
	"context"
)

type ConfigId int64

type Command struct {
	Text      string
	TimeAfter int64
}

type MessageConfig struct {
	Id   ConfigId
	Cmds []Command
}

type Repo interface {
	Find(ctx context.Context, id ConfigId) *MessageConfig
	Save(ctx context.Context, userState MessageConfig)
	FindAll(ctx context.Context) []MessageConfig
}

type messageConfigRepo struct {
	storage map[ConfigId]MessageConfig
}

func (b messageConfigRepo) Find(ctx context.Context, id ConfigId) *MessageConfig {
	if us, ok := b.storage[id]; ok {
		return &us
	} else {
		return nil
	}
}

func (b messageConfigRepo) Save(ctx context.Context, messageConfig MessageConfig) {
	b.storage[messageConfig.Id] = messageConfig
}

func (b messageConfigRepo) FindAll(ctx context.Context) []MessageConfig {
	list := make([]MessageConfig, 0, len(b.storage))
	for _, cfg := range b.storage {
		list = append(list, cfg)
	}
	return list
}

func NewRepo() Repo {
	storage := map[ConfigId]MessageConfig{}

	return messageConfigRepo{
		storage,
	}
}
