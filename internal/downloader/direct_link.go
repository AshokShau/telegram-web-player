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
	"slices"
	"time"

	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
)

type directLink struct {
	query string
}

func newDirectLink(query string) *directLink {
	return &directLink{query: query}
}

func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		case ip4[0] == 127:
			return true
		case ip4[0] == 169 && ip4[1] == 254:
			return true
		case ip4[0] == 0:
			return true
		}
	} else if ip16 := ip.To16(); ip16 != nil {
		if (ip16[0] & 0xfe) == 0xfc {
			return true
		}
		if (ip16[0] == 0xfe) && ((ip16[1] & 0xc0) == 0x80) {
			return true
		}
	}
	return false
}

// isValid checks if the query looks like a valid public URL to prevent SSRF.
func (d *directLink) isValid() bool {
	if !strings.HasPrefix(d.query, "http://") && !strings.HasPrefix(d.query, "https://") {
		return false
	}

	u, err := url.Parse(d.query)
	if err != nil {
		return false
	}

	hostname := u.Hostname()
	if hostname == "" {
		return false
	}

	if strings.EqualFold(hostname, "localhost") {
		return false
	}

	ip := net.ParseIP(hostname)
	if ip != nil {
		if isPrivateIP(ip) {
			return false
		}
	} else {
		ips, err := net.LookupIP(hostname)
		if err != nil {
			return false
		}
		if slices.ContainsFunc(ips, isPrivateIP) {
			return false
		}
	}

	return true
}

func (d *directLink) getInfo() (*utils.PlatformTracks, error) {
	if !d.isValid() {
		return nil, errors.New("invalid url")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		d.query,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("invalid or unplayable link: %w", err)
	}

	var info utils.FFProbeFormat
	if err = json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	var duration int32
	if info.Format.Duration != "" {
		if d, err := strconv.ParseFloat(info.Format.Duration, 64); err == nil {
			duration = int32(d)
		}
	}

	title := info.Format.Tags.Title
	if title == "" {
		parts := strings.Split(d.query, "/")
		if len(parts) > 0 {
			title = parts[len(parts)-1]
			title = strings.SplitN(title, "?", 2)[0]
			title = strings.SplitN(title, "#", 2)[0]
			title, _ = url.QueryUnescape(title)
		}
		if title == "" {
			title = "Direct Link"
		}
	}

	const maxTitleLength = 30
	if len(title) > maxTitleLength {
		title = title[:maxTitleLength-3] + "..."
	}

	track := utils.GetUrlTrack{
		Title:    title,
		Duration: duration,
		Url:      d.query,
		Id:       d.query,
		Platform: utils.DirectLink,
	}

	return &utils.PlatformTracks{Results: []utils.GetUrlTrack{track}}, nil
}

func (d *directLink) search() (*utils.PlatformTracks, error) {
	return d.getInfo()
}

func (d *directLink) getTrack() (*utils.TrackInfo, error) {
	info, err := d.getInfo()
	if err != nil {
		return nil, err
	}

	if len(info.Results) == 0 {
		return nil, errors.New("no track found")
	}

	track := info.Results[0]
	return &utils.TrackInfo{
		Id:       track.Id,
		URL:      track.Url,
		CdnURL:   track.Url,
		Platform: track.Platform,
	}, nil
}

func (d *directLink) downloadTrack(_ *utils.TrackInfo, _ bool) (string, error) {
	return d.query, nil
}
