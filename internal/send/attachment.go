package send

import (
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func LoadLocalAttachment(path string) (Attachment, error) {
	if path == "" {
		return Attachment{}, errors.New("attachment path cannot be empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return Attachment{}, fmt.Errorf("stat attachment %q: %w", path, err)
	}
	if info.IsDir() {
		return Attachment{}, fmt.Errorf("attachment %q is a directory", path)
	}

	dat, err := os.ReadFile(path)
	if err != nil {
		return Attachment{}, fmt.Errorf("read attachment %q: %w", path, err)
	}

	ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if ct == "" {
		ct = http.DetectContentType(dat)
	}

	return Attachment{
		Content:     base64.StdEncoding.EncodeToString(dat),
		Filename:    filepath.Base(path),
		ContentType: ct,
	}, nil
}
