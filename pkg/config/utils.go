package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
)

// UnmatchedTomlKeysError errors are returned by the Load function when
// ErrorOnUnmatchedKeys is set to true and there are unmatched keys in the input
// toml cfg file. The string returned by Error() contains the names of the
// missing keys.
type UnmatchedTomlKeysError struct {
	Keys []toml.Key
}

func (e *UnmatchedTomlKeysError) Error() string {
	return fmt.Sprintf("There are keys in the config file that do not match any field in the given struct: %v", e.Keys)
}

func getConfigurationFileWithENVPrefix(file, env string) (string, time.Time, error) {
	var (
		envFile string
		extname = path.Ext(file)
	)

	if extname == "" {
		envFile = fmt.Sprintf("%v.%v", file, env)
	} else {
		envFile = fmt.Sprintf("%v.%v%v", strings.TrimSuffix(file, extname), env, extname)
	}

	if fileInfo, err := os.Stat(envFile); err == nil && fileInfo.Mode().IsRegular() {
		return envFile, fileInfo.ModTime(), nil
	}
	return "", time.Now(), fmt.Errorf("failed to find file %v", file)
}

func (c *Config) getENVPrefix(cfg interface{}) string {
	if c.Settings.ENVPrefix == "" {
		if prefix := os.Getenv("CONFIG_ENV_PREFIX"); prefix != "" {
			return prefix
		}
		return "CONFIG"
	}
	return c.Settings.ENVPrefix
}

func (c *Config) getConfigurationFiles(watchMode bool, files ...string) ([]string, map[string]time.Time) {
	var resultKeys []string
	var results = map[string]time.Time{}

	if !watchMode && (c.Settings.Debug || c.Settings.Verbose) {
		fmt.Printf("Current environment: '%v'\n", c.GetEnvironment())
	}

	for i := len(files) - 1; i >= 0; i-- {
		foundFile := false
		file := files[i]

		// check configuration
		if fileInfo, err := os.Stat(file); err == nil && fileInfo.Mode().IsRegular() {
			foundFile = true
			resultKeys = append(resultKeys, file)
			results[file] = fileInfo.ModTime()
		}

		// check configuration with env
		if file, modTime, err := getConfigurationFileWithENVPrefix(file, c.GetEnvironment()); err == nil {
			foundFile = true
			resultKeys = append(resultKeys, file)
			results[file] = modTime
		}

		// check example configuration
		if !foundFile {
			if example, modTime, err := getConfigurationFileWithENVPrefix(file, "example"); err == nil {
				if !watchMode && c.Verbose {
					fmt.Printf("Failed to find configuration %v, using example file %v\n", file, example)
				}
				resultKeys = append(resultKeys, example)
				results[example] = modTime
			} else if c.Verbose {
				fmt.Printf("Failed to find configuration %v\n", file)
			}
		}
	}
	return resultKeys, results
}

func processFile(cfg interface{}, file string, errorOnUnmatchedKeys bool) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	switch {
	case strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml"):
		if errorOnUnmatchedKeys {
			return yaml.UnmarshalStrict(data, cfg)
		}
		return yaml.Unmarshal(data, cfg)
	case strings.HasSuffix(file, ".toml"):
		return unmarshalToml(data, cfg, errorOnUnmatchedKeys)
	case strings.HasSuffix(file, ".json"):
		return unmarshalJSON(data, cfg, errorOnUnmatchedKeys)
	default:
		if err := unmarshalToml(data, cfg, errorOnUnmatchedKeys); err == nil {
			return nil
		} else if errUnmatchedKeys, ok := err.(*UnmatchedTomlKeysError); ok {
			return errUnmatchedKeys
		}

		if err := unmarshalJSON(data, cfg, errorOnUnmatchedKeys); err == nil {
			return nil
		} else if strings.Contains(err.Error(), "json: unknown field") {
			return err
		}

		var yamlError error
		if errorOnUnmatchedKeys {
			yamlError = yaml.UnmarshalStrict(data, cfg)
		} else {
			yamlError = yaml.Unmarshal(data, cfg)
		}

		if yamlError == nil {
			return nil
		} else if yErr, ok := yamlError.(*yaml.TypeError); ok {
			return yErr
		}

		return errors.New("failed to decode config")
	}
}

