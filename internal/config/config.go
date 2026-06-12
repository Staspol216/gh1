package pvz_config

import (
	"net/url"
	"os"
	"strconv"

	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
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

	// Prometheus
	PrometheusHost string `envconfig:"PROMETHEUS_HOST" default:"localhost"`
	PrometheusPath string `envconfig:"PROMETHEUS_PATH" default:"/metrics"`
	PrometheusPort int    `envconfig:"PROMETHEUS_PORT" default:"9090"`

	// Jaeger
	JaegerHost          string `envconfig:"JAEGER_HOST" default:"localhost"`
	JaegerCollectorPort int    `envconfig:"JAEGER_COLLECTOR_PORT" default:"14268"`
	JaegerUIPort        int    `envconfig:"JAEGER_UI_PORT" default:"16686"`
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

	app_logger.MyLogger.Info("config loaded",
		zap.String("app_name", cfg.AppName),
		zap.String("env", cfg.Env),
		zap.String("log_level", cfg.LogLevel),
		zap.String("backend_host", cfg.BackendHost),
		zap.Int("backend_http_port", cfg.BackendHTTPPort),
		zap.Int("backend_grpc_port", cfg.BackendGRPCPort),
		zap.String("db_host", cfg.DBHost),
		zap.Int("db_port", cfg.DBPort),
		zap.String("db_name", cfg.DBName),
		zap.String("redis_host", cfg.RedisHost),
		zap.Int("redis_port", cfg.RedisPort),
		zap.String("kafka_host", cfg.KafkaHost),
		zap.Int("kafka_port", cfg.KafkaPort),
		zap.String("jaeger_host", cfg.JaegerHost),
		zap.Int("jaeger_collector_port", cfg.JaegerCollectorPort),
		zap.Int("jaeger_ui_port", cfg.JaegerUIPort),
	)

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

	return u.String()
}

func (c *Config) RedisAddr() string {
	return c.RedisHost + ":" + strconv.Itoa(c.RedisPort)
}

func (c *Config) KafkaAddr() string {
	return c.KafkaHost + ":" + strconv.Itoa(c.KafkaPort)
}

func (c *Config) PrometheusAddr() string {
	return c.PrometheusHost + ":" + strconv.Itoa(c.PrometheusPort)
}

func (c *Config) JaegerCollectorEndpoint() string {
	return "http://" + c.JaegerHost + ":" + strconv.Itoa(c.JaegerCollectorPort) + "/api/traces"
}

func (c *Config) HTTPAddr() string {
	return c.BackendHost + ":" + strconv.Itoa(c.BackendHTTPPort)
}

func (c *Config) GRPCAddr() string {
	return c.BackendHost + ":" + strconv.Itoa(c.BackendGRPCPort)
}
