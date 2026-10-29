package messageslist

import (
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/tview"
)

// stickerRows is how tall stickers are drawn.
const stickerRows = 8

// stickerFormatGIF is the format of an animated GIF sticker, which arikawa has no name for.
const stickerFormatGIF discord.StickerFormatType = 4

// stickerURL returns the URL of the image of sticker, or "" if it is not one, as a Lottie sticker is an animation file.
func stickerURL(sticker discord.StickerItem) discord.URL {
	switch sticker.FormatType {
	case discord.StickerFormatPNG, discord.StickerFormatAPNG:
		return sticker.StickerURLWithType(discord.PNGImage)
	case stickerFormatGIF:
		return sticker.StickerURLWithType(discord.GIFImage)
	}
	return ""
}

// drawStickers names the stickers of message, as the image is not shown before it loads, for Lottie stickers or without previews.
func drawStickers(builder *markdown.Builder, message discord.Message, style tview.Style) {
	for _, sticker := range message.Stickers {
		builder.NewLine()
		builder.Write("Sticker: "+sticker.Name, style)
	}
}
