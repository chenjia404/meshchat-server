package ipfs

import (
	"context"
	"errors"
	"net/url"

	"github.com/ipfs/go-cid"
)

// Client abstracts IPFS-specific operations so the rest of the application only deals with CIDs.
type Client interface {
	ValidateCID(raw string) error
	RegisterMetadata(_ context.Context, raw string) error
	Pin(_ context.Context, raw string) error
}

// LocalClient performs lightweight CID validation and leaves pinning as an extension point.
type LocalClient struct {
	apiURL *url.URL
}

func NewLocalClient(rawURL string) (*LocalClient, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return &LocalClient{apiURL: parsed}, nil
}

func (c *LocalClient) ValidateCID(raw string) error {
	if raw == "" {
		return errors.New("cid is required")
	}
	_, err := cid.Decode(raw)
	return err
}

func (c *LocalClient) RegisterMetadata(_ context.Context, raw string) error {
	return c.ValidateCID(raw)
}

func (c *LocalClient) Pin(_ context.Context, raw string) error {
	return c.ValidateCID(raw)
}
