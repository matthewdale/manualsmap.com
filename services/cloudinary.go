package services

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"
)

type Cloudinary struct {
	cloudinarySecret string
}

func NewCloudinary(cloudinarySecret string) Cloudinary {
	return Cloudinary{cloudinarySecret: cloudinarySecret}
}

// encode encodes parameters in the Cloudinary signature
// string format.
func encode(parameters map[string]string) string {
	var buf strings.Builder

	// Collect and sort the keys in ascending order.
	keys := make([]string, 0, len(parameters))
	for k := range parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build the encoded signature string like
	// key1=value1&key2=value2&...
	for _, key := range keys {
		val := parameters[key]
		if buf.Len() > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.WriteString(val)
	}

	return buf.String()
}

func (svc Cloudinary) UploadSignature(parameters map[string]string) string {
	if parameters == nil {
		return ""
	}
	sig := encode(parameters) + svc.cloudinarySecret
	hash := sha1.Sum([]byte(sig))
	return hex.EncodeToString(hash[:])
}

func (svc Cloudinary) NotificationSignature(body string, timestamp string) string {
	sig := body + timestamp + svc.cloudinarySecret
	hash := sha1.Sum([]byte(sig))
	return hex.EncodeToString(hash[:])
}

func (svc Cloudinary) deliverySignature(img CloudinaryImage, transform string) string {
	sig := img.Path(transform) + svc.cloudinarySecret
	hash := sha1.Sum([]byte(sig))
	return fmt.Sprintf("s--%s--", base64.URLEncoding.EncodeToString(hash[:])[:8])
}

func (svc Cloudinary) URL(img CloudinaryImage, transform string) *url.URL {
	if img.Empty() {
		return new(url.URL)
	}

	return &url.URL{
		Scheme: "https",
		Host:   "res.cloudinary.com",
		Path: path.Join(
			"dawfgqsur",
			"image",
			"authenticated",
			svc.deliverySignature(img, transform),
			img.Path(transform)),
	}
}

type CloudinaryImage struct {
	PublicID string
	Format   string
}

func (img CloudinaryImage) Empty() bool {
	return img.PublicID == "" || img.Format == ""
}

func (img CloudinaryImage) Path(transform string) string {
	return path.Join(
		transform,
		fmt.Sprintf("%s.%s", img.PublicID, img.Format))
}
