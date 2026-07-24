package drite

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type ContainerService struct {
	client *Client
}

type CreateRegistryRequest struct {
	TemplateID  string `json:"templateId,omitempty"`
	Name        string `json:"name"`
	RegistryURL string `json:"registryUrl,omitempty"`
	Username    string `json:"username,omitempty"`
	Secret      string `json:"secret,omitempty"`
	AuthType    string `json:"authType,omitempty"`
	IsDefault   bool   `json:"isDefault"`
}

type CreateContainerAppRequest struct {
	Name          string       `json:"name"`
	Image         string       `json:"image"`
	ContainerPort int          `json:"containerPort,omitempty"`
	Replicas      int          `json:"replicas,omitempty"`
	RunAsNonRoot  *bool        `json:"runAsNonRoot,omitempty"`
	RunAsUser     *int         `json:"runAsUser,omitempty"`
	RunAsGroup    *int         `json:"runAsGroup,omitempty"`
	FSGroup       *int         `json:"fsGroup,omitempty"`
	PlanID        string       `json:"planId"`
	DurationType  DurationType `json:"durationType,omitempty"`
	ClusterID     string       `json:"clusterId,omitempty"`
	Region        string       `json:"region,omitempty"`
	RegistryID    string       `json:"registryId,omitempty"`
}

type UpdateContainerAppRequest struct {
	Name          string `json:"name,omitempty"`
	Image         string `json:"image,omitempty"`
	ContainerPort int    `json:"containerPort,omitempty"`
	Replicas      int    `json:"replicas,omitempty"`
	RunAsNonRoot  *bool  `json:"runAsNonRoot,omitempty"`
	RunAsUser     *int   `json:"runAsUser,omitempty"`
	RunAsGroup    *int   `json:"runAsGroup,omitempty"`
	FSGroup       *int   `json:"fsGroup,omitempty"`
}

type ContainerLogOptions struct {
	Pod          string
	TailLines    int
	SinceSeconds int
}

type ContainerAction string

const (
	ContainerDeploy  ContainerAction = "deploy"
	ContainerStart   ContainerAction = "start"
	ContainerStop    ContainerAction = "stop"
	ContainerRestart ContainerAction = "restart"
	ContainerDelete  ContainerAction = "delete"
)

func (s *ContainerService) Plans(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/containers/plans", nil, nil)
}

func (s *ContainerService) RegistryTemplates(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/containers/registry-templates", nil, nil)
}

func (s *ContainerService) Registries(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/containers/registries", nil, nil)
}

func (s *ContainerService) CreateRegistry(
	ctx context.Context,
	request CreateRegistryRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/containers/registries", nil, request)
}

