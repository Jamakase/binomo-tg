package bot

type Kind string

const (
	Schedule      Kind = "schedule"
	Configuration      = "config"
	StopJob            = "stop-job"
)

type Flow struct {
	kind Kind
	step string
	info interface{}
}

type scheduleConfig struct {
	chatId   int64
	configId ConfigId
	tm       string
}

type SpecConfig struct {
	Commands []Command
	UpText   string
	LowText  string
}

func NewScheduleFlow() Flow {
	return Flow{
		kind: Schedule,
		step: "initial",
		info: scheduleConfig{},
	}
}

func NewConfigFlow() Flow {
	return Flow{
		kind: Configuration,
		step: "initial",
		info: SpecConfig{},
	}
}

func NewStopJob() Flow {
	return Flow{
		kind: StopJob,
		step: "initial",
	}
}
