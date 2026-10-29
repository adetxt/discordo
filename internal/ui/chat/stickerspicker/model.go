package stickerspicker

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	uipicker "github.com/ayn2op/discordo/internal/ui/picker"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/picker"
	"github.com/ayn2op/tview/text"
)

type Model struct {
	uipicker.Model
	state *ningen.State
	// guildID is the guild the stickers were asked for, and guild its stickers.
	guildID discord.GuildID
	guild   []sticker
	// packs are the stickers of the standard packs, which are fetched once.
	packs []sticker
}

// sticker is a sticker to send and the name it is shown under.
type sticker struct {
	id   discord.StickerID
	name string
	tags string
}

// loadedMsg is the stickers fetched for guildID, and packs unless they were fetched before.
type loadedMsg struct {
	guildID discord.GuildID
	guild   []sticker
	packs   []sticker
}

func NewModel(cfg *config.Config, state *ningen.State) Model {
	return Model{Model: uipicker.NewModel(cfg, "Stickers"), state: state}
}

// items returns the stickers of the guild and then those of the packs as the picker's items.
func (m Model) items() picker.Items {
	items := make(picker.Items, 0, len(m.guild)+len(m.packs))
	for _, sticker := range slices.Concat(m.guild, m.packs) {
		items = append(items, picker.Item{Content: text.New(sticker.name), FilterText: sticker.name + " " + sticker.tags, Reference: sticker.id})
	}
	return items
}

var _ tview.Model[Model] = Model{}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case ShowMsg:
		m.guildID, m.guild = msg.GuildID, nil
		m.SetItems(m.items())
		return m, m.load(msg.GuildID, len(m.packs) == 0)
	case loadedMsg:
		if msg.guildID != m.guildID {
			return m, nil
		}

		m.guild = msg.guild
		if msg.packs != nil {
			m.packs = msg.packs
		}
		m.ReplaceItems(m.items())
		return m, nil
	}

	var cmd tview.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}

// load returns the command that fetches the stickers of the guild, if it is one, and those of the packs if withPacks is set.
func (m Model) load(guildID discord.GuildID, withPacks bool) tview.Cmd {
	return func() tview.Msg {
		msg := loadedMsg{guildID: guildID}
		if guildID.IsValid() {
			var stickers []discord.Sticker
			if err := m.state.RequestJSONInto(&stickers, http.MethodGet, api.EndpointGuilds+guildID.String()+"/stickers"); err != nil {
				slog.Error("failed to get the stickers of the guild", "guild_id", guildID, "err", err)
			}

			for _, s := range stickers {
				if s.Available {
					msg.guild = append(msg.guild, sticker{s.ID, s.Name, s.Tags})
				}
			}
		}

		if withPacks {
			var packs struct {
				Packs []struct {
					Name     string            `json:"name"`
					Stickers []discord.Sticker `json:"stickers"`
				} `json:"sticker_packs"`
			}
			if err := m.state.RequestJSONInto(&packs, http.MethodGet, api.Endpoint+"sticker-packs"); err != nil {
				slog.Error("failed to get the sticker packs", "err", err)
			}

			for _, pack := range packs.Packs {
				for _, s := range pack.Stickers {
					msg.packs = append(msg.packs, sticker{s.ID, s.Name + " (" + pack.Name + ")", strings.TrimSpace(s.Tags)})
				}
			}
		}

		return msg
	}
}

// View shows the picker in a box titled Stickers.
func (m Model) View() tview.Widget {
	return m.Model.View(func(item picker.Item) tview.Msg {
		id, ok := item.Reference.(discord.StickerID)
		if !ok {
			return nil
		}
		return SelectedMsg{StickerID: id}
	}, CancelMsg{})
}
