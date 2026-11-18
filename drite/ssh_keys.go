package drite

import (
	"context"
	"net/http"
)

const accountSSHKeysPath = "/api/auth/me/ssh-keys"

type AccountSSHKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PublicKey   string `json:"publicKey"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"createdAt"`
}

type AddSSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
}

func (s *AccountService) ListSSHKeys(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, accountSSHKeysPath, nil, nil)
}

func (s *AccountService) AddSSHKey(ctx context.Context, request AddSSHKeyRequest) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, accountSSHKeysPath, nil, request)
}

func (s *AccountService) DeleteSSHKey(ctx context.Context, keyID string) (*Response, error) {
	id, err := pathSegment(keyID, "SSH key ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodDelete, accountSSHKeysPath+"/"+id, nil, nil)
}
