package pkg

import (
	"fmt"
	"os"

	"github.com/drone/envsubst"
	"github.com/subosito/gotenv"
	"go.yaml.in/yaml/v4"
)

type Config struct {
	DatabaseCfg  `yaml:"database"`
	RabbitMQCfg  `yaml:"rabbitmq"`
	WebSocketCfg `yaml:"websocket"`
	ServicesCfg  `yaml:"services"`
}

type DatabaseCfg struct {
	Host     string `yaml:"host"`
	Port     uint16 `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type RabbitMQCfg struct {
	Host     string `yaml:"host"`
	Port     uint16 `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type WebSocketCfg struct {
	Port uint16 `yaml:"port"`
}

type ServicesCfg struct {
	Secret                string
	RideService           uint16 `yaml:"ride_service"`
	DriverLocationService uint16 `yaml:"driver_location_service"`
	AdminService          uint16 `yaml:"admin_service"`
}

func ParseConfig() (*Config, error) {
	err := gotenv.Load()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile("config.yml")
	if err != nil {
		return nil, err
	}

	// Подставляем переменные окружения + дефолты через :-
	replaced, err := envsubst.EvalEnv(string(data))
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	err = yaml.Unmarshal([]byte(replaced), cfg)
	if err != nil {
		return nil, err
	}
	cfg.ServicesCfg.Secret = os.Getenv("MY_SECRET")
	fmt.Println(cfg.ServicesCfg.Secret)
	return cfg, nil
}
