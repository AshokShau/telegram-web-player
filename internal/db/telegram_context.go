package db

import (
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrTelegramChatInstanceConflict = errors.New("telegram_chat_instance_conflict")

func (db *Database) GetTelegramChatInstance(chatID int64) (string, error) {
	if db == nil || db.chatCache == nil || db.chatDB == nil || chatID >= 0 {
		return "", errors.New("telegram_chat_binding_unavailable")
	}

	if chat, ok := db.chatCache.Get(toKey(chatID)); ok && chat != nil && chat.WebAppChatInstance != "" {
		return chat.WebAppChatInstance, nil
	}
	ctx, cancel := db.ctx()
	defer cancel()
	var chat Chats
	err := db.chatDB.FindOne(ctx, bson.M{"_id": chatID}).Decode(&chat)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	db.chatCache.Set(toKey(chatID), &chat)
	return chat.WebAppChatInstance, nil
}

func (db *Database) BindTelegramChatInstance(chatID int64, instance string) error {
	if db == nil || db.chatCache == nil || db.chatDB == nil || chatID >= 0 || instance == "" {
		return errors.New("invalid_telegram_chat_binding")
	}
	if chat, ok := db.chatCache.Get(toKey(chatID)); ok && chat != nil && chat.WebAppChatInstance != "" {
		if chat.WebAppChatInstance != instance {
			return ErrTelegramChatInstanceConflict
		}
		return nil
	}
	ctx, cancel := db.ctx()
	defer cancel()
	_, err := db.chatDB.UpdateOne(ctx,
		bson.M{"_id": chatID, "$or": bson.A{
			bson.M{"webapp_chat_instance": bson.M{"$exists": false}},
			bson.M{"webapp_chat_instance": ""},
			bson.M{"webapp_chat_instance": instance},
		}},
		bson.M{"$set": bson.M{"webapp_chat_instance": instance}},
		options.UpdateOne().SetUpsert(true),
	)
	if mongo.IsDuplicateKeyError(err) {
		return ErrTelegramChatInstanceConflict
	}
	if err == nil {
		db.chatCache.Delete(toKey(chatID))
	}
	return err
}
