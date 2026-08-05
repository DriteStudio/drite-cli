# Drite CLI and Go SDK

Go client for the Dritestudio customer API. Customers use the `dr_live_...`
API token created on the Dritestudio website.

This repository contains:

- `drite`: an importable Go SDK, split into Account, VPS, Hosting, Billing,
  Tickets, Reseller, and Public services.
- `cmd/drite`: a script-friendly command line client.
- [docs/API.md](./docs/API.md): customer-facing HTTP API documentation with
  authentication, request bodies, errors, limits, and `curl` examples.
- [Docs.md](./Docs.md): endpoint-to-function coverage and request examples.

## Install the CLI

```bash
go install github.com/DriteStudio/drite-cli/cmd/drite@latest
```

Or build the current source:

```bash
go build -trimpath -o dist/drite ./cmd/drite
```

On Windows:

```powershell
go build -trimpath -o dist/drite.exe ./cmd/drite
```

## Authenticate

Save the API token in the current user's config directory:

```powershell
drite auth login --token dr_live_xxx
drite auth status
```

For CI/CD, avoid writing a config file:

```powershell
$env:DRITE_API_KEY = "dr_live_xxx"
$env:DRITE_API_URL = "https://dritestudio.co.th"
```

The CLI never prints the token in `auth status`.

## CLI examples

```powershell
drite me
drite vps list
drite vps get <vps_id>
drite vps start <vps_id>
drite hosting list
drite billing transactions --page 1 --limit 20
drite ticket list --status open
```

Mutation payloads use JSON:

```powershell
drite vps rename <vps_id> --data '{"name":"production"}'
drite vps create --data-file .\create-vps.json
drite ticket create --data-file .\ticket.json
```

Upload a ticket attachment, then put the returned `key` into the ticket body:

```powershell
drite ticket upload .\debug.log
```

Ticket files are restricted to TXT, LOG, and `image/*`, up to 10 MiB each and
five files per message.

## Go SDK example

```go
package main

import (
	"context"
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

	response, err := client.VPS.List(context.Background(), drite.VPSListOptions{
		Take: 20,
	})
	if err != nil {
		log.Fatal(err)
	}

	var payload map[string]any
	if err := response.Decode(&payload); err != nil {
		log.Fatal(err)
	}
	fmt.Println(payload)
}
```

The SDK retains response bodies as JSON so new backend fields do not break
older customer applications. Stable request payloads use typed Go structs.

## Security

- The default authentication header is `Authorization: Bearer <token>`.
- `drite.WithAPIKeyHeader()` uses `X-API-Key` when required.
- Absolute request URLs are rejected so a token cannot accidentally be sent to
  another host.
- Public API calls never attach the customer token.
- Configure exact allowed IP addresses on the website or through
  `Account.SetAPIKeySecurity`; the backend does not support CIDR entries.

## Development

```bash
gofmt -w cmd drite internal
go test ./...
go vet ./...
go build ./cmd/drite
```
