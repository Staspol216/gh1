package pvz_config

import (
	"net/url"
	"os"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AppName  string `envconfig:"APP_NAME" default:"my-app"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
	Env      string `envconfig:"ENV" default:"development"` // "development" or "production"

	// Backend
	BackendHost     string `envconfig:"BACKEND_HOST" default:"0.0.0.0"`
	BackendHTTPPort int    `envconfig:"BACKEND_HTTP_PORT" default:"8080"`
	BackendGRPCPort int    `envconfig:"BACKEND_GRPC_PORT" default:"50051"`

	// Database (Postgres)
	DBHost    string `envconfig:"DB_HOST" required:"true"`
	DBPort    int    `envconfig:"DB_PORT" default:"5432"`
	DBUser    string `envconfig:"DB_USER" required:"true"`
	DBPass    string `envconfig:"DB_PASSWORD" required:"true"`
	DBName    string `envconfig:"DB_NAME" required:"true"`
	DBSSLMode string `envconfig:"DB_SSLMODE" default:"disable"`

	// Redis
	RedisHost        string `envconfig:"REDIS_HOST" required:"true"`
	RedisPort        int    `envconfig:"REDIS_PORT" default:"6379"`
	RedisInsightPort int    `envconfig:"REDIS_INSIGHT_PORT" default:"5540"`

	// Kafka
	KafkaHost string `envconfig:"KAFKA_HOST" required:"true"`
	KafkaPort int    `envconfig:"KAFKA_PORT" default:"9092"`
}

// Load loads env variables into a Config struct.
func Load() (*Config, error) {
	// Only load .env.local in development mode
	if os.Getenv("ENV") == "" || os.Getenv("ENV") == "development" {
		_ = godotenv.Load(".env.local")
	}

	// Load .env as fallback
	_ = godotenv.Load(".env")

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	spew.Dump(cfg)

	return &cfg, nil
}

func (c *Config) DBConnString() string {

	u := &url.URL{
		Scheme: "postgres",
		Host:   c.DBHost + ":" + strconv.Itoa(c.DBPort),
		Path:   c.DBName,
	}

	// Safely set username & password
	u.User = url.UserPassword(c.DBUser, c.DBPass)

	// Add query params (like sslmode)
	q := u.Query()
	q.Set("sslmode", c.DBSSLMode)
	u.RawQuery = q.Encode()

	spew.Dump(u.String())

	return u.String()
}

// Redis address
func (c *Config) RedisAddr() string {
	return c.RedisHost + ":" + strconv.Itoa(c.RedisPort)
}

// Kafka broker address
func (c *Config) KafkaAddr() string {
	return c.KafkaHost + ":" + strconv.Itoa(c.KafkaPort)
}

// HTTP bind address
func (c *Config) HTTPAddr() string {
	return c.BackendHost + ":" + strconv.Itoa(c.BackendHTTPPort)
}

// gRPC bind address
func (c *Config) GRPCAddr() string {
	return c.BackendHost + ":" + strconv.Itoa(c.BackendGRPCPort)
}
