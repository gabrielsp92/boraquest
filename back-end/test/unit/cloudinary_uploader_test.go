package unit_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cld "github.com/cloudinary/cloudinary-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/cloudinary"
)

// uploadRequest is what the fake Cloudinary Upload API received.
type uploadRequest struct {
	path   string
	fields map[string]string
	file   string
}

// fakeCloudinary serves reply on every request and records the last one into got.
func fakeCloudinary(t *testing.T, reply string, got *uploadRequest) *cloudinary.Uploader {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		got.fields = map[string]string{}
		if err := r.ParseMultipartForm(1 << 20); err == nil {
			for k, v := range r.MultipartForm.Value {
				got.fields[k] = v[0]
			}
			if f, _, err := r.FormFile("file"); err == nil {
				b, _ := io.ReadAll(f)
				got.file = string(b)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return cloudinaryAt(t, srv.URL)
}

func cloudinaryAt(t *testing.T, uploadPrefix string) *cloudinary.Uploader {
	t.Helper()
	client, err := cld.NewFromParams("demo", "key", "secret")
	require.NoError(t, err)
	client.Upload.Config.API.UploadPrefix = uploadPrefix
	return cloudinary.NewUploader(client)
}

func TestCloudinaryUploaderUsesEagerDeliveryURL(t *testing.T) {
	var got uploadRequest
	u := fakeCloudinary(t, `{
		"public_id": "boraquest/g1/entries/e1",
		"secure_url": "https://res.cloudinary.com/demo/original.jpg",
		"eager": [{"secure_url": "https://res.cloudinary.com/demo/w_1600/e1.jpg"}]
	}`, &got)

	url, storedID, err := u.Upload(context.Background(), "g1", "e1", strings.NewReader("img-bytes"))

	require.NoError(t, err)
	assert.Equal(t, "https://res.cloudinary.com/demo/w_1600/e1.jpg", url)
	assert.Equal(t, "boraquest/g1/entries/e1", storedID)
	assert.Equal(t, "/v1_1/demo/auto/upload", got.path)
	assert.Equal(t, "img-bytes", got.file)
	assert.Equal(t, "boraquest/g1/entries/e1", got.fields["public_id"])
	assert.Equal(t, "boraquest/g1/entries", got.fields["asset_folder"])
	assert.Equal(t, "false", got.fields["overwrite"])
	assert.Equal(t, "false", got.fields["unique_filename"])
	assert.Equal(t, "w_1600,h_1600,c_limit,q_auto", got.fields["eager"])
	assert.NotEmpty(t, got.fields["signature"], "uploads must be signed with the API secret")
}

func TestCloudinaryUploaderFallsBackToSecureURL(t *testing.T) {
	for name, reply := range map[string]string{
		"no eager":          `{"public_id": "p", "secure_url": "https://res.cloudinary.com/demo/original.jpg"}`,
		"eager without url": `{"public_id": "p", "secure_url": "https://res.cloudinary.com/demo/original.jpg", "eager": [{}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			url, storedID, err := fakeCloudinary(t, reply, &uploadRequest{}).Upload(context.Background(), "g1", "e1", strings.NewReader("x"))

			require.NoError(t, err)
			assert.Equal(t, "https://res.cloudinary.com/demo/original.jpg", url)
			assert.Equal(t, "p", storedID)
		})
	}
}

func TestCloudinaryUploaderAPIError(t *testing.T) {
	u := fakeCloudinary(t, `{"error": {"message": "Invalid Signature"}}`, &uploadRequest{})

	url, storedID, err := u.Upload(context.Background(), "g1", "e1", strings.NewReader("x"))

	assert.EqualError(t, err, "cloudinary: Invalid Signature")
	assert.Empty(t, url)
	assert.Empty(t, storedID)
}

func TestCloudinaryUploaderTransportError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	_, _, err := cloudinaryAt(t, srv.URL).Upload(context.Background(), "g1", "e1", strings.NewReader("x"))

	assert.Error(t, err)
}
