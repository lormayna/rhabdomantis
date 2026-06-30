package cmd

type Config struct {
	ShodanAPIKey string `env:"SHODAN_API_KEY" envDefault:""`
	DBFile       string `env:"DB_FILE" envDefault:"hosts.db"`
	Workers      int    `env:"WORKERS" envDefault:"3"`
}
