package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/DriteStudio/drite-cli/drite"
)

func runReseller(
	ctx context.Context,
	client *drite.Client,
	arguments []string,
) (*drite.Response, error) {
	action, rest, err := requireAction(arguments, "usage: drite reseller ACTION")
	if err != nil {
		return nil, err
	}
	switch action {
	case "vps-plans":
		return client.Reseller.VPSPlans(ctx)
	case "vps-templates":
		return client.Reseller.VPSTemplates(ctx)
	case "vps-list":
		return client.Reseller.VPSList(ctx)
	case "vps-get":
		id, err := requireID(rest, "VPS ID")
		if err != nil {
			return nil, err
		}
		return client.Reseller.VPS(ctx, id)
	case "vps-create":
		body, err := bodyFromArgs[drite.CreateVPSRequest]("reseller VPS create", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Reseller.CreateVPS(ctx, body)
	case "custom-quote":
		body, err := bodyFromArgs[drite.CustomVPSQuoteRequest]("reseller custom quote", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Reseller.QuoteCustomVPS(ctx, body)
	case "custom-create":
		body, err := bodyFromArgs[drite.CreateCustomVPSRequest]("reseller custom create", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Reseller.CreateCustomVPS(ctx, body)
	default:
		return nil, fmt.Errorf("unknown reseller action %q", action)
	}
}

func runPublic(
	ctx context.Context,
	client *drite.Client,
	arguments []string,
) (*drite.Response, error) {
	action, rest, err := requireAction(arguments, "usage: drite public ACTION")
	if err != nil {
		return nil, err
	}
	switch action {
	case "plans":
		return client.Public.Plans(ctx)
	case "hosting-plans":
		return client.Public.HostingPlans(ctx)
	case "article-categories":
		return client.Public.ArticleCategories(ctx)
	case "articles":
		flags := newFlagSet("public articles")
		page := flags.Int("page", 0, "page")
		limit := flags.Int("limit", 0, "page size")
		category := flags.String("category", "", "category")
		search := flags.String("search", "", "search")
		if err := flags.Parse(rest); err != nil {
			return nil, err
		}
		return client.Public.Articles(ctx, drite.ArticleListOptions{
			Page: *page, Limit: *limit, Category: *category, Search: *search,
		})
	case "article":
		slug, err := requireID(rest, "article slug")
		if err != nil {
			return nil, err
		}
		return client.Public.Article(ctx, slug)
	default:
		return nil, fmt.Errorf("unknown public action %q", action)
	}
}

func runRaw(
	ctx context.Context,
	client *drite.Client,
	arguments []string,
) (*drite.Response, error) {
	if len(arguments) < 2 {
		return nil, fmt.Errorf("usage: drite request METHOD /api/path [--query key=value] [--data JSON]")
	}
	method := strings.ToUpper(arguments[0])
	endpoint := arguments[1]
	flags := newFlagSet("request")
	data := addDataFlags(flags)
	var queryFlags repeatedFlag
	flags.Var(&queryFlags, "query", "query parameter key=value (repeatable)")
	public := flags.Bool("public", false, "do not send the API token")
	if err := flags.Parse(arguments[2:]); err != nil {
		return nil, err
	}
	query := make(url.Values)
	for _, item := range queryFlags {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --query %q; use key=value", item)
		}
		query.Add(key, value)
	}
	var body any
	if data.inline != "" || data.file != "" {
		value, err := decodeUntypedData(data, true)
		if err != nil {
			return nil, err
		}
		body = value
	}
	if *public {
		if body != nil || (method != http.MethodGet && method != http.MethodHead) {
			return nil, fmt.Errorf("--public raw requests currently support GET/HEAD without a body")
		}
		return client.PublicRequest(ctx, method, endpoint, query)
	}
	return client.Request(ctx, method, endpoint, query, body)
}
