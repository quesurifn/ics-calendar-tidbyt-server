package config

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"time"
)

type Config struct {
	*Settings
	cfgModTimes map[string]time.Time
}

type Settings struct {
	Environment        string
	ENVPrefix          string
	Debug              bool
	Verbose            bool
	Silent             bool
	AutoReload         bool
	AutoReloadInterval time.Duration
	AutoReloadCallback func(cfg interface{})

	// In case of json files, this field will be used only when compiled with
	// go 1.10 or later.
	// This field will be ignored when compiled with go versions lower than 1.10.
	ErrorOnUnmatchedKeys bool
}

// New initialize a Config
func New(cfg *Settings) *Config {
	if cfg == nil {
		cfg = &Settings{}
	}

	if os.Getenv("CONFIG_DEBUG_MODE") != "" {
		cfg.Debug = true
	}

	if os.Getenv("CONFIG_VERBOSE_MODE") != "" {
		cfg.Verbose = true
	}

	if cfg.AutoReload && cfg.AutoReloadInterval == 0 {
		cfg.AutoReloadInterval = time.Second
	}

	return &Config{Settings: cfg}
}

var testRegexp = regexp.MustCompile(`_test|(\.test$)`)

// GetEnvironment get environment
func (c *Config) GetEnvironment() string {
	if c.Environment == "" {
		if env := os.Getenv("CONFIG_ENV"); env != "" {
			return env
		}

		if testRegexp.MatchString(os.Args[0]) {
			return "test"
		}

		return "development"
	}
	return c.Environment
}

// GetErrorOnUnmatchedKeys returns a boolean indicating if an error should be
// thrown if there are keys in the cfg file that do not correspond to the
// cfg struct
func (c *Config) GetErrorOnUnmatchedKeys() bool {
	return c.ErrorOnUnmatchedKeys
}

// Load will unmarshal configurations to struct from files that you provide
func (c *Config) Load(cfg interface{}, files ...string) (err error) {
	defaultValue := reflect.Indirect(reflect.ValueOf(cfg))
	if !defaultValue.CanAddr() {
		return fmt.Errorf("Config %v should be addressable", cfg)
	}
	_, err = c.load(cfg, false, files...)

	if c.AutoReload {
		go func() {
			timer := time.NewTimer(c.AutoReloadInterval)
			for range timer.C {
				reflectPtr := reflect.New(reflect.ValueOf(cfg).Elem().Type())
				reflectPtr.Elem().Set(defaultValue)

				var changed bool
				if changed, err = c.load(reflectPtr.Interface(), true, files...); err == nil && changed {
					reflect.ValueOf(cfg).Elem().Set(reflectPtr.Elem())
					if c.AutoReloadCallback != nil {
						c.AutoReloadCallback(cfg)
					}
				} else if err != nil {
					fmt.Printf("Failed to reload configuration from %v, got error %v\n", files, err)
				}
				timer.Reset(c.AutoReloadInterval)
			}
		}()
	}
	return
}

// ENV return environment
func ENV() string {
	return New(nil).GetEnvironment()
}

// Load will unmarshal configurations to struct from files that you provide
func Load(cfg interface{}, files ...string) (*Config, error) {
	c := New(nil)
	if err := c.Load(cfg, files...); err != nil {
		return nil, err
	}

	return c, nil
}
