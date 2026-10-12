// Package cloudinary stores entry photos in Cloudinary.
package cloudinary

import (
	"context"
	"errors"
	"io"

	cld "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// deliveryTransformation caps the delivered size so free-tier credits last.
const deliveryTransformation = "w_1600,h_1600,c_limit,q_auto"

// Uploader implements service.PhotoUploader on top of the Cloudinary Upload API.
type Uploader struct {
	client *cld.Cloudinary
}

// NewUploader builds an Uploader from a configured Cloudinary client.
func NewUploader(client *cld.Cloudinary) *Uploader {
	return &Uploader{client: client}
}

// Upload stores file as boraquest/<guildID>/entries/<entryID> and returns the
// URL of its size-capped delivery variant plus the Cloudinary public_id.
func (u *Uploader) Upload(ctx context.Context, guildID, entryID string, file io.Reader) (url, storedID string, err error) {
	folder := "boraquest/" + guildID + "/entries"
	res, err := u.client.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:       folder + "/" + entryID,
		AssetFolder:    folder,
		Overwrite:      new(false),
		UniqueFilename: new(false),
		Eager:          deliveryTransformation,
	})
	if err != nil {
		return "", "", err
	}
	// The SDK reports API-level failures (bad credentials, rejected file) in the body, not as err.
	if res.Error.Message != "" {
		return "", "", errors.New("cloudinary: " + res.Error.Message)
	}
	url = res.SecureURL
	if len(res.Eager) > 0 && res.Eager[0].SecureURL != "" {
		url = res.Eager[0].SecureURL
	}
	return url, res.PublicID, nil
}
