package dgr

import "github.com/bwmarrin/discordgo"

type Choice bool

type InteractionUser struct {
	*discordgo.User
	*discordgo.Member
}

type Mentionable struct {
	User *InteractionUser
	Role *discordgo.Role
	Type MentionableType
}

type MentionableType string

const (
	MentionableTypeUser MentionableType = "USER"
	MentionableTypeRole MentionableType = "ROLE"
)
