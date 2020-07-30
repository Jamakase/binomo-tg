package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

type baseCommand struct {
	msg    tgbotapi.Message
	config Config
}

type command interface {
	Execute() tgbotapi.Chattable
}

type helpCommand struct {
	baseCommand
}

func (h helpCommand) Execute() tgbotapi.Chattable {
	return tgbotapi.NewMessage(h.msg.Chat.ID, "I know next commands:")
}

type testCommand struct {
	baseCommand
}

func (t testCommand) Execute() tgbotapi.Chattable {
	return nil
}

func processCommand(cfg Config, msg *tgbotapi.Message) command {
	var cmd command

	baseCmd := baseCommand{
		*msg,
		cfg,
	}

	switch msg.Command() {
	case "help":
		cmd = helpCommand{
			baseCmd,
		}
	//case "status":
	//	msg.Text = "I'm ok."
	case "test":
		cmd = testCommand{
			baseCmd,
		}
	}

	return cmd
}
