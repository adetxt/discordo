package stickerspicker

import "github.com/ayn2op/arikawa/v3/discord"

// ShowMsg lists the stickers to send in GuildID.
type ShowMsg struct {
	GuildID discord.GuildID
}

// SelectedMsg asks to send the sticker with StickerID.
type SelectedMsg struct {
	StickerID discord.StickerID
}

type CancelMsg struct{}