// GetStringTomlKeys returns a string array of the names of the keys that are passed in as args
func GetStringTomlKeys(list []toml.Key) []string {
	arr := make([]string, len(list))

	for index, key := range list {
		arr[index] = key.String()
	}
	return arr
}

func unmarshalToml(data []byte, cfg interface{}, errorOnUnmatchedKeys bool) error {
	metadata, err := toml.Decode(string(data), cfg)
	if err == nil && len(metadata.Undecoded()) > 0 && errorOnUnmatchedKeys {
		return &UnmatchedTomlKeysError{Keys: metadata.Undecoded()}
	}
	return err
}

// unmarshalJSON unmarshals the given data into the cfg interface.
// If the errorOnUnmatchedKeys boolean is true, an error will be returned if there
// are keys in the data that do not match fields in the cfg interface.
func unmarshalJSON(data []byte, cfg interface{}, errorOnUnmatchedKeys bool) error {
	reader := strings.NewReader(string(data))
	decoder := json.NewDecoder(reader)

	if errorOnUnmatchedKeys {
		decoder.DisallowUnknownFields()
	}

	err := decoder.Decode(cfg)
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func getPrefixForStruct(prefixes []string, fieldStruct *reflect.StructField) []string {
	if fieldStruct.Anonymous && fieldStruct.Tag.Get("anonymous") == "true" {
		return prefixes
	}
	return append(prefixes, fieldStruct.Name)
}

func (c *Config) processDefaults(cfg interface{}) error {
	cfgValue := reflect.Indirect(reflect.ValueOf(cfg))
	if cfgValue.Kind() != reflect.Struct {
		return errors.New("invalid config, should be struct")
	}

	cfgType := cfgValue.Type()
	for i := 0; i < cfgType.NumField(); i++ {
		var (
			fieldStruct = cfgType.Field(i)
			field       = cfgValue.Field(i)
		)

		if !field.CanAddr() || !field.CanInterface() {
			continue
		}

		if isBlank := reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface()); isBlank {
			// Set default configuration if blank
			if value := fieldStruct.Tag.Get("default"); value != "" {
				if err := yaml.Unmarshal([]byte(value), field.Addr().Interface()); err != nil {
					return err
				}
			}
		}

		for field.Kind() == reflect.Ptr {
			field = field.Elem()
		}

		switch field.Kind() {
		case reflect.Struct:
			if err := c.processDefaults(field.Addr().Interface()); err != nil {
				return err
			}
		case reflect.Slice:
			for i := 0; i < field.Len(); i++ {
				if reflect.Indirect(field.Index(i)).Kind() == reflect.Struct {
					if err := c.processDefaults(field.Index(i).Addr().Interface()); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (c *Config) processTags(cfg interface{}, prefixes ...string) error {
	cfgValue := reflect.Indirect(reflect.ValueOf(cfg))
	if cfgValue.Kind() != reflect.Struct {
		return errors.New("invalid config, should be struct")
	}

	cfgType := cfgValue.Type()
	for i := 0; i < cfgType.NumField(); i++ {
		var (
			envNames    []string
			fieldStruct = cfgType.Field(i)
			field       = cfgValue.Field(i)
			envName     = fieldStruct.Tag.Get("env") // read configuration from shell env
		)

		if !field.CanAddr() || !field.CanInterface() {
			continue
		}

		if envName == "" {
			envNames = append(envNames, strings.Join(append(prefixes, fieldStruct.Name), "_"))                  // Config_DB_Name
			envNames = append(envNames, strings.ToUpper(strings.Join(append(prefixes, fieldStruct.Name), "_"))) // CONFIG_DB_NAME
		} else {
			envNames = []string{envName}
		}

		if c.Settings.Verbose {
			fmt.Printf("Trying to load field `%v` from env %v\n", fieldStruct.Name, strings.Join(envNames, ", "))
		}

		// Load From Shell ENV
		for _, env := range envNames {
			if value := os.Getenv(env); value != "" {
				if c.Settings.Debug || c.Settings.Verbose {
					fmt.Printf("Loading configuration for field `%v` from env %v...\n", fieldStruct.Name, env)
				}

				switch reflect.Indirect(field).Kind() {
				case reflect.Bool:
					switch strings.ToLower(value) {
					case "", "0", "f", "false":
						field.Set(reflect.ValueOf(false))
					default:
						field.Set(reflect.ValueOf(true))
					}
				case reflect.String:
					field.Set(reflect.ValueOf(value))
				default:
					if err := yaml.Unmarshal([]byte(value), field.Addr().Interface()); err != nil {
						return err
					}
				}
				break
			}
		}

		if isBlank := reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface()); isBlank && fieldStruct.Tag.Get("required") == "true" {
			// return error if it is required but blank
			return errors.New(fieldStruct.Name + " is required, but blank")
		}

		for field.Kind() == reflect.Ptr {
			field = field.Elem()
		}

		if field.Kind() == reflect.Struct {
			if err := c.processTags(field.Addr().Interface(), getPrefixForStruct(prefixes, &fieldStruct)...); err != nil {
				return err
			}
		}

		if field.Kind() == reflect.Slice {
			if arrLen := field.Len(); arrLen > 0 {
				for i := 0; i < arrLen; i++ {
					if reflect.Indirect(field.Index(i)).Kind() == reflect.Struct {
						if err := c.processTags(field.Index(i).Addr().Interface(), append(getPrefixForStruct(prefixes, &fieldStruct), fmt.Sprint(i))...); err != nil {
							return err
						}
					}
				}
			} else {
				defer func(field reflect.Value, fieldStruct reflect.StructField) {
					if !cfgValue.IsZero() {
						// load slice from env
						newVal := reflect.New(field.Type().Elem()).Elem()
						if newVal.Kind() == reflect.Struct {
							idx := 0
							for {
								newVal = reflect.New(field.Type().Elem()).Elem()
								if err := c.processTags(newVal.Addr().Interface(), append(getPrefixForStruct(prefixes, &fieldStruct), fmt.Sprint(idx))...); err != nil {
									return // err
								} else if reflect.DeepEqual(newVal.Interface(), reflect.New(field.Type().Elem()).Elem().Interface()) {
									break
								} else {
									idx++
									field.Set(reflect.Append(field, newVal))
								}
							}
						}
					}
				}(field, fieldStruct)
			}
		}
	}
	return nil
}

func (c *Config) load(cfg interface{}, watchMode bool, files ...string) (changed bool, err error) {
	defer func() {
		if c.Settings.Debug || c.Settings.Verbose {
			if err != nil {
				fmt.Printf("Failed to load configuration from %v, got %v\n", files, err)
			}

			fmt.Printf("Configuration:\n  %#v\n", cfg)
		}
	}()

	cfgFiles, cfgModTimeMap := c.getConfigurationFiles(watchMode, files...)

	if watchMode {
		if len(cfgModTimeMap) == len(c.cfgModTimes) {
			var changed bool
			for f, t := range cfgModTimeMap {
				if v, ok := c.cfgModTimes[f]; !ok || t.After(v) {
					changed = true
				}
			}

			if !changed {
				return false, nil
			}
		}
	}

	// process defaults
	if err := c.processDefaults(cfg); err != nil {
		return true, err
	}

	for _, file := range cfgFiles {
		if c.Settings.Debug || c.Settings.Verbose {
			fmt.Printf("Loading configurations from file '%v'...\n", file)
		}
		if err = processFile(cfg, file, c.GetErrorOnUnmatchedKeys()); err != nil {
			return true, err
		}
	}
	c.cfgModTimes = cfgModTimeMap

	if prefix := c.getENVPrefix(cfg); prefix == "-" {
		err = c.processTags(cfg)
	} else {
		err = c.processTags(cfg, prefix)
	}

	return true, err
}
