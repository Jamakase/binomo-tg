package bot

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUserStateRepo_Save(t *testing.T) {
	t.Run("Saves new object", func(t *testing.T) {
		storage := make(map[chatId]userState)
		b := userStateRepo{
			storage: storage,
		}

		flow := Flow{
			kind: "schedule",
			step: "initial",
		}

		userState := userState{
			id:   chatId(1),
			flow: &flow,
		}

		b.Save(nil, userState)

		if val, ok := storage[1]; assert.True(t, ok, "state should be saved") {
			assert.Equal(t, val, userState, "states should be equal")
		}
	})

	t.Run("Overrides old object", func(t *testing.T) {
		storage := make(map[chatId]userState)

		flow := Flow{
			kind: "schedule",
			step: "initial",
		}

		userState := userState{
			id:   chatId(1),
			flow: &flow,
		}

		storage[1] = userState

		b := userStateRepo{
			storage: storage,
		}

		userState.flow.step = "second"

		b.Save(nil, userState)

		if val, ok := storage[1]; assert.True(t, ok, "state should be saved") {
			assert.Equal(t, "second", val.flow.step, "states should be equal")
		}
	})
}

func TestUserStateRepo_FindUserState(t *testing.T) {
	storage := make(map[chatId]userState)

	flow := Flow{
		kind: "schedule",
		step: "initial",
	}

	userState := userState{
		id:   chatId(1),
		flow: &flow,
	}

	storage[1] = userState

	b := userStateRepo{
		storage: storage,
	}

	if state := b.FindUserState(nil, 1); assert.NotNil(t, state, "no state found") {
		assert.Equal(t, userState, *state, "states should be equal")
	}
}
