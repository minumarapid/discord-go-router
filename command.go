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

type SlashTarget interface {
	slashTarget()
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

func RegSlash[T any](target SlashTarget, name string, description string, handler func(c *Context[T])) {
	if err := RegSlashE(target, name, description, handler); err != nil {
		panic(err)
	}
}

func RegSlashE[T any](target SlashTarget, name string, description string, handler func(c *Context[T])) error {
	switch t := target.(type) {
	case *Dgr:
		return regRootSlashE(t, name, description, handler)
	case *CommandGroup:
		return regGroupSlashE(t, name, description, handler)
	case *SubCommandGroup:
		return regSubGroupSlashE(t, name, description, handler)
	default:
		return fmt.Errorf("dgr: unsupported slash command target %T", target)
	}
}

func (*Dgr) slashTarget() {}

func regRootSlashE[T any](d *Dgr, name string, description string, handler func(c *Context[T])) error {
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

	options, err := slashOptionsFromType(t)
	if err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.initLocked()

	if _, exists := d.interactionHandlers[name]; exists {
		return fmt.Errorf("dgr: command %q is already registered", name)
	}

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

type CommandGroup struct {
	d       *Dgr
	command *discordgo.ApplicationCommand
}

func (*CommandGroup) slashTarget() {}

type SubCommandGroup struct {
	d       *Dgr
	command *discordgo.ApplicationCommand
	option  *discordgo.ApplicationCommandOption
}

func (*SubCommandGroup) slashTarget() {}

func Group(d *Dgr, name string, description string) *CommandGroup {
	group, err := GroupE(d, name, description)
	if err != nil {
		panic(err)
	}
	return group
}

func GroupE(d *Dgr, name string, description string) (*CommandGroup, error) {
	if d == nil {
		return nil, fmt.Errorf("dgr: nil router")
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.initLocked()

	if _, exists := d.interactionHandlers[name]; exists {
		return nil, fmt.Errorf("dgr: command %q is already registered", name)
	}

	command := &discordgo.ApplicationCommand{
		Name:        name,
		Description: description,
		Type:        discordgo.ChatApplicationCommand,
		Options:     []*discordgo.ApplicationCommandOption{},
	}
	d.commands = append(d.commands, command)
	d.interactionHandlers[name] = groupInteractionHandler(d)

	return &CommandGroup{d: d, command: command}, nil
}

func SubGroup(group *CommandGroup, name string, description string) *SubCommandGroup {
	subGroup, err := SubGroupE(group, name, description)
	if err != nil {
		panic(err)
	}
	return subGroup
}

func SubGroupE(group *CommandGroup, name string, description string) (*SubCommandGroup, error) {
	if group == nil || group.d == nil || group.command == nil {
		return nil, fmt.Errorf("dgr: nil command group")
	}

	group.d.mu.Lock()
	defer group.d.mu.Unlock()
	group.d.initLocked()

	for _, opt := range group.command.Options {
		if opt.Name == name {
			return nil, fmt.Errorf("dgr: subcommand or group %q is already registered in %q", name, group.command.Name)
		}
	}

	option := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
		Name:        name,
		Description: description,
		Options:     []*discordgo.ApplicationCommandOption{},
	}
	group.command.Options = append(group.command.Options, option)

	return &SubCommandGroup{
		d:       group.d,
		command: group.command,
		option:  option,
	}, nil
}

func regGroupSlashE[T any](g *CommandGroup, name string, description string, handler func(c *Context[T])) error {
	if g == nil || g.d == nil || g.command == nil {
		return fmt.Errorf("dgr: nil command group")
	}
	if handler == nil {
		return fmt.Errorf("dgr: nil subcommand handler for %q", name)
	}

	var tmp T
	t := reflect.TypeOf(tmp)
	if t == nil || t.Kind() != reflect.Struct {
		return fmt.Errorf("dgr: subcommand args for %q must be a struct, got %v", name, t)
	}

	options, err := slashOptionsFromType(t)
	if err != nil {
		return err
	}

	g.d.mu.Lock()
	defer g.d.mu.Unlock()
	g.d.initLocked()

	for _, opt := range g.command.Options {
		if opt.Name == name {
			return fmt.Errorf("dgr: subcommand %q is already registered in %q", name, g.command.Name)
		}
	}

	g.command.Options = append(g.command.Options, &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        name,
		Description: description,
		Options:     options,
	})

	handlerKey := groupHandlerKey(g.command.Name, name)
	g.d.interactionHandlers[handlerKey] = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		data := i.ApplicationCommandData()
		if len(data.Options) == 0 || data.Options[0] == nil {
			return
		}

		args := parseSlashArgsFromOptions[T](data.Options[0].Options, data.Resolved)
		handler(&Context[T]{
			Session:     s,
			Interaction: i.Interaction,
			Args:        args,
			dgr:         g.d,
		})
	}

	return nil
}

