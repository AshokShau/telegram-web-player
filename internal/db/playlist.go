/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package db

import (
	"ashokshau/tg-web/internal/utils"
	"context"
	"crypto/rand"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Song represents a single song in a playlist.
type Song struct {
	URL      string `json:"url" bson:"url"`
	Name     string `json:"name" bson:"name"`
	TrackID  string `json:"track_id" bson:"track_id"`
	Duration int32  `json:"duration" bson:"duration"`
	Platform string `json:"platform" bson:"platform"`
}

// Playlist represents a user's playlist.
type Playlist struct {
	ID     string `bson:"_id" json:"id"`
	Name   string `bson:"name" json:"name"`
	UserID int64  `bson:"user_id" json:"user_id"`
	Songs  []Song `bson:"songs" json:"songs"`
}

// generateUniquePlaylistID generates a unique ID for a playlist.
func generateUniquePlaylistID() string {
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	return fmt.Sprintf("tgpl_%x", b)
}

// CreatePlaylist creates a new playlist for a user.
func (db *Database) CreatePlaylist(name string, userID int64) (string, error) {
	ctx, cancel := db.ctx()
	defer cancel()

	id := generateUniquePlaylistID()
	playlist := Playlist{
		ID:     id,
		Name:   name,
		UserID: userID,
		Songs:  []Song{},
	}
	_, err := db.playlistDB.InsertOne(ctx, playlist)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetPlaylist retrieves a playlist by its ID.
func (db *Database) GetPlaylist(id string) (*Playlist, error) {
	ctx, cancel := db.ctx()
	defer cancel()

	var playlist Playlist
	err := db.playlistDB.FindOne(ctx, bson.M{"_id": id}).Decode(&playlist)
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}

// DeletePlaylist deletes a playlist by its ID.
func (db *Database) DeletePlaylist(id string, userID int64) error {
	ctx, cancel := db.ctx()
	defer cancel()

	_, err := db.playlistDB.DeleteOne(ctx, bson.M{"_id": id, "user_id": userID})
	return err
}

// RenamePlaylist updates the name of a playlist.
func (db *Database) RenamePlaylist(id string, name string, userID int64) error {
	ctx, cancel := db.ctx()
	defer cancel()

	_, err := db.playlistDB.UpdateOne(
		ctx,
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": bson.M{"name": name}},
	)
	return err
}

func (db *Database) songExists(id string, trackID string) bool {
	ctx, cancel := db.ctx()
	defer cancel()

	var playlist Playlist
	err := db.playlistDB.FindOne(ctx, bson.M{"_id": id}).Decode(&playlist)
	if err != nil {
		return false
	}

	for _, song := range playlist.Songs {
		if song.TrackID == trackID {
			return true
		}
	}
	return false
}

// AddSongToPlaylist adds a song to a playlist.
func (db *Database) AddSongToPlaylist(id string, song Song, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("unauthorized: valid user ID required")
	}

	if db.songExists(id, song.TrackID) {
		return nil
	}

	ctx, cancel := db.ctx()
	defer cancel()

	filter := bson.M{"_id": id, "user_id": userID}

	res, err := db.playlistDB.UpdateOne(
		ctx,
		filter,
		bson.M{"$push": bson.M{"songs": song}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("playlist not found or unauthorized")
	}
	return nil
}

// RemoveSongFromPlaylist removes a song from a playlist by its track ID.
func (db *Database) RemoveSongFromPlaylist(id string, trackID string, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("unauthorized: valid user ID required")
	}

	if !db.songExists(id, trackID) {
		return fmt.Errorf("track with ID %s not found in playlist", trackID)
	}

	ctx, cancel := db.ctx()
	defer cancel()

	filter := bson.M{"_id": id, "user_id": userID}

	res, err := db.playlistDB.UpdateOne(
		ctx,
		filter,
		bson.M{"$pull": bson.M{"songs": bson.M{"track_id": trackID}}},
	)

	if err != nil {
		return fmt.Errorf("error removing song: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("playlist not found or unauthorized")
	}

	return nil
}

// GetUserPlaylists retrieves all playlists for a user.
func (db *Database) GetUserPlaylists(userID int64) ([]Playlist, error) {
	ctx, cancel := db.ctx()
	defer cancel()

	var playlists []Playlist
	cursor, err := db.playlistDB.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		_ = cursor.Close(ctx)
	}(cursor, ctx)

	for cursor.Next(ctx) {
		var playlist Playlist
		if err := cursor.Decode(&playlist); err != nil {
			return nil, err
		}
		playlists = append(playlists, playlist)
	}
	return playlists, nil
}

func ConvertSongsToTracks(songs []Song) []utils.GetUrlTrack {
	tracks := make([]utils.GetUrlTrack, 0, len(songs))

	for _, song := range songs {
		tracks = append(tracks, utils.GetUrlTrack{
			Url:      song.URL,
			Title:    song.Name,
			Id:       song.TrackID,
			Duration: song.Duration,
			Platform: song.Platform,
		})
	}

	return tracks
}
