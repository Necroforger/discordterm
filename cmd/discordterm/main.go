package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Necroforger/discordterm"
	"github.com/alecthomas/kingpin"
	"github.com/bwmarrin/discordgo"
	. "github.com/logrusorgru/aurora"
)

var (
	app = kingpin.New("Discordterm", "A simple command line client for discord")

	username = app.Flag("username", "Username to log in with").Short('u').String()
	password = app.Flag("password", "password to log in with only provide if you are entering a username as well").Short('p').String()
	token    = app.Flag("token", "A user or bot token to log in with. If using a bot token, remember to prefix it with 'Bot '").Short('t').String()
	args     = app.Arg("args", "String of arguments to log in with. If a user name and password is provided "+
		"Attempt to log in with it, Otherwise will attempt to log in using the first argument as a token").Strings()

	showNicknames = app.Flag("show-nicknames", "Show users' nicknames in place of usernames when possible").Default("true").Bool()
	showImages    = app.Flag("show-images", "Automatically print images").Short('i').Bool()
	imageWidth    = app.Flag("img-width", "Sets the default width of images").Default("100").Uint()
	colorImages   = app.Flag("color-images", "If enabled, images will have color").Bool()
	colorText     = app.Flag("color-text", "If enabled, Text will be colored").Short('c').Default("true").Bool()
)

// MinInt returns the minimum of two integers
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxInt returns the greater of two ints
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

const helpMessage = `====| Commands: |==============================================
/say        say something in the currently active channel
/gl         lists all the available guilds
/cl         lists all the available channels in the selected guild
/leave      leave the current channel to stop listening for messages

/g [n]      selects a guild by index. If no guild is selected,
            print information about the current guild.

/gr [text]  selects a guild by name with the given regular expression
            read the example for "/cr"
       
/c [n]      selects a channel by index. If no channel is selected,
            print information about the current channel

/cr [text]  selects a channel by name with a regular expression
            example: "/cr go_discordgo" will select a channel by the
            name of go_discordgo.                                    
         
/m [n]      retrieves n messages from the active channel's history 
            will retrieve 10 messages if no argument is specified  

/p [line 1] Send a multi-line paragraph to the current channel
            Type /send to send the message or cancel to do nothing

/roles [guildID]    lists the roles in the specified guild, or the current guild.
/upload [path]      Uploads the file located at 'path' to the current channel

/img-auto [off|on]  Auto image will automatically print message images
                    When set to on.

/text-color [off|on] Enable / disable colored text                    

/img-width [width]        Sets the default width of ascii images
/img-color [on|off]       Print images with color or black and white
/img [message id] [width] displays the given message's images
/avatar [userid]          displays the avatar of the given user

/members [lastid]         displays a list of up to 1000 users in your
                          current guild. Call with lastID to retrieve
                          more users.

/presences                displays the list of presences in your current guild
                          call with lastID to retrieve more presences

/member-info [userid]     display information about a particular member in your
                          current guild

/show-nicknames [on|off]  toggle showing users' nicknames in place of their usernames

/username [username]      Set a new username for your account
/status  [online|idle|dnd|invisible|offline] Updates your current online status
/playing   [string]       Set your playing status to the given string
/playing-off              Set your playing status to off
/streaming [string]       // TODO:: Set your streaming status to the given string

/member-add-role [userid] [roleid]     Add role ROLEID to user USERID
/member-remove-role [userid] [roleid]  remove role ROLEID from user USERID
/member-nick [userid] [nickname]       Set a member's nickname in the current guild
/nick [nickname]                       Set your own nickname in the current guild

/delete [messageid]       Deletes the message with the given ID in your active channel
/edit   [messageid]       Edits the message with the given ID in your active channel

/ls [n]     If you are not in a guild, lists guilds
            If you are in a guild but not in a channel, lists channels
            If you are in a channel, lists 'n' messages with 25 being
            the default
            
/cd [i|..]  If you are not in a guild, selects the guild with index i
            If you are in guild, selects the channel with index i
            If you are in a channel and provide .. as an argument,
            leave the channel. If you are in a guild but not a channel
            and provide .. as an argument, leave the guild.
                                           
/help       prints this help menu                                  
/exit       closes this program
================================================================     
`

func isOn(txt string) bool {
	return strings.ToLower(txt) == "on"
}

func isOff(txt string) bool {
	return strings.ToLower(txt) == "off"
}

func formatBoolOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// readInputLoop waits for text and executes
func readInputLoop(dt *discordterm.Client) {
	rd := bufio.NewReader(os.Stdin)
	for {
		// Read a line of input
		line, err := rd.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		// Remove whitespace characters from line
		line = strings.TrimSpace(line)

		err = executeCommand(dt, line)
		if err != nil {
			log.Println(err)
		}
	}
}

