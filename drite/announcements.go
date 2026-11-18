package drite

import (
	"context"
	"net/http"
	"net/url"
)

const announcementsPath = "/api/auth/me/announcements"

type AnnouncementListOptions struct {
	View   string
	Cursor string
	Limit  int
}

func (s *AccountService) Announcements(ctx context.Context, options AnnouncementListOptions) (*Response, error) {
	query := make(url.Values)
	if options.View != "" {
		query.Set("view", options.View)
	}
	if options.Cursor != "" {
		query.Set("cursor", options.Cursor)
	}
	queryInt(query, "limit", options.Limit)
	return s.client.Request(ctx, http.MethodGet, announcementsPath, query, nil)
}

func (s *AccountService) AnnouncementUnreadCount(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, announcementsPath+"/unread-count", nil, nil)
}

func (s *AccountService) ReadAllAnnouncements(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, announcementsPath+"/read-all", nil, nil)
}

func (s *AccountService) Announcement(ctx context.Context, announcementID string) (*Response, error) {
	id, err := pathSegment(announcementID, "announcement ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, announcementsPath+"/"+id, nil, nil)
}

func (s *AccountService) ReadAnnouncement(ctx context.Context, announcementID string) (*Response, error) {
	id, err := pathSegment(announcementID, "announcement ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, announcementsPath+"/"+id+"/read", nil, nil)
}
