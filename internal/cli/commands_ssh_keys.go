package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/DriteStudio/drite-cli/drite"
)

func runSSHKey(ctx context.Context, account *drite.AccountService, arguments []string) (*drite.Response, error) {
	action, rest, err := requireAction(arguments, "usage: drite ssh-key add|list|delete")
	if err != nil {
		return nil, err
	}
	switch action {
	case "list":
		if len(rest) != 0 {
			return nil, fmt.Errorf("usage: drite ssh-key list")
		}
		return account.ListSSHKeys(ctx)
	case "add":
		flags := newFlagSet("ssh-key add")
		name := flags.String("name", "", "name of the saved key")
		file := flags.String("file", "", "path to the SSH public key file")
		if err := flags.Parse(rest); err != nil {
			return nil, err
		}
		if flags.NArg() != 0 || strings.TrimSpace(*name) == "" || *file == "" {
			return nil, fmt.Errorf("usage: drite ssh-key add --name NAME --file PATH")
		}
		publicKey, err := readSSHPublicKeyFile(*file)
		if err != nil {
			return nil, err
		}
		return account.AddSSHKey(ctx, drite.AddSSHKeyRequest{Name: strings.TrimSpace(*name), PublicKey: publicKey})
	case "delete":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: drite ssh-key delete KEY_ID")
		}
		return account.DeleteSSHKey(ctx, rest[0])
	default:
		return nil, fmt.Errorf("unknown ssh-key action %q", action)
	}
}

func readSSHPublicKeyFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open SSH public key file: %w", err)
	}
	defer file.Close()

	const maxBytes = 16 << 10
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read SSH public key file: %w", err)
	}
	if len(content) > maxBytes {
		return "", fmt.Errorf("SSH public key file exceeds 16 KiB")
	}

	key := strings.TrimSpace(string(content))
	separator := strings.IndexAny(key, " \t")
	if separator < 0 || strings.ContainsAny(key, "\r\n\x00") || strings.Contains(key, "PRIVATE KEY") ||
		(!strings.HasPrefix(key[:separator], "ssh-") && !strings.HasPrefix(key[:separator], "ecdsa-")) {
		return "", fmt.Errorf("expected one SSH public key in OpenSSH format; use the .pub file")
	}
	return key, nil
}

func vpsBodyWithSSHKeys(
	ctx context.Context,
	account *drite.AccountService,
	name string,
	arguments []string,
	requireBody bool,
	request any,
	keyIDs **[]string,
) (*drite.Response, error) {
	flags := newFlagSet(name)
	data := addDataFlags(flags)
	var requestedKeys repeatedFlag
	flags.Var(&requestedKeys, "ssh-key", "saved SSH key name or ID (repeatable)")
	if err := parseBodyFlags(flags, arguments); err != nil {
		return nil, err
	}
	if err := decodeDataInto(data, requireBody || len(requestedKeys) == 0, request); err != nil {
		return nil, err
	}
	if len(requestedKeys) == 0 {
		return nil, nil
	}

	response, err := account.ListSSHKeys(ctx)
	if err != nil {
		return response, err
	}
	var list struct {
		Data []drite.AccountSSHKey `json:"data"`
	}
	if err := response.Decode(&list); err != nil {
		return nil, err
	}

	var ids []string
	if *keyIDs != nil {
		ids = append(ids, **keyIDs...)
	}
	for _, requested := range requestedKeys {
		id := findSSHKeyID(list.Data, strings.TrimSpace(requested))
		if id == "" {
			return nil, fmt.Errorf("saved SSH key %q was not found", requested)
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	*keyIDs = &ids
	return nil, nil
}

func findSSHKeyID(keys []drite.AccountSSHKey, selection string) string {
	for _, key := range keys {
		if key.ID == selection {
			return key.ID
		}
	}
	for _, key := range keys {
		if strings.EqualFold(key.Name, selection) {
			return key.ID
		}
	}
	return ""
}
