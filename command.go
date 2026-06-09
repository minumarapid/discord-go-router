package dgr

import (
	"reflect"

	"github.com/bwmarrin/discordgo"
)

func Selected[T any](structPtr *T) *Choice {
	v := reflect.ValueOf(structPtr).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		if field.Kind() == reflect.Bool && field.Bool() {
			return field.Addr().Interface().(*Choice)
		}
	}
	return nil
}

func RegSlash[T any](d *Dgr, name string, description string, handler func(c *Context[T])) {
	var tmp T
	t := reflect.TypeOf(tmp)
	options := make([]*discordgo.ApplicationCommandOption, 0)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		cmdTag := field.Tag.Get("dgr")
		if cmdTag == "" {
			continue
		}

		var optType discordgo.ApplicationCommandOptionType
		var choices []*discordgo.ApplicationCommandOptionChoice

		// 💡 構造体のフィールド型を直接チェックする
		fieldType := field.Type

		switch {
		// 1. 【追加】ユーザー型（統合型）
		case fieldType == reflect.TypeOf(&InteractionUser{}):
			optType = discordgo.ApplicationCommandOptionUser

		// 2. 【追加】チャンネル型
		case fieldType == reflect.TypeOf(&discordgo.Channel{}):
			optType = discordgo.ApplicationCommandOptionChannel

		// 3. 【追加】ロール型
		case fieldType == reflect.TypeOf(&discordgo.Role{}):
			optType = discordgo.ApplicationCommandOptionRole

		// 4. 【追加】添付ファイル型
		case fieldType == reflect.TypeOf(&discordgo.MessageAttachment{}):
			optType = discordgo.ApplicationCommandOptionAttachment

		case fieldType == reflect.TypeOf(&Mentionable{}):
			optType = discordgo.ApplicationCommandOptionMentionable

		// 5. 既存のベース型判定
		case fieldType.Kind() == reflect.Int || fieldType.Kind() == reflect.Int64:
			optType = discordgo.ApplicationCommandOptionInteger
		case fieldType.Kind() == reflect.Float32 || fieldType.Kind() == reflect.Float64:
			optType = discordgo.ApplicationCommandOptionNumber
		case fieldType.Kind() == reflect.Bool:
			optType = discordgo.ApplicationCommandOptionBoolean
		case fieldType.Kind() == reflect.Struct: // 前回作ったChoices用
			optType = discordgo.ApplicationCommandOptionString
			choices = make([]*discordgo.ApplicationCommandOptionChoice, 0)
			structType := field.Type
			for j := 0; j < structType.NumField(); j++ {
				subField := structType.Field(j)
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
					Name:  subField.Name,
					Value: subField.Name,
				})
			}
		default:
			optType = discordgo.ApplicationCommandOptionString
		}

		options = append(options, &discordgo.ApplicationCommandOption{
			Type:        optType,
			Name:        cmdTag,
			Description: field.Tag.Get("desc"),
			Required:    field.Tag.Get("required") == "true",
			Choices:     choices,
		})
	}

	d.commands = append(d.commands, &discordgo.ApplicationCommand{
		Name:        name,
		Description: description,
		Type:        discordgo.ChatApplicationCommand,
		Options:     options,
	})

	d.interactionHandlers[name] = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		var args T

		cmdData := i.ApplicationCommandData()
		sentOptions := cmdData.Options
		resolved := cmdData.Resolved // 💡 Discordから届いた実体データ（Resolved）

		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
		for _, opt := range sentOptions {
			optionMap[opt.Name] = opt
		}

		v := reflect.ValueOf(&args).Elem()
		typeOfArgs := v.Type()

		for j := 0; j < typeOfArgs.NumField(); j++ {
			fieldT := typeOfArgs.Field(j)
			fieldV := v.Field(j)

			tag := fieldT.Tag.Get("dgr")
			if tag == "" {
				continue
			}

			if opt, ok := optionMap[tag]; ok {
				fieldType := fieldT.Type

				if resolved == nil {
					resolved = &discordgo.ApplicationCommandInteractionDataResolved{}
				}

				// 💡 受信時も型ごとに Resolved からデータを引き抜いて Set する
				switch {
				case fieldType == reflect.TypeOf(&InteractionUser{}):
					userID := opt.Value.(string) // 💡 修正：生のインターフェースから string (ID) をアサーション

					memberData := resolved.Members[userID]
					if memberData == nil {
						memberData = &discordgo.Member{}
					}

					combined := &InteractionUser{
						User:   resolved.Users[userID],
						Member: memberData,
					}
					fieldV.Set(reflect.ValueOf(combined))

				case fieldType == reflect.TypeOf(&discordgo.Channel{}):
					chID := opt.Value.(string) // 💡 修正
					if ch, ok := resolved.Channels[chID]; ok {
						fieldV.Set(reflect.ValueOf(ch))
					}

				case fieldType == reflect.TypeOf(&discordgo.Role{}):
					roleID := opt.Value.(string) // 💡 修正
					if role, ok := resolved.Roles[roleID]; ok {
						fieldV.Set(reflect.ValueOf(role))
					}

				case fieldType == reflect.TypeOf(&discordgo.MessageAttachment{}):
					attachID := opt.Value.(string) // 💡 修正
					if attach, ok := resolved.Attachments[attachID]; ok {
						fieldV.Set(reflect.ValueOf(attach))
					}

				case fieldType == reflect.TypeOf(&Mentionable{}):
					id := opt.Value.(string) // 💡 修正
					m := &Mentionable{}

					if u, ok := resolved.Users[id]; ok {
						memberData := resolved.Members[id]
						if memberData == nil {
							memberData = &discordgo.Member{}
						}
						m.User = &InteractionUser{
							User:   u,
							Member: memberData,
						}
						m.Type = MentionableTypeUser
					} else if r, ok := resolved.Roles[id]; ok {
						m.Role = r
						m.Type = MentionableTypeRole
					}

					fieldV.Set(reflect.ValueOf(m))

				// 既存のベース型マッピング
				default:
					switch fieldV.Kind() {
					case reflect.String:
						fieldV.SetString(opt.StringValue())
					case reflect.Int, reflect.Int64:
						fieldV.SetInt(opt.IntValue())
					case reflect.Float32, reflect.Float64:
						fieldV.SetFloat(opt.FloatValue())
					case reflect.Bool:
						fieldV.SetBool(opt.BoolValue())
					case reflect.Struct:
						selectedValue := opt.StringValue()
						structFieldV := fieldV.FieldByName(selectedValue)
						if structFieldV.IsValid() && structFieldV.Kind() == reflect.Bool {
							structFieldV.SetBool(true)
						}
					}
				}
			}
		}

		handler(&Context[T]{
			Session:     s,
			Interaction: i.Interaction,
			Args:        args,
			dgr:         d,
		})
	}
}

