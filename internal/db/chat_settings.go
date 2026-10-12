package db

import "ashokshau/tg-web/internal/utils"

type ChatSettings struct {
	RestrictedPlay    bool   `json:"restrictedPlay"`
	AdminMode         string `json:"adminMode"`
	CmdDelete         bool   `json:"cmdDelete"`
	AutoDelete        bool   `json:"autoDelete"`
	AutoDeleteMinutes int    `json:"autoDeleteMinutes"`
	SilentMode        bool   `json:"silentMode"`
	Language          string `json:"language"`
}

func (db *Database) GetChatSettings(chatID int64) (*ChatSettings, error) {
	if chatID >= 0 {
		return nil, nil
	}
	chat, err := db.getChat(chatID)
	if err != nil {
		return nil, err
	}
	settings := &ChatSettings{AdminMode: utils.Everyone, AutoDeleteMinutes: DefaultAutoDeleteMinutes, Language: "en"}
	if chat != nil {
		settings.RestrictedPlay, settings.CmdDelete = chat.AdminPlay, chat.CmdDelete
		settings.AutoDelete, settings.SilentMode = chat.AutoDelete, chat.SilentMode
		if chat.AdminMode == utils.Admins {
			settings.AdminMode = utils.Admins
		}
		if ValidateAutoDeleteMinutes(chat.AutoDeleteMinutes) == nil {
			settings.AutoDeleteMinutes = chat.AutoDeleteMinutes
		}
	}
	settings.Language, err = db.GetLanguage(chatID)
	if err != nil {
		return nil, err
	}
	return settings, nil
}
