package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/DriteStudio/drite-cli/drite"
)

func printResponse(writer io.Writer, response *drite.Response, compact bool) error {
	if response == nil {
		return nil
	}
	body := bytes.TrimSpace(response.Body)
	if len(body) == 0 {
		_, err := fmt.Fprintln(writer, "{}")
		return err
	}
	if !json.Valid(body) {
		_, err := fmt.Fprintln(writer, string(body))
		return err
	}
	if compact {
		_, err := fmt.Fprintln(writer, string(body))
		return err
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, body, "", "  "); err != nil {
		return err
	}
	pretty.WriteByte('\n')
	_, err := writer.Write(pretty.Bytes())
	return err
}
