package cli

import (
	"context"
	"fmt"

	"github.com/DriteStudio/drite-cli/drite"
)

func runBilling(
	ctx context.Context,
	client *drite.Client,
	arguments []string,
) (*drite.Response, error) {
	action, rest, err := requireAction(arguments, "usage: drite billing ACTION")
	if err != nil {
		return nil, err
	}
	switch action {
	case "transactions", "export":
		flags := newFlagSet("billing " + action)
		page := flags.Int("page", 0, "page number")
		limit := flags.Int("limit", 0, "page size")
		month := flags.String("month", "", "month in YYYY-MM")
		start := flags.String("start-date", "", "ISO start date")
		end := flags.String("end-date", "", "ISO end date")
		if err := flags.Parse(rest); err != nil {
			return nil, err
		}
		options := drite.TransactionListOptions{
			Page: *page, Limit: *limit, Month: *month, StartDate: *start, EndDate: *end,
		}
		if action == "export" {
			return client.Billing.ExportTransactions(ctx, options)
		}
		return client.Billing.Transactions(ctx, options)
	case "due":
		return client.Billing.DueItems(ctx)
	case "pay-due":
		body, err := bodyFromArgs[drite.PayDueItemRequest]("billing pay due", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Billing.PayDueItem(ctx, body)
	case "topup-history":
		return client.Billing.TopupHistory(ctx)
	case "topup-status":
		reference, err := requireID(rest, "reference number")
		if err != nil {
			return nil, err
		}
		return client.Billing.TopupStatus(ctx, reference)
	case "document-url":
		flags := newFlagSet("billing document-url")
		id := flags.String("id", "", "transaction/topup ID")
		documentType := flags.String("type", "", "transaction or topup")
		if err := flags.Parse(rest); err != nil {
			return nil, err
		}
		if *id == "" || *documentType == "" {
			return nil, fmt.Errorf("--id and --type are required")
		}
		return client.Billing.SignedDocumentURL(ctx, *id, *documentType)
	default:
		return nil, fmt.Errorf("unknown billing action %q", action)
	}
}
