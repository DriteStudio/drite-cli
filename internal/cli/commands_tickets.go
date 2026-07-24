package cli

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"

	"github.com/DriteStudio/drite-cli/drite"
)

func runTickets(
	ctx context.Context,
	client *drite.Client,
	arguments []string,
) (*drite.Response, error) {
	action, rest, err := requireAction(arguments, "usage: drite ticket ACTION")
	if err != nil {
		return nil, err
	}
	switch action {
	case "list":
		flags := newFlagSet("ticket list")
		page := flags.Int("page", 0, "page number")
		limit := flags.Int("limit", 0, "page size")
		status := flags.String("status", "all", "ticket status")
		category := flags.String("category", "all", "ticket category")
		search := flags.String("search", "", "ID or subject search")
		if err := flags.Parse(rest); err != nil {
			return nil, err
		}
		return client.Tickets.List(ctx, drite.TicketListOptions{
			Page: *page, Limit: *limit, Status: *status, Category: *category, Search: *search,
		})
	case "create":
		body, err := bodyFromArgs[drite.CreateTicketRequest]("ticket create", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Tickets.Create(ctx, body)
	case "upload":
		filePath, err := requireID(rest, "file path")
		if err != nil {
			return nil, err
		}
		flags := newFlagSet("ticket upload")
		contentType := flags.String("content-type", "", "MIME type; inferred when omitted")
		if err := flags.Parse(rest[1:]); err != nil {
			return nil, err
		}
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("open attachment: %w", err)
		}
		defer file.Close()
		detectedType := *contentType
		if detectedType == "" {
			detectedType = mime.TypeByExtension(filepath.Ext(filePath))
		}
		return client.Tickets.UploadAttachment(ctx, filepath.Base(filePath), detectedType, file)
	}

	id, err := requireID(rest, "ticket ID")
	if err != nil {
		return nil, err
	}
	switch action {
	case "get":
		return client.Tickets.Get(ctx, id)
	case "updates":
		flags := newFlagSet("ticket updates")
		since := flags.String("since", "", "RFC3339 updatedSince value")
		if err := flags.Parse(rest[1:]); err != nil {
			return nil, err
		}
		return client.Tickets.Updates(ctx, id, *since)
	case "reply":
		body, err := bodyFromArgs[drite.ReplyTicketRequest]("ticket reply", rest[1:], true)
		if err != nil {
			return nil, err
		}
		return client.Tickets.Reply(ctx, id, body)
	default:
		return nil, fmt.Errorf("unknown ticket action %q; customers cannot close tickets", action)
	}
}
