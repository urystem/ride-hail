package pkg

import (
	"fmt"
	"os"

	"github.com/drone/envsubst"
	"github.com/subosito/gotenv"
	"go.yaml.in/yaml/v4"
)

type Config struct {
	DatabaseCfg  `yaml:"database" json:"database"`
	RabbitMQCfg  `yaml:"rabbitmq" json:"rabbitmq"`
	WebSocketCfg `yaml:"websocket" json:"websocket"`
	ServicesCfg  `yaml:"services" json:"services"`
}

type DatabaseCfg struct {
	Host     string `yaml:"host" json:"host"`
	Port     uint16 `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
	Database string `yaml:"database" json:"database"`
}

type RabbitMQCfg struct {
	Host     string `yaml:"host" json:"host"`
	Port     uint16 `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
}

type WebSocketCfg struct {
	Port uint16 `yaml:"port" json:"port"`
}

type ServicesCfg struct {
	Secret                string 
	RideService           uint16 `yaml:"ride_service" json:"ride_service"`
	DriverLocationService uint16 `yaml:"driver_location_service" json:"driver_location_service"`
	AdminService          uint16 `yaml:"admin_service" json:"admin_service"`
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
