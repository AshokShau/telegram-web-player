package downloader

import (
	"ashokshau/tg-web/internal/config"
	"context"
	"crypto/sha256"
	"fmt"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/sync/singleflight"
)

var coverExtractions singleflight.Group

func embeddedCoverURL(source string) string {
	info, err := os.Stat(source)
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}
	identity := fmt.Sprintf("%s:%d:%d", source, info.Size(), info.ModTime().UnixNano())
	name := fmt.Sprintf("web-cover-%x.jpg", sha256.Sum256([]byte(identity)))
	path := filepath.Join(config.DownloadsDir, name)
	value, err, _ := coverExtractions.Do(path, func() (any, error) {
		if validEmbeddedCover(path) {
			return "/thumbnail/embedded/" + name, nil
		}
		if err := os.MkdirAll(config.DownloadsDir, 0o755); err != nil {
			return "", err
		}
		temp, err := os.CreateTemp(config.DownloadsDir, ".web-cover-*.jpg")
		if err != nil {
			return "", err
		}
		tempPath := temp.Name()
		_ = temp.Close()
		defer os.Remove(tempPath)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-hide_banner", "-loglevel", "error", "-y",
			"-i", source, "-map", "0:v:disp:attached_pic", "-frames:v", "1", "-an", "-c:v", "mjpeg", "-q:v", "3", tempPath)
		if err := cmd.Run(); err != nil || !validEmbeddedCover(tempPath) {
			return "", err
		}
		if err := os.Rename(tempPath, path); err != nil {
			return "", err
		}
		return "/thumbnail/embedded/" + name, nil
	})
	if err != nil {
		return ""
	}
	return value.(string)
}

func validEmbeddedCover(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > 8<<20 {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	imageInfo, err := jpeg.DecodeConfig(file)
	return err == nil && imageInfo.Width >= 256 && imageInfo.Height >= 256
}
