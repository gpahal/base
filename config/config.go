package config

import (
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/pkg/errors"

	_ "github.com/joho/godotenv/autoload"
)

type LoadOptions struct {
	Validator *validator.Validate
}

func Load(filePath string, config any) error {
	return LoadWithOptions(filePath, config, LoadOptions{})
}

func LoadWithOptions(configFilePath string, config any, opts LoadOptions) error {
	configFile, err := os.Open(configFilePath)
	if err != nil {
		return err
	}
	defer configFile.Close()

	configBytes, err := io.ReadAll(configFile)
	if err != nil {
		return err
	}

	configMap := make(map[string]any)
	if err := json.Unmarshal(configBytes, &configMap); err != nil {
		return err
	}

	if newConfigMap, err := substituteEnvVars(configMap); err != nil {
		return err
	} else {
		configMap = newConfigMap.(map[string]any)
	}

	if err := mapstructure.Decode(configMap, config); err != nil {
		return err
	}

	if opts.Validator != nil {
		if err := opts.Validator.Struct(config); err != nil {
			return err
		}
	}

	return nil
}

func substituteEnvVars(value any) (any, error) {
	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, "ENV[") && strings.HasSuffix(v, "]") {
			envVar := v[4 : len(v)-1]
			if value, ok := os.LookupEnv(envVar); ok {
				return value, nil
			} else {
				return nil, errors.Errorf("environment variable %s not found", envVar)
			}
		}
	case map[string]any:
		for key, value := range v {
			if newValue, err := substituteEnvVars(value); err != nil {
				return nil, err
			} else {
				v[key] = newValue
			}
		}
	case []any:
		for idx, value := range v {
			if newValue, err := substituteEnvVars(value); err != nil {
				return nil, err
			} else {
				v[idx] = newValue
			}
		}
	}

	return value, nil
}