func regSubGroupSlashE[T any](g *SubCommandGroup, name string, description string, handler func(c *Context[T])) error {
	if g == nil || g.d == nil || g.command == nil || g.option == nil {
		return fmt.Errorf("dgr: nil subcommand group")
	}
	if handler == nil {
		return fmt.Errorf("dgr: nil subcommand handler for %q", name)
	}

	var tmp T
	t := reflect.TypeOf(tmp)
	if t == nil || t.Kind() != reflect.Struct {
		return fmt.Errorf("dgr: subcommand args for %q must be a struct, got %v", name, t)
	}

	options, err := slashOptionsFromType(t)
	if err != nil {
		return err
	}

	g.d.mu.Lock()
	defer g.d.mu.Unlock()
	g.d.initLocked()

	for _, opt := range g.option.Options {
		if opt.Name == name {
			return fmt.Errorf("dgr: subcommand %q is already registered in %q %q", name, g.command.Name, g.option.Name)
		}
	}

	g.option.Options = append(g.option.Options, &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        name,
		Description: description,
		Options:     options,
	})

	handlerKey := subGroupHandlerKey(g.command.Name, g.option.Name, name)
	g.d.interactionHandlers[handlerKey] = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		data := i.ApplicationCommandData()
		subcommand := selectedSubcommandOption(data.Options)
		if subcommand == nil {
			return
		}

		args := parseSlashArgsFromOptions[T](subcommand.Options, data.Resolved)
		handler(&Context[T]{
			Session:     s,
			Interaction: i.Interaction,
			Args:        args,
			dgr:         g.d,
		})
	}

	return nil
}

func groupInteractionHandler(d *Dgr) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		data := i.ApplicationCommandData()
		if len(data.Options) == 0 || data.Options[0] == nil {
			return
		}

		key := interactionHandlerKey(data.Name, data.Options[0])
		d.mu.RLock()
		handler := d.interactionHandlers[key]
		d.mu.RUnlock()
		if handler != nil {
			handler(s, i)
		}
	}
}

func interactionHandlerKey(commandName string, opt *discordgo.ApplicationCommandInteractionDataOption) string {
	if opt.Type != discordgo.ApplicationCommandOptionSubCommandGroup || len(opt.Options) == 0 || opt.Options[0] == nil {
		return groupHandlerKey(commandName, opt.Name)
	}
	return subGroupHandlerKey(commandName, opt.Name, opt.Options[0].Name)
}