// Args provides helper methods for arguments
type Args []string

// Get ...
func (a Args) Get(n int) string {
	if n >= len(a) || n < 0 {
		return ""
	}
	return a[n]
}

// After ...
func (a Args) After(n int) string {
	if n >= 0 && n < len(a) {
		return strings.Join(a[n:], " ")
	}
	return ""
}

// execCmd executes a command
func executeCommand(dt *discordterm.Client, line string) error {
	if strings.HasPrefix(line, "") {
		// Remove the prefix from the command
		line = line[1:]

		// Parse arguments by spaces
		args := Args(strings.Split(line, " "))

		switch args[0] {

		// Guild list
		case "gl", "lg", "guild_list", "guilds":
			for i, g := range dt.Cli.State.Guilds {
				if dt.Conf.ColorText {
					if dt.ActiveGuild() == g.ID {
						fmt.Println(Magenta(i), "\t", Magenta(g.Name))
					} else if n := dt.GuildUnreadMessages(g.ID); n > 0 {
						fmt.Println(Green(i), "\t", Green(g.Name), Red("["+strconv.Itoa(n)+"]"))
					} else {
						fmt.Println(i, "\t", g.Name)
					}
				} else {
					fmt.Println(i, "\t", g.Name)
				}

			}

		// Channel list
		case "cl", "lc", "channel_list", "channels":
			if dt.ActiveGuild() == "" {
				return errors.New("You need to select a guild first")
			}
			channels, err := dt.Cli.GuildChannels(dt.ActiveGuild())
			if err != nil {
				return err
			}
			guild, err := dt.Cli.Guild(dt.ActiveGuild())
			if err != nil {
				return err
			}
			fmt.Println(guild.Name)
			for i, c := range channels {
				if c.Type == discordgo.ChannelTypeGuildVoice {
					continue
				}
				if dt.Conf.ColorText {
					if dt.ActiveChannel() == c.ID {
						fmt.Println(Magenta(i), "\t", Magenta(c.Name))
					} else if n := dt.ChannelUnreadMessages(guild.ID, c.ID); n > 0 {
						fmt.Println(Green(i), "\t", Green(c.Name), Red("["+strconv.Itoa(n)+"]"))
					} else {
						fmt.Println(i, "\t", c.Name)
					}
				} else {
					fmt.Println(i, "\t", c.Name)
				}
			}

		case "ls":
			if dt.ActiveChannel() != "" { // If in a channel list messages
				executeCommand(dt, "/m 25")
			} else if dt.ActiveGuild() != "" { // if in a guild list channels
				executeCommand(dt, "/cl")
			} else { // If not in a guild list guilds
				executeCommand(dt, "/gl")
			}

		case "cd":
			// if the argument is ..
			// Leave the current channel or guild
			if args.Get(1) == ".." {
				if dt.ActiveChannel() != "" {
					dt.SetChannel("")
				} else if dt.ActiveGuild() != "" {
					dt.SetGuild("")
				}
				return nil
			}

			// If in a guild select a channel by index
			if dt.ActiveGuild() != "" {
				executeCommand(dt, "/c "+args.Get(1))
			} else { // if not in a guild select a guild by index
				executeCommand(dt, "/g "+args.Get(1))
			}

		//Select guild by index
		case "g", "guild":
			if args.Get(1) == "" {
				return errors.New("Please select a guild index")
			}
			n, err := strconv.Atoi(args.Get(1))
			if err != nil {
				return err
			}
			if n >= len(dt.Cli.State.Guilds) || n < 0 {
				return errors.New("Index out of bounds")
			}
			// Update ActiveGuild
			dt.SetGuild(dt.Cli.State.Guilds[n].ID)

			fmt.Printf("Selected guild: %s\n", dt.Cli.State.Guilds[n].Name)
			executeCommand(dt, "/c 0")

		// Select guild regex
		case "gr", "guild_regex":
			if args.Get(1) == "" {
				return errors.New("Please provide a regular expression to search with")
			}
			q, err := regexp.Compile(strings.ToLower(args.After(1)))
			if err != nil {
				return err
			}
			for i, g := range dt.Cli.State.Guilds {
				if q.MatchString(strings.ToLower(g.Name)) {
					// Select the index of the matched guild
					// Using the index selection command
					// On the first matched guild
					executeCommand(dt, "/g "+strconv.Itoa(i))
					return nil
				}
			}

		// Select channel
		case "c", "channel":
			if dt.ActiveGuild() == "" {
				return errors.New("You need to select a guild first")
			}
			if args.Get(1) == "" {
				return errors.New("Please select a channel index")
			}
			n, err := strconv.Atoi(args.Get(1))
			if err != nil {
				return err
			}
			channels, err := dt.Cli.GuildChannels(dt.ActiveGuild())
			if err != nil {
				return err
			}
			if n >= len(channels) || n < 0 {
				return errors.New("index out of bounds")
			}
			// Set current channel
			dt.SetChannel(channels[n].ID)
			// Mark channel messages as read
			dt.MarkRead(dt.ActiveGuild(), channels[n].ID)
			fmt.Printf("Selected channel: %s\n", channels[n].Name)

		// Select channel regex
		case "cr", "channel_regex":
			if dt.ActiveGuild() == "" {
				return errors.New("You must be in a guild to use this command")
			}
			if args.Get(1) == "" {
				return errors.New("Please provide a regular expression to search with")
			}
			q, err := regexp.Compile(strings.ToLower(args.After(1)))
			if err != nil {
				return err
			}
			channels, err := dt.Cli.GuildChannels(dt.ActiveGuild())
			if err != nil {
				return err
			}
			for i, c := range channels {
				if q.MatchString(strings.ToLower(c.Name)) {
					// Execute the index selection command
					// On the first matched channel
					executeCommand(dt, "/c "+strconv.Itoa(i))
					return nil
				}
			}

		// Retrieve messages
		case "m", "messages":
			if dt.ActiveChannel() == "" {
				return errors.New("You need to be in a channel to retrieve messages")
			}

			// Set amount of messages to get
			n, err := strconv.Atoi(args.Get(1))
			if err != nil {
				n = 10
			}

			messages, err := dt.Cli.ChannelMessages(dt.ActiveChannel(), n, "", "", "")
			if err != nil {
				return err
			}

			if len(messages) == 0 {
				fmt.Println("No messages to retrieve")
				return nil
			}

			for i := len(messages) - 1; i >= 0; i-- {
				dt.PrintMessage(messages[i])
			}

		// Uploads a file
		case "upload":
			if dt.ActiveChannel() == "" {
				return errors.New("You need to be in a channel to upload a file")
			}
			if args.Get(1) == "" {
				return errors.New("Please enter a file path to upload")
			}

			// Get file stats for name
			finfo, err := os.Stat(args.Get(1))
			if err != nil {
				return err
			}

			f, err := os.Open(args.Get(1))
			if err != nil {
				return err
			}
			defer f.Close()

			dt.Cli.ChannelFileSend(dt.ActiveChannel(), finfo.Name(), f)

		// set the default width of images
		case "img-width":
			if n, err := strconv.Atoi(args.Get(1)); err == nil {
				dt.Conf.ImageWidth = uint(n)
				fmt.Println("Image width set to ", n)
			} else {
				return errors.New("Invalid number")
			}

		// Set the default height of an image
		case "img-height":
			if n, err := strconv.Atoi(args.Get(1)); err == nil {
				dt.Conf.ImageHeight = uint(n)
				fmt.Println("Image height set to ", n)
			} else {
				return errors.New("Invalid number")
			}

		// Automatically display images on new messages
		case "img-auto":
			if args.Get(1) == "" {
				fmt.Println(formatBoolOnOff(dt.Conf.ShowImages))
				return nil
			}
			if isOn(args.Get(1)) {
				dt.Conf.ShowImages = true
				fmt.Println("Now automatically displaying images")
			}
			if isOff(args.Get(1)) {
				dt.Conf.ShowImages = false
				fmt.Println("No longer automatically displaying images")
			}

		case "img-color":
			if args.Get(1) == "" {
				fmt.Println(formatBoolOnOff(dt.Conf.ColorImages))
				return nil
			}
			if isOn(args.Get(1)) {
				dt.Conf.ColorImages = true
				fmt.Println("Images will be rendered in color")
			}
			if isOff(args.Get(1)) {
				dt.Conf.ColorImages = false
				fmt.Println("Images will be rendered in grayscale")
			}

		case "text-color":
			if args.Get(1) == "" {
				fmt.Println(formatBoolOnOff(dt.Conf.ColorText))
				return nil
			}
			if isOn(args.Get(1)) {
				fmt.Println("Text will be colored")
				dt.Conf.ColorText = true
			}
			if isOff(args.Get(1)) {
				fmt.Println("Text will not be colored")
				dt.Conf.ColorText = false
			}

		case "img":
			if dt.ActiveChannel() == "" {
				return errors.New("You need to be in a channel to use this command")
			}
			if args.Get(1) == "" {
				return errors.New("Please provide a message ID")
			}

			var width uint
			if n, err := strconv.Atoi(args.Get(2)); err == nil {
				width = uint(n)
			} else {
				width = dt.Conf.ImageWidth
			}

			messages, err := dt.Cli.ChannelMessages(dt.ActiveChannel(), 100, "", "", "")
			if err != nil {
				return err
			}

			var m *discordgo.Message
			for _, message := range messages {
				if strings.Contains(message.ID, args.Get(1)) {
					m = message
				}
			}
			if m == nil {
				return errors.New("Message ID was not in the past 100 messages, or did not contain the given substring")
			}

			dt.PrintMessageComplex(m, &discordterm.Config{
				ColorText:   dt.Conf.ColorText,
				ImageWidth:  width,
				ImageHeight: 0,
				ShowImages:  true,
				ColorImages: dt.Conf.ColorImages,
			})

		// Prints a user's avatar
		case "avatar":
			if dt.ActiveGuild() == "" {
				return errors.New("You must be in a guild to use this command")
			}
			var avatarURL string
			if args.Get(1) != "" {
				m, err := dt.Cli.GuildMember(dt.ActiveGuild(), args.Get(1))
				if err != nil {
					return err
				}
				avatarURL = m.User.AvatarURL("256")
			} else {
				avatarURL = dt.Cli.State.User.AvatarURL("256")
			}

			var width uint
			if n, err := strconv.Atoi(args.Get(2)); err == nil {
				width = uint(n)
			} else {
				width = dt.Conf.ImageWidth
			}

			if dt.Conf.ColorText {
				fmt.Println(Green(avatarURL))
			} else {
				fmt.Println(avatarURL)
			}

			dt.PrintImageURLComplex(avatarURL, &discordterm.Config{
				ShowImages:  true,
				ColorText:   dt.Conf.ColorText,
				ImageWidth:  width,
				ColorImages: dt.Conf.ColorImages,
			})

		// Write a multiline paragraph
		case "p", "paragraph":
			if dt.ActiveChannel() == "" {
				return errors.New("You need to be in a channel to use this command")
			}

			lines := []string{}
			if args.After(1) != "" {
				lines = append(lines, args.After(1))
			}

			rd := bufio.NewReader(os.Stdin)

			fmt.Println("Enter your paragraph in multiple lines. Type /send or /cancel to finish")
		loop:
			for {
				line := strings.Trim(ReadInputString(rd), "\r\n")
				switch line {
				case "/send":
					_, err := dt.Cli.ChannelMessageSend(dt.ActiveChannel(), strings.Join(lines, "\n"))
					if err != nil {
						return err
					}
					break loop
				case "/cancel":
					break loop
				default:
					lines = append(lines, line)
				}
			}

		// Print a list of guild members
		case "members":
			if dt.ActiveGuild() == "" {
				return errors.New("You need to be in a guild to use this command")
			}

			ms, err := dt.Cli.GuildMembers(dt.ActiveGuild(), args.Get(1), 1000)
			if err != nil {
				return err
			}
			if len(ms) == 0 {
				fmt.Println("No users returned")
				return nil
			}
			for _, m := range ms {
				nicknamePadLeft := strings.Repeat(" ", MaxInt(0, 35-utf8.RuneCountInString(m.User.Username)))
				if dt.Conf.ColorText {
					fmt.Println(Cyan(m.User.ID), "\t", Red(m.User.Username), nicknamePadLeft, Green(m.Nick))
				} else {
					fmt.Println(m.User.ID, "\t", m.User.Username, nicknamePadLeft, m.Nick)
				}
			}

		case "presences":
			if dt.ActiveGuild() == "" {
				return errors.New("You need to be in a guild to use this command")
			}

			guild, err := dt.Cli.State.Guild(dt.ActiveGuild())
			if err != nil {
				return err
			}
			ps := guild.Presences
			if len(ps) == 0 {
				fmt.Println("No users returned")
				return nil
			}
			for _, p := range ps {
				m, err := dt.Cli.State.Member(dt.ActiveGuild(), p.User.ID)
				if err != nil {
					m = &discordgo.Member{
						User: &discordgo.User{
							Username: "-------",
						},
						Nick: "---------",
					}
				}

				var game string
				if p.Game != nil {
					game = p.Game.Name
				}

				nicknamePadLeft := strings.Repeat(" ", MaxInt(0, 35-utf8.RuneCountInString(m.User.Username)))
				statusPadLeft := strings.Repeat(" ", MaxInt(0, 25-utf8.RuneCountInString(m.Nick)))
				gamePadLeft := strings.Repeat(" ", MaxInt(0, 7-len(string(p.Status))))
				if dt.Conf.ColorText {
					fmt.Println(Cyan(p.User.ID), "\t", Red(m.User.Username), nicknamePadLeft, Green(m.Nick), statusPadLeft, discordterm.ColorStatus(string(p.Status)), gamePadLeft, game)
				} else {
					fmt.Println(p.User.ID, "\t", m.User.Username, nicknamePadLeft, m.Nick, statusPadLeft, p.Status, gamePadLeft, game)
				}
			}

		case "delete":
			if dt.ActiveChannel() == "" {
				return errors.New("You need to be in a channel to use this command")
			}
			if args.Get(1) == "" {
				return errors.New("Please provide a message ID as an argument")
			}
			err := dt.Cli.ChannelMessageDelete(dt.ActiveChannel(), args.Get(1))
			if err != nil {
				return err
			}
			// Refresh the message list after deletion
			executeCommand(dt, "/m 25")

		case "edit":
			if dt.ActiveChannel() == "" {
				return errors.New("You need to be in a channel to use this command")
			}
			if args.Get(1) == "" {
				return errors.New("Please specify a message id")
			}
			// Replace the message with the second argument
			_, err := dt.Cli.ChannelMessageEdit(dt.ActiveChannel(), args.Get(1), args.After(2))
			if err != nil {
				return err
			}
			// Refresh the list of messages after editing"
			executeCommand(dt, "/m 25")

		// List the roles in a guild
		case "roles":
			var guildID string
			if id := args.Get(1); id != "" {
				guildID = id
			} else {
				if dt.ActiveGuild() == "" {
					return errors.New("Supply a guildID or enter a guild to use this command")
				}
				guildID = dt.ActiveGuild()
			}

			guild, err := dt.Cli.State.Guild(guildID)
			if err != nil {
				return err
			}
			if len(guild.Roles) == 0 {
				return errors.New("No roles found")
			}
			sort.Sort(discordgo.Roles(guild.Roles))
			for _, role := range guild.Roles {
				if dt.Conf.ColorText {
					fmt.Println(Cyan(role.ID), "\t", Green(role.Name))
				} else {
					fmt.Println(role.ID, "\t", role.Name)
				}
			}

		// Prints various information about a member. Like their nickname and roles
		case "member-info", "m-info":
			if dt.ActiveGuild() == "" {
				return errors.New("You must be in a guild to use this command")
			}

			var userID string
			if args.Get(1) == "" {
				userID = dt.Cli.State.User.ID
			} else {
				userID = args.Get(1)
			}

			member, err := dt.Cli.GuildMember(dt.ActiveGuild(), userID)
			if err != nil {
				return err
			}
			memberRoles := []*discordgo.Role{}
			guild, err := dt.Cli.State.Guild(dt.ActiveGuild())
			if err != nil {
				return err
			}

			// Obtain a list of roles the user has
			for _, mrole := range member.Roles {
				for _, grole := range guild.Roles {
					if mrole == grole.ID {
						memberRoles = append(memberRoles, grole)
					}
				}
			}

			var (
				username      string
				nickname      string
				discriminator string
				avatarURL     string
			)

			if dt.Conf.ColorText {
				username = Red(member.User.Username).String()
				nickname = Green(member.Nick).String()
				discriminator = Cyan(member.User.Discriminator).String()
				avatarURL = Green(member.User.AvatarURL("")).String()
			} else {
				username = member.User.Username
				nickname = member.Nick
				discriminator = member.User.Discriminator
				avatarURL = member.User.AvatarURL("")
			}

			fmt.Println("ID           \t", member.User.ID)
			fmt.Println("Username:    \t", username)
			fmt.Println("Nickname:    \t", nickname)
			fmt.Println("Discriminator\t", discriminator)
			fmt.Println("Avatar URL:  \t", avatarURL)
			if len(memberRoles) > 0 {
				fmt.Println("Roles: ")
				for _, role := range memberRoles {
					if dt.Conf.ColorText {
						fmt.Println("    ", Cyan(role.ID), "\t", Red(role.Name))
					} else {
						fmt.Println("    ", role.ID, "\t", role.Name)
					}
				}
			}

		// Adds a role to a guild member
		case "member-add-role":
			if dt.ActiveGuild() == "" {
				return errors.New("You need to be in a guild to use this command")
			}
			if args.Get(1) == "" || args.Get(2) == "" {
				return errors.New("Please provide a member ID and a role ID")
			}
			err := dt.Cli.GuildMemberRoleAdd(dt.ActiveGuild(), args.Get(1), args.Get(2))
			if err != nil {
				return err
			}
			fmt.Println("Granted role to user")

		// Removes a role from a guild member
		case "member-remove-role":
			if dt.ActiveGuild() == "" {
				return errors.New("You need to be in a guild to use this command")
			}
			if args.Get(1) == "" || args.Get(2) == "" {
				return errors.New("Please provide a memberID and a role ID")
			}
			err := dt.Cli.GuildMemberRoleRemove(dt.ActiveGuild(), args.Get(1), args.Get(2))
			if err != nil {
				return err
			}

		// Set the nickname of another user
		case "member-nick":
			if dt.ActiveGuild() == "" {
				return errors.New("You need to be in a guild to use this command")
			}
			if args.Get(1) == "" {
				return errors.New("You need to enter a userID")
			}
			err := dt.Cli.GuildMemberNickname(dt.ActiveGuild(), args.Get(1), args.Get(2))
			if err != nil {
				return err
			}
			fmt.Println("Nickname set to " + args.Get(2))

		case "show-nicknames", "show-nicks":
			if args.Get(1) == "" {
				fmt.Println(formatBoolOnOff(dt.Conf.ShowNicknames))
				return nil
			}
			if isOn(args.Get(1)) {
				fmt.Println("Nicknames will be displayed")
				dt.Conf.ShowNicknames = true
			} else if isOff(args.Get(1)) {
				fmt.Println("Nicknames will not be displayed")
				dt.Conf.ShowNicknames = false
			}

		// Change your username
		case "username":
			if args.Get(1) == "" {
				return errors.New("Please enter the username you wish to use as an argument")
			}
			_, err := dt.Cli.UserUpdate(*username, *password, args.Get(1), dt.Cli.State.User.Avatar, "")
			if err != nil {
				return err
			}
			fmt.Println("Username changed to " + args.Get(1))

		case "status":
			if args.Get(1) == "" {
				return errors.New("Please enter a status to switch to")
			}
			_, err := dt.Cli.UserUpdateStatus(discordgo.Status(args.Get(1)))
			if err != nil {
				return err
			}

		// Update your playing status
		case "playing":
			err := dt.Cli.UpdateStatus(0, args.Get(1))
			if err != nil {
				return err
			}
			fmt.Println("Playing status set to: ", args.Get(1))

		case "playing-off":
			err := dt.Cli.UpdateStatus(1, "")
			if err != nil {
				return err
			}
			fmt.Println("Playing status set to nothing")

		// Set your own nickname
		case "nick":
			return executeCommand(dt, "/member-nick @me "+args.After(1))

		// leave channel
		case "leave":
			dt.SetChannel("")

		// help
		case "help":
			fmt.Println(helpMessage)

		case "say":
			if dt.ActiveChannel() == "" {
				return errors.New("You are not currently in a channel")
			}
			_, err := dt.Cli.ChannelMessageSend(dt.ActiveChannel(), args.After(1))
			if err != nil {
				return err
			}
			if !dt.Conf.ShowImages {
				executeCommand(dt, "/ls")
			}

		case "exit":
			os.Exit(0)

		}

	} else { 
		// Send a message to the currently active channel
		// if dt.ActiveChannel() == "" {
		// 	return errors.New("You are not currently in a channel")
		// }
		// _, err := dt.Cli.ChannelMessageSend(dt.ActiveChannel(), line)
		// if err != nil {
		// 	return err
		// }

		// // Refresh the window only if images are disabled for
		// // Performance reasons
		// if !dt.Conf.ShowImages {
		// 	executeCommand(dt, "/ls")
		// }
	}
	return nil
}

