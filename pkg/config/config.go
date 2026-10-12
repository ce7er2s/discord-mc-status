package config

import "time"

type Config struct {
	Address                         string
	Port                            int
	UpdateInterval, TimeoutInterval time.Duration
}
