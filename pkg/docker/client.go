package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const DefaultSocketPath = "/var/run/docker.sock"

type Client interface {
	Import(ctx context.Context, src io.Reader, opts ImportOptions) (string, error)
	Save(ctx context.Context, image string, dst io.Writer) error
	Close() error
}

type client struct {
	http *http.Client
}

var _ Client = (*client)(nil)

func NewClient(socketPath string) Client {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	dialer := &net.Dialer{}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", socketPath)
		},
		DisableCompression: true,
	}
	return &client{http: &http.Client{Transport: transport}}
}

func (c *client) Close() error {
	c.http.CloseIdleConnections()
	return nil
}

type ImportOptions struct {
	Repository string
	Tag        string
	Platform   string
	Changes    []string
}

func (c *client) Import(ctx context.Context, src io.Reader, opts ImportOptions) (string, error) {
	if src == nil {
		return "", fmt.Errorf("docker import: source is nil")
	}
	query := url.Values{"fromSrc": {"-"}}
	if opts.Repository != "" {
		query.Set("repo", opts.Repository)
	}
	if opts.Tag != "" {
		query.Set("tag", opts.Tag)
	}
	if opts.Platform != "" {
		query.Set("platform", opts.Platform)
	}
	for _, change := range opts.Changes {
		query.Add("changes", change)
	}
	res, err := c.request(ctx, http.MethodPost, "/images/create?"+query.Encode(), struct{ io.Reader }{src})
	if err != nil {
		return "", fmt.Errorf("docker import: %w", err)
	}
	defer res.Body.Close()

	decoder := json.NewDecoder(res.Body)
	var imageID string
	for {
		var message struct {
			Status      string `json:"status"`
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
		}
		if err := decoder.Decode(&message); err == io.EOF {
			break
		} else if err != nil {
			return "", fmt.Errorf("docker import: decode response: %w", err)
		}
		if message.ErrorDetail.Message != "" {
			return "", fmt.Errorf("docker import: %s", message.ErrorDetail.Message)
		}
		if message.Error != "" {
			return "", fmt.Errorf("docker import: %s", message.Error)
		}
		if strings.HasPrefix(message.Status, "sha256:") {
			imageID = message.Status
		}
	}
	if imageID == "" {
		return "", fmt.Errorf("docker import: response did not contain an image ID")
	}
	return imageID, nil
}

func (c *client) Save(ctx context.Context, image string, dst io.Writer) error {
	if image == "" {
		return fmt.Errorf("docker save: image is empty")
	}
	if dst == nil {
		return fmt.Errorf("docker save: destination is nil")
	}
	res, err := c.request(ctx, http.MethodGet, "/images/"+url.PathEscape(image)+"/get", nil)
	if err != nil {
		return fmt.Errorf("docker save: %w", err)
	}
	defer res.Body.Close()
	if _, err := io.Copy(dst, res.Body); err != nil {
		return fmt.Errorf("docker save: copy archive: %w", err)
	}
	return nil
}

func (c *client) request(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, "http://docker"+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return res, nil
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	if err != nil {
		return nil, fmt.Errorf("%s: read error response: %w", res.Status, err)
	}
	var message struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &message) == nil && message.Message != "" {
		return nil, fmt.Errorf("%s: %s", res.Status, message.Message)
	}
	return nil, fmt.Errorf("%s: %s", res.Status, strings.TrimSpace(string(data)))
}
