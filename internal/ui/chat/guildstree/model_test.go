package guildstree

import (
	"slices"
	"testing"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/state/store/defaultstore"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview/tree"
)

func TestModelMoveDMToFront(t *testing.T) {
	t.Run("moves rendered channel to front", func(t *testing.T) {
		firstID := discord.ChannelID(1)
		secondID := discord.ChannelID(2)
		thirdID := discord.ChannelID(3)

		first := tree.NewNode("first").SetReference(firstID)
		second := tree.NewNode("second").SetReference(secondID)
		third := tree.NewNode("third").SetReference(thirdID)
		dmRoot := tree.NewNode("Direct Messages").SetChildren([]*tree.Node{first, second, third})
		m := Model{
			dmRootNode: dmRoot,
			nodes: map[discord.Snowflake]*tree.Node{
				discord.Snowflake(firstID):  first,
				discord.Snowflake(secondID): second,
				discord.Snowflake(thirdID):  third,
			},
		}

		m.moveDMToFront(thirdID)

		want := []*tree.Node{third, first, second}
		if !slices.Equal(dmRoot.Children(), want) {
			t.Fatalf("children = %v, want %v", dmRoot.Children(), want)
		}
	})

	t.Run("ignores unrendered channel", func(t *testing.T) {
		firstID := discord.ChannelID(1)
		first := tree.NewNode("first").SetReference(firstID)
		dmRoot := tree.NewNode("Direct Messages").SetChildren([]*tree.Node{first})
		m := Model{
			dmRootNode: dmRoot,
			nodes:      map[discord.Snowflake]*tree.Node{},
		}

		m.moveDMToFront(discord.ChannelID(2))

		children := dmRoot.Children()
		if len(children) != 1 || children[0] != first {
			t.Fatalf("children = %v, want the original DM node", children)
		}
	})
}

func TestAdjacentNode(t *testing.T) {
	a, b, c := tree.NewNode("a"), tree.NewNode("b"), tree.NewNode("c")
	root := tree.NewNode("").SetChildren([]*tree.Node{a, tree.NewNode("guild").AddChild(b), c})
	match := func(n *tree.Node) bool { return n == a || n == b }
	for _, test := range []struct {
		name          string
		current       *tree.Node
		after, before *tree.Node
	}{
		{"none", nil, a, b},
		{"a", a, b, b},
		{"b", b, a, a},
		{"c", c, a, b},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Run("after", func(t *testing.T) {
				if got := adjacentNode(root, test.current, match, false); got != test.after {
					t.Errorf("got %v, want %v", got, test.after)
				}
			})
			t.Run("before", func(t *testing.T) {
				if got := adjacentNode(root, test.current, match, true); got != test.before {
					t.Errorf("got %v, want %v", got, test.before)
				}
			})
		})
	}
	t.Run("no match", func(t *testing.T) {
		if got := adjacentNode(root, a, func(*tree.Node) bool { return false }, false); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
}

func TestMentionsLabel(t *testing.T) {
	for _, test := range []struct {
		name  string
		count int
		want  string
	}{
		{"none", 0, ""},
		{"some", 5, " (5)"},
		{"most counted", maxMentions, " (99)"},
		{"more", maxMentions + 1, " (99+)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := mentionsLabel(test.count); got != test.want {
				t.Errorf("mentionsLabel(%d) = %q, want %q", test.count, got, test.want)
			}
		})
	}
}

func TestModelMentions(t *testing.T) {
	s := ningen.FromState(state.NewWithStore("", defaultstore.New()))
	s.State.Call(&gateway.ReadyEvent{ReadStates: []gateway.ReadState{
		{ChannelID: 1, LastMessageID: 10, MentionCount: 2},
		{ChannelID: 2, LastMessageID: 20, MentionCount: 3},
	}})
	m := Model{state: s}
	channels := []discord.Channel{{ID: 1}, {ID: 2}, {ID: 3}}
	if got := m.mentions(channels...); got != 5 {
		t.Errorf("mentions = %d, want 5", got)
	}
}

