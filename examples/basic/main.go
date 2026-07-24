package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/DriteStudio/drite-cli/drite"
)

func main() {
	client, err := drite.NewClient(os.Getenv("DRITE_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.VPS.List(context.Background(), drite.VPSListOptions{Take: 20})
	if err != nil {
		var apiErr *drite.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("API error: HTTP %d code=%s message=%s", apiErr.StatusCode, apiErr.Code, apiErr.Message)
		}
		log.Fatal(err)
	}

	var payload map[string]any
	if err := response.Decode(&payload); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", payload)
}
