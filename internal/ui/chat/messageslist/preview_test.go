package messageslist

import (
	"image"
	"slices"
	"strings"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/ui/emojis"
	"github.com/ayn2op/discordo/internal/ui/images"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
)

func TestModelPreviewSources(t *testing.T) {
	for _, tt := range []struct {
		name    string
		message discord.Message
		want    []previewSource
	}{
		{"file attachment", discord.Message{Attachments: []discord.Attachment{{Proxy: "a"}}}, nil},
		{"embed image over thumbnail", discord.Message{Embeds: []discord.Embed{{
			Type:      discord.ImageEmbed,
			Image:     &discord.EmbedImage{Proxy: "i", Width: 2, Height: 1},
			Thumbnail: &discord.EmbedThumbnail{Proxy: "t", Width: 2, Height: 1},
		}}}, []previewSource{{"i", 2, 1}}},
		{"gif thumbnail", discord.Message{Embeds: []discord.Embed{{
			Type:      discord.GIFVEmbed,
			Thumbnail: &discord.EmbedThumbnail{Proxy: "t", Width: 2, Height: 1},
		}}}, []previewSource{{"t", 2, 1}}},
		{"article thumbnail", discord.Message{Embeds: []discord.Embed{{
			Type:      discord.ArticleEmbed,
			Thumbnail: &discord.EmbedThumbnail{Proxy: "t", Width: 2, Height: 1},
		}}}, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ml := newTestModel(t, testConfig(t))
			if got := ml.previewSources(tt.message); !slices.Equal(got, tt.want) {
				t.Fatalf("previewSources() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("embeds disabled", func(t *testing.T) {
		ml := newTestModel(t, testConfig(t))
		ml.cfg.UI.MessagesList.Embeds.Enabled = false
		message := discord.Message{
			Attachments: []discord.Attachment{{Proxy: "a", Width: 2, Height: 1}},
			Embeds: []discord.Embed{{
				Type:  discord.ImageEmbed,
				Image: &discord.EmbedImage{Proxy: "i", Width: 2, Height: 1},
			}},
		}
		want := []previewSource{{"a", 2, 1}}
		if got := ml.previewSources(message); !slices.Equal(got, want) {
			t.Fatalf("previewSources() = %v, want %v", got, want)
		}
	})
}

func TestModelEmojiLine(t *testing.T) {
	ml := newTestModel(t, testConfig(t))
	t.Run("not loaded", func(t *testing.T) {
		if got := ml.emojiLine("5"); got != nil {
			t.Fatalf("line = %v, want nil before the emoji is loaded", got)
		}
	})
	t.Run("loaded", func(t *testing.T) {
		ml.images = map[discord.URL]tviewimage.Widget{emojis.URLFromID(5): tviewimage.New(image.NewAlpha(image.Rect(0, 0, 2, 2))).Width(2)}
		if got := ml.emojiLine("5"); len(got) != 2 {
			t.Fatalf("line = %v, want the two cells of the emoji", got)
		}
	})
}

func TestModelLoadImages(t *testing.T) {
	ml := newTestModel(t, testConfig(t))
	ml.setMessages([]discord.Message{{ID: 2, ChannelID: 1, Content: "hi <:cat:5>"}})
	msg := ml.LoadImages()().(images.RequestedMsg)
	t.Run("content emoji", func(t *testing.T) {
		if rows := msg.Squares[emojis.URLFromID(5)]; rows != emojis.Rows {
			t.Fatalf("the emoji in the content was asked for %d rows tall, want %d", rows, emojis.Rows)
		}
	})
	t.Run("keeps the parsed content", func(t *testing.T) {
		if ml.items[len(ml.items)-1].root == nil {
			t.Fatal("the parsed content was not kept for rendering")
		}
	})
}

func TestModelLoadImagesStickers(t *testing.T) {
	message := discord.Message{ID: 2, ChannelID: 1, Stickers: []discord.StickerItem{
		{ID: 10, Name: "png", FormatType: discord.StickerFormatPNG},
		{ID: 11, Name: "apng", FormatType: discord.StickerFormatAPNG},
		{ID: 12, Name: "lottie", FormatType: discord.StickerFormatLottie},
		{ID: 13, Name: "gif", FormatType: stickerFormatGIF},
	}}
	for _, tt := range []struct {
		name    string
		preview bool
		want    []discord.URL
	}{
		{"preview", true, []discord.URL{
			"https://cdn.discordapp.com/stickers/10.png",
			"https://cdn.discordapp.com/stickers/11.png",
			"https://cdn.discordapp.com/stickers/13.gif",
		}},
		{"no preview", false, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ml := newTestModel(t, testConfig(t))
			ml.cfg.UI.MessagesList.Attachments.Preview = tt.preview
			ml.cfg.UI.MessagesList.Avatars.Preview = false
			ml.setMessages([]discord.Message{message})

			var got []discord.URL
			if cmd := ml.LoadImages(); cmd != nil {
				for url, rows := range cmd().(images.RequestedMsg).Squares {
					if rows != stickerRows {
						t.Fatalf("%s was asked for %d rows tall, want %d", url, rows, stickerRows)
					}
					got = append(got, url)
				}
			}
			slices.Sort(got)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("asked for %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelDrawStickers(t *testing.T) {
	ml := newTestModel(t, testConfig(t))
	message := discord.Message{ID: 2, ChannelID: 1, Stickers: []discord.StickerItem{{ID: 12, Name: "wave", FormatType: discord.StickerFormatLottie}}}
	var got string
	for _, span := range ml.renderMessage(message, nil, tview.Style{}, false) {
		got += span.Text
	}
	if !strings.Contains(got, "Sticker: wave") {
		t.Fatalf("rendered %q, want it to name the sticker", got)
	}
}