func RegMessageCtx(d *Dgr, name string, handler func(c *Context[discordgo.Message])) {
	d.commands = append(d.commands, &discordgo.ApplicationCommand{
		Name: name,
		Type: discordgo.MessageApplicationCommand,
	})

	d.interactionHandlers[name] = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		data := i.ApplicationCommandData()

		var targetMessage discordgo.Message

		if data.Resolved != nil && data.Resolved.Messages != nil {
			if msg, ok := data.Resolved.Messages[data.TargetID]; ok {
				targetMessage = *msg
			}
		}

		handler(&Context[discordgo.Message]{
			Session:     s,
			Interaction: i.Interaction,
			Args:        targetMessage,
			dgr:         d,
		})
	}
}

func RegUserCtx(d *Dgr, name string, handler func(c *Context[discordgo.User])) {
	d.commands = append(d.commands, &discordgo.ApplicationCommand{
		Name: name,
		Type: discordgo.UserApplicationCommand,
	})

	d.interactionHandlers[name] = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		data := i.ApplicationCommandData()

		var targetUser discordgo.User

		if data.Resolved != nil && data.Resolved.Users != nil {
			if user, ok := data.Resolved.Users[data.TargetID]; ok {
				targetUser = *user
			}
		}

		handler(&Context[discordgo.User]{
			Session:     s,
			Interaction: i.Interaction,
			Args:        targetUser,
			dgr:         d,
		})
	}
}