func selectedSubcommandOption(options []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.ApplicationCommandInteractionDataOption {
	if len(options) == 0 || options[0] == nil {
		return nil
	}
	if options[0].Type != discordgo.ApplicationCommandOptionSubCommandGroup {
		return options[0]
	}
	if len(options[0].Options) == 0 {
		return nil
	}
	return options[0].Options[0]
}

func groupHandlerKey(groupName string, subcommandName string) string {
	return groupName + " " + subcommandName
}

func subGroupHandlerKey(groupName string, subGroupName string, subcommandName string) string {
	return groupName + " " + subGroupName + " " + subcommandName
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
	return parseSlashArgsFromOptions[T](cmdData.Options, cmdData.Resolved)
}

func parseSlashArgsFromOptions[T any](options []*discordgo.ApplicationCommandInteractionDataOption, resolved *discordgo.ApplicationCommandInteractionDataResolved) T {
	var args T
	if resolved == nil {
		resolved = &discordgo.ApplicationCommandInteractionDataResolved{}
	}

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
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

func slashOptionsFromType(t reflect.Type) ([]*discordgo.ApplicationCommandOption, error) {
	options := make([]*discordgo.ApplicationCommandOption, 0)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		cmdTag := field.Tag.Get("dgr")
		if cmdTag == "" {
			continue
		}

		optType, choices := slashOptionType(field.Type)
		options = append(options, &discordgo.ApplicationCommandOption{
			Type:        optType,
			Name:        cmdTag,
			Description: field.Tag.Get("desc"),
			Required:    field.Tag.Get("required") == "true",
			Choices:     choices,
		})
	}

	return options, nil
}

func slashOptionType(fieldType reflect.Type) (discordgo.ApplicationCommandOptionType, []*discordgo.ApplicationCommandOptionChoice) {
	switch {
	case fieldType == interactionUserPtr:
		return discordgo.ApplicationCommandOptionUser, nil
	case fieldType == channelPtr:
		return discordgo.ApplicationCommandOptionChannel, nil
	case fieldType == rolePtr:
		return discordgo.ApplicationCommandOptionRole, nil
	case fieldType == messageAttachmentPtr:
		return discordgo.ApplicationCommandOptionAttachment, nil
	case fieldType == mentionablePtr:
		return discordgo.ApplicationCommandOptionMentionable, nil
	case fieldType.Kind() == reflect.Int || fieldType.Kind() == reflect.Int64:
		return discordgo.ApplicationCommandOptionInteger, nil
	case fieldType.Kind() == reflect.Float32 || fieldType.Kind() == reflect.Float64:
		return discordgo.ApplicationCommandOptionNumber, nil
	case fieldType.Kind() == reflect.Bool:
		return discordgo.ApplicationCommandOptionBoolean, nil
	case fieldType.Kind() == reflect.Struct:
		return discordgo.ApplicationCommandOptionString, choicesFromType(fieldType)
	default:
		return discordgo.ApplicationCommandOptionString, nil
	}
}

func choicesFromType(t reflect.Type) []*discordgo.ApplicationCommandOptionChoice {
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0)
	for j := 0; j < t.NumField(); j++ {
		subField := t.Field(j)
		if subField.Type != choiceType {
			continue
		}
		spec := choiceSpecFromField(subField)
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  spec.Name,
			Value: spec.Value,
		})
	}
	return choices
}

func setOptionValue(fieldType reflect.Type, fieldV reflect.Value, opt *discordgo.ApplicationCommandInteractionDataOption, resolved *discordgo.ApplicationCommandInteractionDataResolved) {
	switch fieldType {
	case interactionUserPtr:
		if resolved == nil {
			return
		}
		userID, ok := optionResolvedID(opt)
		if !ok {
			return
		}
		user := resolved.Users[userID]
		member := resolved.Members[userID]
		if user != nil || member != nil {
			fieldV.Set(reflect.ValueOf(&InteractionUser{User: user, Member: member}))
		}

	case channelPtr:
		if resolved == nil {
			return
		}
		id, ok := optionResolvedID(opt)
		if !ok {
			return
		}
		if ch := resolved.Channels[id]; ch != nil {
			fieldV.Set(reflect.ValueOf(ch))
		}

	case rolePtr:
		if resolved == nil {
			return
		}
		id, ok := optionResolvedID(opt)
		if !ok {
			return
		}
		if role := resolved.Roles[id]; role != nil {
			fieldV.Set(reflect.ValueOf(role))
		}

	case messageAttachmentPtr:
		if resolved == nil {
			return
		}
		id, ok := optionResolvedID(opt)
		if !ok {
			return
		}
		if attach := resolved.Attachments[id]; attach != nil {
			fieldV.Set(reflect.ValueOf(attach))
		}

	case mentionablePtr:
		if resolved == nil {
			return
		}
		id, ok := optionResolvedID(opt)
		if !ok {
			return
		}
		if mentionable := resolvedMentionable(id, resolved); mentionable != nil {
			fieldV.Set(reflect.ValueOf(mentionable))
		}

	default:
		setScalarOrChoiceValue(fieldV, opt)
	}
}

func optionResolvedID(opt *discordgo.ApplicationCommandInteractionDataOption) (string, bool) {
	if opt == nil {
		return "", false
	}
	id, ok := opt.Value.(string)
	return id, ok
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
