package job

import (
	"context"
	"github.com/robfig/cron"
)

type CronStore interface {
	Add(ctx context.Context, chatId int64, cron *cron.Cron)
	List(ctx context.Context) []int64
	Remove(ctx context.Context, channelId int64) bool
}

type cronStore struct {
	storage map[int64]*cron.Cron
}

func (c cronStore) Add(ctx context.Context, channelId int64, cron *cron.Cron) {
	c.storage[channelId] = cron
}

func (c cronStore) Remove(ctx context.Context, channelId int64) bool {
	if cr, ok := c.storage[channelId]; ok {
		cr.Stop()
		delete(c.storage, channelId)
		return true
	}

	return false
}

func (c cronStore) List(ctx context.Context) []int64 {
	list := make([]int64, 0, len(c.storage))
	for chId := range c.storage {
		list = append(list, chId)
	}
	return list
}

func NewStore() CronStore {
	return cronStore{
		storage: make(map[int64]*cron.Cron),
	}
}
