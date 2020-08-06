package bot

import (
	"context"
)

type chatId int64

type userState struct {
	id   chatId
	flow *Flow
}

type UserStateRepo interface {
	FindUserState(ctx context.Context, id chatId) *userState
	Save(ctx context.Context, userState userState)
}

type userStateRepo struct {
	storage map[chatId]userState
}

func (b userStateRepo) FindUserState(ctx context.Context, id chatId) *userState {
	if us, ok := b.storage[id]; ok {
		return &us
	} else {
		return nil
	}
}

func (b userStateRepo) Save(ctx context.Context, userState userState) {
	b.storage[userState.id] = userState
}

func NewRepo() UserStateRepo {
	storage := map[chatId]userState{}

	return userStateRepo{
		storage,
	}
}