func TestModelCreateChannelNodesThreads(t *testing.T) {
	const (
		me       = discord.UserID(1)
		guildID  = discord.GuildID(10)
		category = discord.ChannelID(20)
		text     = discord.ChannelID(21)
		forum    = discord.ChannelID(22)
		thread   = discord.ChannelID(30)
		post     = discord.ChannelID(31)
		archived = discord.ChannelID(32)
		oldPost  = discord.ChannelID(33)
		stale    = discord.ChannelID(34)
	)

	st := state.NewWithStore("", defaultstore.New())
	s := ningen.FromState(st)
	if err := st.Cabinet.MyselfSet(discord.User{ID: me}, true); err != nil {
		t.Fatal(err)
	}
	if err := st.Cabinet.GuildSet(&discord.Guild{ID: guildID, OwnerID: me}, false); err != nil {
		t.Fatal(err)
	}
	if err := st.Cabinet.RoleSet(guildID, &discord.Role{ID: discord.RoleID(guildID)}, false); err != nil {
		t.Fatal(err)
	}
	if err := st.Cabinet.MemberSet(guildID, &discord.Member{User: discord.User{ID: me}}, false); err != nil {
		t.Fatal(err)
	}

	recent := discord.MessageID(discord.NewSnowflake(time.Now().Add(-time.Hour)))
	longAgo := discord.MessageID(discord.NewSnowflake(time.Now().Add(-2 * threadInactivity)))

	// Threads have no position, so sorting puts them before the channels they are in.
	channels := []discord.Channel{
		{ID: thread, GuildID: guildID, ParentID: text, Type: discord.GuildPublicThread, LastMessageID: recent},
		{ID: post, GuildID: guildID, ParentID: forum, Type: discord.GuildPublicThread, LastMessageID: recent},
		{ID: oldPost, GuildID: guildID, ParentID: forum, Type: discord.GuildPublicThread, LastMessageID: recent, ThreadMetadata: &discord.ThreadMetadata{Archived: true}},
		{ID: archived, GuildID: guildID, ParentID: text, Type: discord.GuildPublicThread, LastMessageID: recent, ThreadMetadata: &discord.ThreadMetadata{Archived: true}},
		{ID: stale, GuildID: guildID, ParentID: text, Type: discord.GuildPublicThread, LastMessageID: longAgo},
		{ID: category, GuildID: guildID, Type: discord.GuildCategory},
		{ID: text, GuildID: guildID, ParentID: category, Type: discord.GuildText, Position: 1},
		{ID: forum, GuildID: guildID, ParentID: category, Type: discord.GuildForum, Position: 2},
	}
	for _, channel := range channels {
		if err := st.Cabinet.ChannelSet(&channel, false); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.Default()
	m := Model{root: tree.NewNode(""), cfg: &cfg, state: s, nodes: make(map[discord.Snowflake]*tree.Node)}
	m.createChannelNodes(m.root, channels)

	if node := m.nodes[discord.Snowflake(thread)]; node == nil {
		t.Error("thread is missing from the tree")
	} else if !slices.Contains(m.nodes[discord.Snowflake(text)].Children(), node) {
		t.Error("thread is not under the channel it is in")
	}
	if m.nodes[discord.Snowflake(post)] != nil {
		t.Error("post of a forum is in the tree before the forum is selected")
	}
	if m.nodes[discord.Snowflake(archived)] != nil {
		t.Error("archived thread is in the tree")
	}
	if m.nodes[discord.Snowflake(stale)] != nil {
		t.Error("thread without a message for more than a day is in the tree")
	}
	if m.expandable(m.nodes[discord.Snowflake(text)]) {
		t.Error("selecting a channel with threads expands it instead of opening it")
	}
	if !m.expandable(m.nodes[discord.Snowflake(category)]) {
		t.Error("selecting a category does not expand it")
	}

	textNode := m.nodes[discord.Snowflake(text)]
	textNode.Collapse()
	m.selectNode(textNode)
	if !textNode.Expanded() {
		t.Error("selecting a collapsed channel does not bring its threads back")
	}

	forumNode := m.nodes[discord.Snowflake(forum)]
	m.expandForum(forumNode, channels[len(channels)-1])
	if m.nodes[discord.Snowflake(post)] == nil {
		t.Error("post of a forum is not in the tree after selecting the forum")
	}
	if m.nodes[discord.Snowflake(oldPost)] != nil {
		t.Error("archived post of a forum is in the tree")
	}
}
