package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) Client {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "bistro-docker-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "docker.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener.Close()
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	client := NewClient(socket)
	t.Cleanup(func() { client.Close() })
	return client
}

func TestImport(t *testing.T) {
	source := &trackedReader{Reader: strings.NewReader("rootfs tar bytes")}
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/images/create" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		query := r.URL.Query()
		for key, want := range map[string]string{
			"fromSrc": "-", "repo": "example/test", "tag": "latest", "platform": "linux/arm64",
		} {
			if got := query.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		if got := query["changes"]; !reflect.DeepEqual(got, []string{"ENV FOO=a&b", `CMD ["/bin/sh"]`}) {
			t.Errorf("unexpected changes: %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/octet-stream" {
			t.Errorf("unexpected content type: %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "rootfs tar bytes" {
			t.Errorf("unexpected body: %q, error: %v", body, err)
		}
		fmt.Fprintln(w, `{"status":"Importing"}`)
		w.(http.Flusher).Flush()
		fmt.Fprintln(w, `{"status":"sha256:example"}`)
	})
	id, err := client.Import(context.Background(), source, ImportOptions{
		Repository: "example/test",
		Tag:        "latest",
		Platform:   "linux/arm64",
		Changes:    []string{"ENV FOO=a&b", `CMD ["/bin/sh"]`},
	})
	if err != nil || id != "sha256:example" {
		t.Fatalf("Import() = %q, %v", id, err)
	}
	if source.closed {
		t.Fatal("Import closed the caller's reader")
	}
}

func TestImportResponseErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"stream error", `{"status":"sha256:example"}` + "\n" + `{"error":"import failed"}`, "import failed"},
		{"error detail", `{"errorDetail":{"message":"invalid tar"}}`, "invalid tar"},
		{"malformed response", `{"status":`, "decode response"},
		{"missing result", `{"status":"Importing"}`, "image ID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				fmt.Fprint(w, tc.body)
			})
			_, err := client.Import(context.Background(), strings.NewReader("tar"), ImportOptions{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Import() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestSave(t *testing.T) {
	archive := []byte{'t', 'a', 'r', 0, 0xff, '\n'}
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/images/registry.example/team/image:latest/get" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.Write(archive)
	})
	var dst bytes.Buffer
	if err := client.Save(context.Background(), "registry.example/team/image:latest", &dst); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dst.Bytes(), archive) {
		t.Fatalf("archive was changed: %q", dst.Bytes())
	}
	want := errors.New("destination failed")
	if err := client.Save(context.Background(), "registry.example/team/image:latest", failingWriter{want}); !errors.Is(err, want) {
		t.Fatalf("Save() error = %v, want %v", err, want)
	}
}

func TestHTTPErrors(t *testing.T) {
	for _, body := range []string{`{"message":"image not found"}`, "image not found"} {
		t.Run(body, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, body)
			})
			check := func(err error) {
				t.Helper()
				if err == nil || !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "image not found") {
					t.Fatalf("unexpected error: %v", err)
				}
			}
			check(client.Save(context.Background(), "missing", io.Discard))
			_, err := client.Import(context.Background(), strings.NewReader("tar"), ImportOptions{})
			check(err)
		})
	}
}

func TestCanceledContext(t *testing.T) {
	client := NewClient(filepath.Join(t.TempDir(), "unused.sock"))
	defer client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Save(ctx, "example", io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := client.Import(ctx, strings.NewReader("tar"), ImportOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Import() error = %v", err)
	}
}

type trackedReader struct {
	io.Reader
	closed bool
}

func (r *trackedReader) Close() error {
	r.closed = true
	return nil
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }
