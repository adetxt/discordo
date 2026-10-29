package stickerspicker

import (
	"testing"

	"github.com/ayn2op/discordo/internal/config"
)

func TestModelLoaded(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}

	m := NewModel(cfg, nil)
	m.guildID = 1
	t.Run("stale guild", func(t *testing.T) {
		got, _ := m.Update(loadedMsg{guildID: 2, guild: []sticker{{id: 1, name: "a"}}})
		if len(got.guild) != 0 {
			t.Fatalf("guild = %v, want the stickers of another guild dropped", got.guild)
		}
	})
	t.Run("guild before packs, packs kept", func(t *testing.T) {
		got, _ := m.Update(loadedMsg{guildID: 1, guild: []sticker{{id: 1, name: "g"}}, packs: []sticker{{id: 2, name: "p"}}})
		got, _ = got.Update(loadedMsg{guildID: 1, guild: []sticker{{id: 1, name: "g"}}})
		if items := got.items(); len(items) != 2 || items[0].FilterText != "g " || items[1].FilterText != "p " {
			t.Fatalf("items = %v, want the guild's sticker then the pack's", items)
		}
	})
}