// Must ...
func Must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// ReadInputString reads a string of input, returning a string, and panics on error
func ReadInputString(rd *bufio.Reader) string {
	line, err := rd.ReadString('\n')
	Must(err)
	return line
}

// QueryInputString queries for a string of input
func QueryInputString(rd *bufio.Reader, query string) string {
	fmt.Println(query)
	fmt.Printf(">")
	return strings.TrimSpace(ReadInputString(rd))
}

// GetLoginInfoFromInput ...
func GetLoginInfoFromInput() {
	rd := bufio.NewReader(os.Stdin)
	mode := QueryInputString(rd, "Login with (t)oken or (u)sername and password")

	if strings.Contains(strings.ToLower(mode), "u") {
		// Log in with username and password
		*username = QueryInputString(rd, "Username: ")
		*password = QueryInputString(rd, "Password: ")
	} else if strings.Contains(strings.ToLower(mode), "t") {
		// Log in with token
		*token = QueryInputString(rd, "Token: ")
	} else {
		fmt.Println("Selected invalid option")
	}
}

func main() {
	app.Parse(os.Args[1:])

	// Create login credentials from arguments
	if len(*args) >= 2 {
		*username = (*args)[0]
		*password = (*args)[1]
	} else if len(*args) == 1 {
		*token = (*args)[0]
	}

	// Request login from user
	if *username == "" && *password == "" && *token == "" {
		GetLoginInfoFromInput()
	}

	session, err := discordgo.New(*username, *password, *token)
	if err != nil {
		log.Fatal(err)
	}
	dt := discordterm.NewClient(session, &discordterm.Config{
		ShowImages:    *showImages,
		ImageWidth:    *imageWidth,
		ColorText:     *colorText,
		ColorImages:   *colorImages,
		ShowNicknames: *showNicknames,
	})

	ready := make(chan bool)
	session.AddHandlerOnce(func(_ *discordgo.Session, _ *discordgo.Ready) {
		ready <- true
	})

	// Open websocket connection
	err = session.Open()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(helpMessage)

	// Wait for ready event to send guild and User info
	// Otherwise the State.User might be nil
	<-ready
	if dt.Cli.State.User != nil {
		fmt.Println("Connected as", dt.Cli.State.User.Username)
	} else {
		fmt.Println("Connected to discord...")
	}

	fmt.Println("Select a guild and channel")

	// Read input and execute commands
	go readInputLoop(dt)

	<-make(chan struct{})
}

