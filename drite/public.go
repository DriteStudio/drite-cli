package drite

import (
	"context"
	"net/http"
	"net/url"
)

type PublicService struct {
	client *Client
}

type ArticleListOptions struct {
	Page     int
	Limit    int
	Category string
	Search   string
}

func (s *PublicService) Plans(ctx context.Context) (*Response, error) {
	return s.client.PublicRequest(ctx, http.MethodGet, "/api/un_auth/plans/all", nil)
}

func (s *PublicService) HostingPlans(ctx context.Context) (*Response, error) {
	return s.client.PublicRequest(ctx, http.MethodGet, "/api/un_auth/hosting/plans", nil)
}

func (s *PublicService) ArticleCategories(ctx context.Context) (*Response, error) {
	return s.client.PublicRequest(ctx, http.MethodGet, "/api/un_auth/articles/categories", nil)
}

func (s *PublicService) Articles(
	ctx context.Context,
	options ArticleListOptions,
) (*Response, error) {
	query := make(url.Values)
	queryInt(query, "page", options.Page)
	queryInt(query, "limit", options.Limit)
	if options.Category != "" {
		query.Set("category", options.Category)
	}
	if options.Search != "" {
		query.Set("search", options.Search)
	}
	return s.client.PublicRequest(ctx, http.MethodGet, "/api/un_auth/articles", query)
}

func (s *PublicService) Article(ctx context.Context, slug string) (*Response, error) {
	value, err := pathSegment(slug, "article slug")
	if err != nil {
		return nil, err
	}
	return s.client.PublicRequest(ctx, http.MethodGet, "/api/un_auth/articles/"+value, nil)
}
