package dgr

import (
	"fmt"
	"reflect"

	"github.com/bwmarrin/discordgo"
)

var (
	choiceType           = reflect.TypeOf(Choice(false))
	interactionUserPtr   = reflect.TypeOf((*InteractionUser)(nil))
	channelPtr           = reflect.TypeOf((*discordgo.Channel)(nil))
	rolePtr              = reflect.TypeOf((*discordgo.Role)(nil))
	messageAttachmentPtr = reflect.TypeOf((*discordgo.MessageAttachment)(nil))
	mentionablePtr       = reflect.TypeOf((*Mentionable)(nil))
)

type SelectedChoice struct {
	Choice *Choice
	Name   string
	Value  string
}

func Selected[T any](structPtr *T) *Choice {
	selected := SelectedChoiceOf(structPtr)
	if selected == nil {
		return nil
	}
	return selected.Choice
}

func SelectedChoiceOf[T any](structPtr *T) *SelectedChoice {
	if structPtr == nil {
		return nil
	}

	v := reflect.ValueOf(structPtr)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldT := v.Type().Field(i)

		if field.Type() == choiceType && field.Bool() && field.CanAddr() && field.Addr().CanInterface() {
			spec := choiceSpecFromField(fieldT)
			return &SelectedChoice{
				Choice: field.Addr().Interface().(*Choice),
				Name:   spec.Name,
				Value:  spec.Value,
			}
		}
	}
	return nil
}

func RegSlash[T any](d *Dgr, name string, description string, handler func(c *Context[T])) {
	if err := RegSlashE(d, name, description, handler); err != nil {
		panic(err)
	}
}

