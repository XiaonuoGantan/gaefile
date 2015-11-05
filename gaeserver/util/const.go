package util

import (
	"os"
)

func GetBucketName() string {
	return os.Getenv("GOOGLE_CLOUD_STAORAGE_BUCKET")
}
