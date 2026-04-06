package ipfs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/ipfs/go-cid"
)

// Client abstracts IPFS-specific operations so the rest of the application only deals with CIDs.
type Client interface {
	ValidateCID(raw string) error
	RegisterMetadata(_ context.Context, raw string) error
	Pin(_ context.Context, raw string) error
	Add(_ context.Context, name string, content io.Reader) (string, error)
}

// LocalClient performs lightweight CID validation and leaves pinning as an extension point.
type LocalClient struct {
	apiURL *url.URL
	http   *http.Client
}

func NewLocalClient(rawURL string) (*LocalClient, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("ipfs api url must be an absolute url with scheme and host")
	}
	return &LocalClient{
		apiURL: parsed,
		http:   &http.Client{},
	}, nil
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

func (c *LocalClient) Add(ctx context.Context, name string, content io.Reader) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", sanitizeName(name))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, content); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	endpoint := *c.apiURL
	endpoint.Path = joinURLPath(endpoint.Path, "/api/v0/add")
	query := endpoint.Query()
	query.Set("pin", "true")
	query.Set("cid-version", "1")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = resp.Status
		}
		return "", errors.New(message)
	}

	var result struct {
		Hash string `json:"Hash"`
	}
	decoder := json.NewDecoder(resp.Body)
	for {
		if err := decoder.Decode(&result); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
	}
	if result.Hash == "" {
		return "", errors.New("ipfs add returned empty cid")
	}
	return result.Hash, nil
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "upload"
	}
	return path.Base(strings.ReplaceAll(name, "\\", "/"))
}

func joinURLPath(basePath, suffix string) string {
	basePath = strings.TrimRight(basePath, "/")
	if basePath == "" {
		return suffix
	}
	return basePath + suffix
}
