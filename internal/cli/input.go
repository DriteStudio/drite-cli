package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

type dataFlags struct {
	inline string
	file   string
}

func addDataFlags(flags *flag.FlagSet) *dataFlags {
	data := &dataFlags{}
	flags.StringVar(&data.inline, "data", "", "inline JSON request body")
	flags.StringVar(&data.file, "data-file", "", "path to JSON request body")
	return data
}

func (data *dataFlags) bytes(required bool) ([]byte, error) {
	if data.inline != "" && data.file != "" {
		return nil, fmt.Errorf("use only one of --data or --data-file")
	}
	if data.file != "" {
		content, err := os.ReadFile(data.file)
		if err != nil {
			return nil, fmt.Errorf("read --data-file: %w", err)
		}
		return content, nil
	}
	if data.inline != "" {
		if strings.HasPrefix(data.inline, "@") {
			content, err := os.ReadFile(strings.TrimPrefix(data.inline, "@"))
			if err != nil {
				return nil, fmt.Errorf("read --data file: %w", err)
			}
			return content, nil
		}
		return []byte(data.inline), nil
	}
	if required {
		return nil, fmt.Errorf("request body is required; use --data or --data-file")
	}
	return []byte("{}"), nil
}

func decodeData[T any](data *dataFlags, required bool) (T, error) {
	var value T
	err := decodeDataInto(data, required, &value)
	return value, err
}

func decodeDataInto(data *dataFlags, required bool, target any) error {
	content, err := data.bytes(required)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, target); err != nil {
		return fmt.Errorf("decode JSON request body: %w", err)
	}
	return nil
}

func decodeUntypedData(data *dataFlags, required bool) (any, error) {
	return decodeData[any](data, required)
}

func bodyFromArgs[T any](name string, arguments []string, required bool) (T, error) {
	flags := newFlagSet(name)
	data := addDataFlags(flags)
	if err := parseBodyFlags(flags, arguments); err != nil {
		var zero T
		return zero, err
	}
	return decodeData[T](data, required)
}

func parseBodyFlags(flags *flag.FlagSet, arguments []string) error {
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	return nil
}

func parseRequiredBool(value string, flagName string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "":
		return false, fmt.Errorf("%s is required and must be true or false", flagName)
	default:
		return false, fmt.Errorf("%s must be true or false", flagName)
	}
}

type repeatedFlag []string

func (values *repeatedFlag) String() string {
	return strings.Join(*values, ",")
}

func (values *repeatedFlag) Set(value string) error {
	*values = append(*values, value)
	return nil
}
