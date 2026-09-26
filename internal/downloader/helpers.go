/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package downloader

import (
	"ashokshau/tg-web/internal/utils"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	downloadTimeout        = 40 * time.Second
	defaultDownloadDirPerm = 0755
)

var (
	errMissingCDNURL = errors.New("missing cdn url")
)

// processDownload processes a track download based on platform metadata.
func processDownload(track *utils.TrackInfo) (string, error) {
	if track.CdnURL == "" {
		return "", errMissingCDNURL
	}

	if track.Key != "" && strings.EqualFold(track.Platform, "spotify") {
		return processSpotify(track)
	}

	return track.CdnURL, nil
}

var (
	sanitizeRegex = regexp.MustCompile(`[<>:"/\\|?*]`)
	filenameRegex = regexp.MustCompile(`filename\*?=(?:UTF-8'')?([^;]+)`)
)

// sanitizeFilename removes invalid characters from a filename to ensure it is safe for the filesystem.
func sanitizeFilename(fileName string) string {
	fileName = strings.ReplaceAll(fileName, "/", "")
	fileName = strings.ReplaceAll(fileName, "\\", "")
	fileName = sanitizeRegex.ReplaceAllString(fileName, "")
	fileName = strings.TrimSpace(fileName)
	return fileName
}

// extractFilename parses the Content-Disposition header to extract the original filename.
func extractFilename(contentDisp string) string {
	if contentDisp == "" {
		return ""
	}
	matches := filenameRegex.FindStringSubmatch(contentDisp)
	if len(matches) > 1 {
		decoded, err := url.QueryUnescape(matches[1])
		if err == nil {
			return decoded
		}
	}
	return ""
}