func (s *ContainerService) DeleteRegistry(ctx context.Context, registryID string) (*Response, error) {
	id, err := pathSegment(registryID, "registry ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodDelete, "/api/auth/containers/registries/"+id, nil, nil)
}

func (s *ContainerService) Apps(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/containers/apps", nil, nil)
}

func (s *ContainerService) App(ctx context.Context, appID string) (*Response, error) {
	path, err := containerAppPath(appID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, path, nil, nil)
}

func (s *ContainerService) CreateApp(
	ctx context.Context,
	request CreateContainerAppRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/containers/apps", nil, request)
}

func (s *ContainerService) UpdateApp(
	ctx context.Context,
	appID string,
	request UpdateContainerAppRequest,
) (*Response, error) {
	path, err := containerAppPath(appID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPatch, path, nil, request)
}

func (s *ContainerService) Action(
	ctx context.Context,
	appID string,
	action ContainerAction,
) (*Response, error) {
	switch action {
	case ContainerDeploy, ContainerStart, ContainerStop, ContainerRestart, ContainerDelete:
	default:
		return nil, fmt.Errorf("drite: invalid container action %q", action)
	}
	path, err := containerAppPath(appID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, path+"/"+string(action), nil, nil)
}

func (s *ContainerService) SetAutoRenewal(
	ctx context.Context,
	appID string,
	enabled bool,
) (*Response, error) {
	path, err := containerAppPath(appID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(
		ctx,
		http.MethodPatch,
		path+"/auto-renewal",
		nil,
		AutoRenewalRequest{Enabled: enabled},
	)
}

func (s *ContainerService) Renew(
	ctx context.Context,
	appID string,
	duration DurationType,
) (*Response, error) {
	path, err := containerAppPath(appID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(
		ctx,
		http.MethodPost,
		path+"/renew",
		nil,
		struct {
			DurationType DurationType `json:"durationType,omitempty"`
		}{DurationType: duration},
	)
}

func (s *ContainerService) Operations(ctx context.Context, appID string) (*Response, error) {
	return s.getAppAction(ctx, appID, "operations", nil)
}

func (s *ContainerService) Runtime(ctx context.Context, appID string) (*Response, error) {
	return s.getAppAction(ctx, appID, "runtime", nil)
}

func (s *ContainerService) Logs(
	ctx context.Context,
	appID string,
	options ContainerLogOptions,
) (*Response, error) {
	query := make(url.Values)
	if options.Pod != "" {
		query.Set("pod", options.Pod)
	}
	queryInt(query, "tailLines", options.TailLines)
	queryInt(query, "sinceSeconds", options.SinceSeconds)
	return s.getAppAction(ctx, appID, "logs", query)
}

func (s *ContainerService) Environment(ctx context.Context, appID string) (*Response, error) {
	return s.getAppAction(ctx, appID, "env", nil)
}

func (s *ContainerService) SetEnvironmentVariable(
	ctx context.Context,
	appID string,
	key string,
	value string,
) (*Response, error) {
	path, err := containerAppPath(appID)
	if err != nil {
		return nil, err
	}
	envKey, err := pathSegment(key, "environment key")
	if err != nil {
		return nil, err
	}
	return s.client.Request(
		ctx,
		http.MethodPut,
		path+"/env/"+envKey,
		nil,
		struct {
			Value string `json:"value"`
		}{Value: value},
	)
}

func (s *ContainerService) DeleteEnvironmentVariable(
	ctx context.Context,
	appID string,
	key string,
) (*Response, error) {
	path, err := containerAppPath(appID)
	if err != nil {
		return nil, err
	}
	envKey, err := pathSegment(key, "environment key")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodDelete, path+"/env/"+envKey, nil, nil)
}

func (s *ContainerService) AttachRegistry(
	ctx context.Context,
	appID string,
	registryID string,
) (*Response, error) {
	return s.registryAction(ctx, http.MethodPost, appID, registryID)
}

func (s *ContainerService) DetachRegistry(
	ctx context.Context,
	appID string,
	registryID string,
) (*Response, error) {
	return s.registryAction(ctx, http.MethodDelete, appID, registryID)
}

func (s *ContainerService) AddDomain(
	ctx context.Context,
	appID string,
	hostname string,
) (*Response, error) {
	path, err := containerAppPath(appID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(
		ctx,
		http.MethodPost,
		path+"/domains",
		nil,
		struct {
			Hostname string `json:"hostname"`
		}{Hostname: hostname},
	)
}

func (s *ContainerService) VerifyDomain(
	ctx context.Context,
	appID string,
	domainID string,
) (*Response, error) {
	return s.domainAction(ctx, http.MethodPost, appID, domainID)
}

func (s *ContainerService) DeleteDomain(
	ctx context.Context,
	appID string,
	domainID string,
) (*Response, error) {
	return s.domainAction(ctx, http.MethodDelete, appID, domainID)
}

func (s *ContainerService) getAppAction(
	ctx context.Context,
	appID string,
	action string,
	query url.Values,
) (*Response, error) {
	path, err := containerAppPath(appID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, path+"/"+action, query, nil)
}

func (s *ContainerService) registryAction(
	ctx context.Context,
	method string,
	appID string,
	registryID string,
) (*Response, error) {
	path, err := containerAppPath(appID)
	if err != nil {
		return nil, err
	}
	registry, err := pathSegment(registryID, "registry ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, method, path+"/registries/"+registry, nil, nil)
}

func (s *ContainerService) domainAction(
	ctx context.Context,
	method string,
	appID string,
	domainID string,
) (*Response, error) {
	path, err := containerAppPath(appID)
	if err != nil {
		return nil, err
	}
	domain, err := pathSegment(domainID, "domain ID")
	if err != nil {
		return nil, err
	}
	suffix := path + "/domains/" + domain
	if method == http.MethodPost {
		suffix += "/verify"
	}
	return s.client.Request(ctx, method, suffix, nil, nil)
}

func containerAppPath(appID string) (string, error) {
	id, err := pathSegment(appID, "container app ID")
	if err != nil {
		return "", err
	}
	return "/api/auth/containers/apps/" + id, nil
}
