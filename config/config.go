package config

type Simullink struct {
	Email        EmailConfig `yaml:"email"`
	CityCode     string      `yaml:"city_code"`
	URL          string      `yaml:"url"`
	TagsSelected []string    `yaml:"tags_selected,omitempty"`
	DBDir        string      `yaml:"db_dir"`
	Log          Log         `yaml:"log"`
}

type Zhengzai struct {
	Email  EmailConfig `yaml:"email"`
	AdCode string      `yaml:"ad_code"`
	URL    string      `yaml:"url"`
	DBDir  string      `yaml:"db_dir"`
	Log    `yaml:"log"`
}

type EmailConfig struct {
	From     string `yaml:"from"`
	Password string `yaml:"password"`
	Server   string `yaml:"server"`
	Port     int    `yaml:"port"`
	To       string `yaml:"to"`
}

type Log struct {
	LogSuffix string `yaml:"log_suffix"`
	LogDir    string `yaml:"log_dir"`
}
