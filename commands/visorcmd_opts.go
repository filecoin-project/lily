package commands

type VisorCmdOpts struct {
	Name string
	Api  string

	Lens          string
	LensCacheHint int

	Repo   string
	RepoRO bool

	DB                string
	DBPoolSize        int
	DBAllowUpsert     bool
	DBAllowMigrations bool

	LogLevel      string
	LogLevelNamed string

	Tracing            bool
	JaegerHost         string
	JaegerPort         int
	JaegerName         string
	JaegerSampleType   string
	JaegerSamplerParam float64

	PrometheusPort string
}

var VisorCmdFlags VisorCmdOpts
