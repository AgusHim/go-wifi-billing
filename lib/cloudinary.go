package lib

import (
	"context"
	"errors"
	"mime/multipart"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

func newCloudinaryClient() (*cloudinary.Cloudinary, error) {
	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		return nil, err
	}
	return cld, nil
}

// UploadExpenseImage validates, resizes to max 1280px, converts to WebP,
// uploads to Cloudinary folder "wifi_billing/expenses", and returns (secureURL, publicID, error).
func UploadExpenseImage(fileHeader *multipart.FileHeader) (string, string, error) {
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}
	contentType := fileHeader.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		return "", "", errors.New("invalid file type: only JPEG, PNG, WebP, and GIF are allowed")
	}

	const maxSize = 5 * 1024 * 1024 // 5 MB
	if fileHeader.Size > maxSize {
		return "", "", errors.New("file size exceeds 5MB limit")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	cld, err := newCloudinaryClient()
	if err != nil {
		return "", "", err
	}

	ctx := context.Background()
	uploadResult, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:         "wifi_billing/expenses",
		Format:         "webp",
		Transformation: "c_limit,w_1280,h_1280,q_80",
	})
	if err != nil {
		return "", "", err
	}

	return uploadResult.SecureURL, uploadResult.PublicID, nil
}

// DeleteExpenseImage removes an image from Cloudinary by its public_id.
// A blank publicID is treated as a no-op.
func DeleteExpenseImage(publicID string) error {
	if publicID == "" {
		return nil
	}
	cld, err := newCloudinaryClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, err = cld.Upload.Destroy(ctx, uploader.DestroyParams{PublicID: publicID})
	return err
}