func RegSlashE[T any](d *Dgr, name string, description string, handler func(c *Context[T])) error {
	if d == nil {
		return fmt.Errorf("dgr: nil router")
	}
	if handler == nil {
		return fmt.Errorf("dgr: nil slash command handler for %q", name)
	}

	var tmp T
	t := reflect.TypeOf(tmp)
	if t == nil || t.Kind() != reflect.Struct {
		return fmt.Errorf("dgr: slash command args for %q must be a struct, got %v", name, t)
	}

	options := make([]*discordgo.ApplicationCommandOption, 0)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		cmdTag := field.Tag.Get("dgr")
		if cmdTag == "" {
			continue
		}

		var optType discordgo.ApplicationCommandOptionType
		var choices []*discordgo.ApplicationCommandOptionChoice

		fieldType := field.Type

		switch {
		case fieldType == interactionUserPtr:
			optType = discordgo.ApplicationCommandOptionUser

		case fieldType == channelPtr:
			optType = discordgo.ApplicationCommandOptionChannel

		case fieldType == rolePtr:
			optType = discordgo.ApplicationCommandOptionRole

		case fieldType == messageAttachmentPtr:
			optType = discordgo.ApplicationCommandOptionAttachment

		case fieldType == mentionablePtr:
			optType = discordgo.ApplicationCommandOptionMentionable

		case fieldType.Kind() == reflect.Int || fieldType.Kind() == reflect.Int64:
			optType = discordgo.ApplicationCommandOptionInteger
		case fieldType.Kind() == reflect.Float32 || fieldType.Kind() == reflect.Float64:
			optType = discordgo.ApplicationCommandOptionNumber
		case fieldType.Kind() == reflect.Bool:
			optType = discordgo.ApplicationCommandOptionBoolean
		case fieldType.Kind() == reflect.Struct:
			optType = discordgo.ApplicationCommandOptionString
			choices = make([]*discordgo.ApplicationCommandOptionChoice, 0)
			structType := field.Type
			for j := 0; j < structType.NumField(); j++ {
				subField := structType.Field(j)
				if subField.Type != choiceType {
					continue
				}
				spec := choiceSpecFromField(subField)
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
					Name:  spec.Name,
					Value: spec.Value,
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

	d.mu.Lock()
	defer d.mu.Unlock()
	d.initLocked()

	d.commands = append(d.commands, &discordgo.ApplicationCommand{
		Name:        name,
		Description: description,
		Type:        discordgo.ChatApplicationCommand,
		Options:     options,
	})

	d.interactionHandlers[name] = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		args := parseSlashArgs[T](i)

		handler(&Context[T]{
			Session:     s,
			Interaction: i.Interaction,
			Args:        args,
			dgr:         d,
		})
	}

	return nil
}

func RegMessageCtx(d *Dgr, name string, handler func(c *Context[discordgo.Message])) {
	_ = RegMessageCtxE(d, name, handler)
}

func RegMessageCtxE(d *Dgr, name string, handler func(c *Context[discordgo.Message])) error {
	if d == nil {
		return fmt.Errorf("dgr: nil router")
	}
	if handler == nil {
		return fmt.Errorf("dgr: nil message context handler for %q", name)
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.initLocked()

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

	return nil
}

func RegUserCtx(d *Dgr, name string, handler func(c *Context[discordgo.User])) {
	_ = RegUserCtxE(d, name, handler)
}

func RegUserCtxE(d *Dgr, name string, handler func(c *Context[discordgo.User])) error {
	if d == nil {
		return fmt.Errorf("dgr: nil router")
	}
	if handler == nil {
		return fmt.Errorf("dgr: nil user context handler for %q", name)
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.initLocked()

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

	return nil
}

func parseSlashArgs[T any](i *discordgo.InteractionCreate) T {
	var args T
	if i == nil {
		return args
	}

	cmdData := i.ApplicationCommandData()
	resolved := cmdData.Resolved
	if resolved == nil {
		resolved = &discordgo.ApplicationCommandInteractionDataResolved{}
	}

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(cmdData.Options))
	for _, opt := range cmdData.Options {
		if opt != nil {
			optionMap[opt.Name] = opt
		}
	}

	v := reflect.ValueOf(&args).Elem()
	if v.Kind() != reflect.Struct {
		return args
	}
	typeOfArgs := v.Type()

	for j := 0; j < typeOfArgs.NumField(); j++ {
		fieldT := typeOfArgs.Field(j)
		fieldV := v.Field(j)

		tag := fieldT.Tag.Get("dgr")
		if tag == "" || !fieldV.CanSet() {
			continue
		}

		opt := optionMap[tag]
		if opt == nil {
			continue
		}

		setOptionValue(fieldT.Type, fieldV, opt, resolved)
	}

	return args
}

func setOptionValue(fieldType reflect.Type, fieldV reflect.Value, opt *discordgo.ApplicationCommandInteractionDataOption, resolved *discordgo.ApplicationCommandInteractionDataResolved) {
	switch fieldType {
	case interactionUserPtr:
		userID := opt.StringValue()
		user := resolved.Users[userID]
		member := resolved.Members[userID]
		if user != nil || member != nil {
			fieldV.Set(reflect.ValueOf(&InteractionUser{User: user, Member: member}))
		}

	case channelPtr:
		if ch := resolved.Channels[opt.StringValue()]; ch != nil {
			fieldV.Set(reflect.ValueOf(ch))
		}

	case rolePtr:
		if role := resolved.Roles[opt.StringValue()]; role != nil {
			fieldV.Set(reflect.ValueOf(role))
		}

	case messageAttachmentPtr:
		if attach := resolved.Attachments[opt.StringValue()]; attach != nil {
			fieldV.Set(reflect.ValueOf(attach))
		}

	case mentionablePtr:
		if mentionable := resolvedMentionable(opt.StringValue(), resolved); mentionable != nil {
			fieldV.Set(reflect.ValueOf(mentionable))
		}

	default:
		setScalarOrChoiceValue(fieldV, opt)
	}
}

func resolvedMentionable(id string, resolved *discordgo.ApplicationCommandInteractionDataResolved) *Mentionable {
	if u := resolved.Users[id]; u != nil {
		return &Mentionable{
			User: &InteractionUser{
				User:   u,
				Member: resolved.Members[id],
			},
			Type: MentionableTypeUser,
		}
	}
	if r := resolved.Roles[id]; r != nil {
		return &Mentionable{
			Role: r,
			Type: MentionableTypeRole,
		}
	}
	return nil
}

func setScalarOrChoiceValue(fieldV reflect.Value, opt *discordgo.ApplicationCommandInteractionDataOption) {
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
		setChoiceValue(fieldV, opt.StringValue())
	}
}

type choiceSpec struct {
	Name  string
	Value string
}

func choiceSpecFromField(field reflect.StructField) choiceSpec {
	spec := choiceSpec{
		Name:  field.Name,
		Value: field.Name,
	}

	if tag := field.Tag.Get("dgr"); tag != "" {
		spec.Name = tag
		spec.Value = tag
	}
	if tag := field.Tag.Get("name"); tag != "" {
		spec.Name = tag
	}
	if tag := field.Tag.Get("label"); tag != "" {
		spec.Name = tag
	}
	if tag := field.Tag.Get("value"); tag != "" {
		spec.Value = tag
	}

	return spec
}

func setChoiceValue(fieldV reflect.Value, selectedValue string) {
	fieldType := fieldV.Type()
	for i := 0; i < fieldV.NumField(); i++ {
		choiceFieldV := fieldV.Field(i)
		choiceFieldT := fieldType.Field(i)
		if choiceFieldV.Type() != choiceType || !choiceFieldV.CanSet() {
			continue
		}

		spec := choiceSpecFromField(choiceFieldT)
		if selectedValue == spec.Value || selectedValue == choiceFieldT.Name {
			choiceFieldV.SetBool(true)
			return
		}
	}
}