//***********+:  -*#**********************************#+===*******************************************
//***********##+:  -*##*##****************######*******===+#******************************************
//*************##*:  -+++===================++++*****#+-==*******##***********************************
//***********#***++------=+=+++++++++++====---=-:::--=-=-=***###*=+##*********************************
//*********#*++==-=++*-  :*##*###########*******++===:==..:---==   :+##*******************************
//*******#*+====+****##*-  :*#******************####*===**+==-:      :+**###**************************
//*****#*====+**#******##*:  -*#******************#+-==*#*####-        .-==+**###*********************
//****#+===+*##**********##+.  =##***************#+===*******#.          .-::-=+**###*****************
//***#+-==+#***************##=. .+#**************+===*****+#**            :++--:::-=+*###*************
//****-=-+#******************#*-  -*#*********#*====****#+=+#*             .*#**+=-:::-=+*###*********
//****==-#*********+++===-==+**#*:  =##******#+===+#*******#++               +#**##*+=-:..:=*###******
//****==-******++++++++++*====#*##=  :*#***#*====*#*******+#%*.               =####*##**+-:  .-+*##***
//***#+-:-*#*==++++***###*---+#***#*:  =##*+===+***********=*##=.              -=++*######**=:   :+*##
//****#=::-====+*####**+-::-+#******#+. .+====*#*********++**##*+=:.                .:=*##**##*=:   -+
//*****#*=:=-.=++++==-:::-+*#********#*:.===****#********#%%#******#=.                  .=*#***##*+:
//******#===-:::::::--=+*##********#*+===- .*#**++**+=*#%#****#%%%#*##-           .        -*#****##*=
//*****#+-==##*******###********#**+=-==**:  =#*+*=.-*%#****%%%%%%%#+**-          .+:        -*#*****#
//*****#+==+#***************##**+==-=+****#=  .+-  =%%*++*#%%%%%#*+++==+=.         :#-        .+#*****
//*******===+*********###***++==-=+**#*****#*.    +@%++==#%%#**+++*##*++++:         -#=         +#****
//********====++******+++===-==+**##********-    =@*-..:%%##+**#*+.=**++++=.         +#-         +#***
//*********++==============++**+*****#****#:     *-  :+*#**=:...:+#.=###+=-+.        :#*.         ****
//*********##***++++++++***#*+++***=:=+*##.      .  *@@+#@:  .:   %+.**+=+##+         +*=         :#**
//**************######*###*++***##*=:+*-=-.*.      #@@@%@@*.   .-*@=-**%#*#*#-        :**.         ***
//***********************++**#+***=-+#=---=+   .  =@@@@@@@@%##%@@#***#%%#*###*         +*:         +#*
//*#******************+++*##**+*#=-+#--===-:.--: .%@#-#@@@@@@@@%***#%%%%%*#***-        -*=         +#*
// =##**************++*****+++=*=-+#-=====-.-+-:.-@@+=+@@@%%@%*+++#%%%%%#++++==        .*=        .***
//: .+##**********++***+++****#*:+#--====:-+==-#*+@@#+=%@@%%@***+#%%%%*+++++-=#:        *=        -#**
//#+. :*#*******++**+++**##****-=#=-====:-##==-##+%@%+=#@@@@@@%+%@%%+-**##+=*##=        +=        ****
//*##=. -*#*#*++*#+++**********-+=----=-=@@#==-##=:%@#=*@@@@@#+%*=::+=-#*==*#*#=        =-  ..::-=*++*
//***#*=  =*++**+=+*#*********##**++=-::+###-==+.. .#@@@@@@@% :.:    % ==*****+=        -==+****##*++*
//*****#*: =+*++++**********#**+++=-++------.=*---   =@@@@@@%:    .-*+-+#%*++++-        =****+++**++**
//******++*++++*************+--=--=+*#=-====.-*---    .-*#%%#%####**=+*#%%+=++=:        ++=**++++*****
//***#*=++==***+***+++***+=---===+*#**=-----:.++::.       .-+*#**+--+#*%%#++===         ==+++++*******
//***=:++=+#*+*+******++=====+***#**#+=+++++=:-*-::         .:---=*+#*%%%*==++         .++==**##******
//*++=***##*++*++*++==--+++=+#*******+----==--:==-=           :*+#****%%%==+*:         :++++++++******
//+**##*****+*++*+---==+++=+*********#*+++=====-=-.         :+#+*%+*++%%#=++=          =++****++++****
//**********+***+:-===+++++*************+==****=-+.     .--+##+*%%++#=+@#=++:          +*+********++**
//************++==+++******+**********+==*#****+-==    .:.+##=+%%%+++-:*#++=          -*++++++++++*++*
//********#*+=++*******++++*++****++==++*******+::+:     =%+:.%%%%#+= .++=+:          =++**++++++++***
//*******+==-=**++++**********+++===+++*#**+++**  =+.   :#:  =%%%#=+.  =+++          -************#***
//******=-===+++*****++++++++====++****#+=+***#: :.==   :    +%%%.:=    +*=          *#***************
//*#**#+:=+=+++++++=======++==+*******#++*****=:+= .+-       -%%=  :    .+=         -#****************
//-+***++****************+*+=********#*+#******#*   :+:      -%*          -=         -#***************
//--==-+*+**++***********++*=+*******= -*##***#*.  =+-+.     *+=     -  .. .-      .  :*#*************
//:.-===++****+++**++***+++*+=*+++++=+=. :+##**..-*##+=+.   .#**=    *- -*. :-    +#=   =#************
//..:+=====++**************++=++++**=*##+: .-**+*#***#=== -: ***#:   ** +#*..-=  +#*#*:  :*#**********
//.:**++==-----==+++++++++===++++++*=+**##*-. :+##*****==--#-****#-  **+**#*:.-**#****#=  .=#*********
//.+*****+++=--:::::---:::=+++++++++=*****##*=. .-*##***==-+#*****#= =#****#+::-*******#*: .-*#*******
//=*****+*+***+++++=====-=+==+++++**=+#******##*-. :+*##*==:*#*****#*+******#+::-#*******#=.:-+#******
//#*****+*++*****************+++++++*=*********##*+: .-+##-:-********#*******#+::=#*******#*:-==******
//+*####*+*+*#**++++++++*#*#**##+- +*+=***********##*=. .=+-=-*#**************#=::+#*******#*====*#***
//==++==+**++*****++*+*+****##+:  :*+*++#************##+-. .:=-*#**************#-::**********#+==-+#**
//:==:...-+*++#*****++*+***#*:   .**++*=***************##*+: .--*##**************::-#*********#+===+**
//::-==-..:=*+*******+++*#*=    .+*+*+*++#****************##*=-::-+###**********#=::+#*********#+====*
//::::-==-..:=*#######****-... :*+++++*+=********************#*-:: .-+###*********-:-***********#*====
//::::::-==-..:-==+++++==:.:.:.=**++++****+++*******************=::.. .-*###*****#+::=#************===
//-:::::::-==-...........:::::::-+************++****************#=:-=+-. .-*#####*#---**************==
//==-----:::-==-::::::::::::::::::=+++***********++**************#+-:-*#+-. .-++++*=.:=#************#+
//=========--:-==-:::::::::::::::::::::--==+++*****++**####*******#*=::=*##=::++====-::*#*************
//======---=+***====-::::::::::::::::::::::::+-=++***++++++*********#+-:-===+*##*=++++==+#************
//===---=+***+=-:. :==-:::::::::::::::::::::++::-*+++**.     -#******#*=:::====----------+#***********
//---=***+=-..       .-=--:::::::::::::::::+*:::+*+*+*=     -**********#-:::::::---------:=+###*******
//=+**+=:.             :-==-::::::::::::::++:::=*+*+*+.   .*#***********==-:::-----------:. .-+*###***
//*+=-.                :::-===-:::::::::-+=:::-*+++**.    =#***********#+---------------:=#*=:. :-+*##
//--- --               ::::::-==--:::::=+-:::=*++**+.     +#*************=:::::-------::-***##*+-. .:=
//+*+-++:              :::::::::-==--:+=:::-+****+-  .    =#*************#+-::::::::::-+*******###*=:.
//##*-++*=            .::::::::::::====:::=**+=-:   +**++*****************#**+===--==+*#***********###
//***=###%+           ..::::::::-=+=:--==---.     -+**####*******************#########****************
//***+####%+         .:::::::-=++=::.   .---:.   .*#**************************************************
//***+%%###%#:      .:...:-=++=:::=+=.. .:++=--::-+**#************************************************
