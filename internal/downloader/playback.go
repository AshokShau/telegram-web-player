package downloader

import (
	"ashokshau/tg-web/internal/config"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

var conversions singleflight.Group

func playbackPath(source string) (string, error) {
	info, err := os.Stat(source)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return "", fmt.Errorf("audio source is empty or not a regular file")
	}

	identity := fmt.Sprintf("%s:%d:%d", source, info.Size(), info.ModTime().UnixNano())
	digest := sha256.Sum256([]byte(identity))
	output := filepath.Join(config.DownloadsDir, fmt.Sprintf("web-%x.m4a", digest))
	if validFile(output) {
		return output, nil
	}

	value, err, _ := conversions.Do(output, func() (any, error) {
		if validFile(output) {
			return output, nil
		}
		compatible, err := browserCompatible(source)
		if err != nil {
			return "", err
		}
		if compatible {
			return source, nil
		}
		if err := os.MkdirAll(config.DownloadsDir, 0o755); err != nil {
			return "", err
		}
		temp, err := os.CreateTemp(config.DownloadsDir, ".web-*.m4a")
		if err != nil {
			return "", err
		}
		tempPath := temp.Name()
		_ = temp.Close()
		defer os.Remove(tempPath)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-hide_banner", "-loglevel", "error", "-y",
			"-i", source, "-map", "0:a:0", "-vn", "-sn", "-dn", "-map_metadata", "-1",
			"-c:a", "aac", "-profile:a", "aac_low", "-b:a", "192k", "-ar", "48000", "-ac", "2",
			"-movflags", "+faststart", "-f", "ipod", tempPath)
		if detail, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("convert audio: %w: %s", err, strings.TrimSpace(string(detail)))
		}
		if !validFile(tempPath) {
			return "", fmt.Errorf("audio conversion produced an empty file")
		}
		if err := os.Rename(tempPath, output); err != nil {
			return "", err
		}
		return output, nil
	})
	if err != nil {
		return "", err
	}
	return value.(string), nil
}

func validFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

func browserCompatible(path string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "a:0",
		"-show_entries", "stream=codec_name,profile,sample_rate,channels", "-of", "json", path).Output()
	if err != nil {
		return false, fmt.Errorf("inspect audio: %w", err)
	}
	var result struct {
		Streams []struct {
			CodecName  string `json:"codec_name"`
			Profile    string `json:"profile"`
			SampleRate string `json:"sample_rate"`
			Channels   int    `json:"channels"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &result); err != nil || len(result.Streams) == 0 {
		return false, fmt.Errorf("no readable audio stream")
	}
	stream := result.Streams[0]
	rate, err := strconv.Atoi(stream.SampleRate)
	if err != nil || rate <= 0 || rate > 48000 || stream.Channels < 1 || stream.Channels > 2 {
		return false, nil
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		return stream.CodecName == "mp3", nil
	case ".m4a", ".mp4":
		return stream.CodecName == "aac" && stream.Profile == "LC", nil
	default:
		return false, nil
	}
}
