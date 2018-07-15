package discordterm

import (
	"errors"
	"fmt"
	"github.com/Necroforger/textify"
	"github.com/bwmarrin/discordgo"
	. "github.com/logrusorgru/aurora"
	"image"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

// MinInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxInt returns the greater of two ints
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ColorStatus colours a status string
func ColorStatus(status string) string {
	switch status {
	case "online":
		return Green(status).String()
	case "offline", "invisible":
		return status
	case "dnd":
		return Red(status).String()
	case "away":
		return Brown(status).String()
	default:
		return status
	}
}

// Client is a discordterm client
type Client struct {
	sync.Mutex
	Cli *discordgo.Session
	// ActiveGuild stores the currently selected guild
	activeGuild string
	// ActiveChannel stores the currently selected channel
	activeChannel string

	// UnreadChannels is a map[guildid][channelid] that tracks
	// The count is incremented every time a message is recieved
	// In a channel other than the currently active one
	UnreadChannels map[string]map[string]int

	Conf *Config
}

// Config is the configuration object
type Config struct {
	// Color the text output
	ColorText bool

	// Image options
	ColorImages bool
	ShowImages  bool
	ImageWidth  uint
	ImageHeight uint

	// Show users' nicknames in the chat
	ShowNicknames bool
}

// NewConfig returns the default config
func NewConfig() *Config {
	conf := &Config{
		ColorText:   true,
		ColorImages: true,
		ShowImages:  true,
		ImageWidth:  100,
	}
	return conf
}

// NewClient returns a new client
func NewClient(s *discordgo.Session, conf *Config) *Client {
	if conf == nil {
		conf = NewConfig()
	}
	c := &Client{
		Cli:            s,
		Conf:           conf,
		UnreadChannels: map[string]map[string]int{},
	}
	c.addHandlers()
	return c
}

// PrintImageComplex accepts a config struct
func (c *Client) PrintImageComplex(img image.Image, conf *Config) error {
	if conf == nil {
		conf = NewConfig()
	}
	opts := textify.NewOptions()
	opts.Width = conf.ImageWidth
	opts.Height = conf.ImageHeight
	opts.Palette = textify.PaletteReverse[1:]
	opts.Resize = true

	if c.Conf.ColorImages {
		opts.ColorMode = textify.ColorTerminal
	}

	err := textify.NewEncoder(os.Stdout).Encode(img, opts)
	if err != nil {
		return err
	}

	return nil
}

// PrintImage prints an image to the terminal screen with the client settings
func (c *Client) PrintImage(img image.Image) error {
	return c.PrintImageComplex(img, c.Conf)
}

// PrintImageURLComplex accepts a config struct
func (c *Client) PrintImageURLComplex(path string, conf *Config) error {
	resp, err := http.Get(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return err
	}

	return c.PrintImageComplex(img, conf)
}

// PrintImageURL prints an image from URL
func (c *Client) PrintImageURL(path string) error {
	return c.PrintImageURLComplex(path, c.Conf)
}

// PrintEmbeds prints message embeds
func (c *Client) PrintEmbeds(embeds []*discordgo.MessageEmbed, conf *Config) {
	for _, em := range embeds {

		var frameWidth int
		frameWidth = maxInt(frameWidth, len(em.Title))
		frameWidth = maxInt(frameWidth, len(strings.Split(em.Description, "\n")[0]))
		for _, v := range em.Fields {
			frameWidth = maxInt(frameWidth, len(v.Name))
			frameWidth = maxInt(frameWidth, len(strings.Split(v.Value, "\n")[0]))
		}

		// Determine the length of the border considering images and
		// Image URLs
		if conf.ShowImages {
			if em.Image != nil || em.Thumbnail != nil {
				frameWidth = maxInt(frameWidth, int(conf.ImageWidth))
			}
		} else {
			if em.Image != nil {
				frameWidth = maxInt(frameWidth, len(em.Image.URL))
			}
			if em.Thumbnail != nil {
				frameWidth = maxInt(frameWidth, len(em.Thumbnail.URL))
			}
		}

		// Draw frame top border
		fmt.Println("|", strings.Repeat("=", frameWidth), "|")

		// Print title
		if em.Title != "" {
			if conf.ColorText {
				fmt.Println(Red(em.Title))
			} else {
				fmt.Println(em.Title)
			}
		}

		// Print description
		if em.Description != "" {
			fmt.Println(em.Description)
		}

		// Display embed image
		if em.Image != nil {
			if em.Image.URL != "" {
				if conf.ShowImages {
					// print image
					err := c.PrintImageURLComplex(em.Image.URL, conf)
					if err != nil {
						log.Println(err)
					}
				} else {
					// print image URL
					if c.Conf.ColorText {
						fmt.Println(Green(em.Image.URL))
					} else {
						fmt.Println(em.Image.URL)
					}
				}
			}
		}

		// Display thumbnail image
		if em.Thumbnail != nil {
			if em.Thumbnail.URL != "" {
				if conf.ShowImages {
					// Print image
					err := c.PrintImageURLComplex(em.Thumbnail.URL, conf)
					if err != nil {
						log.Println(err)
					}
				} else {
					// Print image URL
					if c.Conf.ColorText {
						fmt.Println(Green(em.Thumbnail.URL))
					} else {
						fmt.Println(em.Thumbnail.URL)
					}
				}
			}
		}
		for _, v := range em.Fields {
			fmt.Println("|-", v.Name, strings.Repeat("-", frameWidth-len(v.Name)-2))
			fmt.Println(v.Value)
		}

		// Draw frame bottom border
		fmt.Println("|", strings.Repeat("_", frameWidth), "|")
	}
}

// PrintAttachments prints message attachments
func (c *Client) PrintAttachments(attachments []*discordgo.MessageAttachment, conf *Config) {
	for _, a := range attachments {
		if a.URL != "" && a.Filename != "" {
			if conf.ColorText {
				fmt.Println(Green(a.Filename), " \t", Green(a.URL))
			} else {
				fmt.Println(a.Filename, " \t", a.URL)
			}
			if conf.ShowImages {
				c.PrintImageURLComplex(a.URL, conf)
			}
		}
	}
}

func (c *Client) PrintMessageComplex(m *discordgo.Message, conf *Config) {
	if conf == nil {
		conf = NewConfig()
	}

	var displayName string

	// Fetch user nickname or use regular username
	if !conf.ShowNicknames || func() error {
		channel, err := c.Cli.State.Channel(m.ChannelID)
		if err != nil {
			channel, err = c.Cli.Channel(m.ChannelID)
			if err != nil {
				return err
			}
		}
		member, err := c.Cli.State.Member(channel.GuildID, m.Author.ID)
		if err != nil { /*
				member, err = c.Cli.GuildMember(channel.GuildID, m.Author.ID)
				if err != nil {
					return err
				}
				// add member to state to prevent future API requests
				err := c.Cli.State.MemberAdd(member)
				if err != nil {
					log.Println(err)
					return err
				}*/
			return err
		}
		if member.Nick != "" {
			displayName = member.Nick
			return nil
		}
		return errors.New("User has no nickname")
	}() != nil {
		displayName = m.Author.Username
	}

	paddingUseridLeft := strings.Repeat(" ", maxInt(0, 30-len(displayName)))

	if conf.ColorText {
		fmt.Println(Cyan(displayName), paddingUseridLeft, Blue(m.ID), "\t", Blue(m.Author.ID))
	} else {
		fmt.Println(displayName, paddingUseridLeft, "\t", m.Author.ID)
	}
	if m.Content != "" {
		fmt.Println(m.ContentWithMentionsReplaced())
	}

	c.PrintAttachments(m.Attachments, conf)
	c.PrintEmbeds(m.Embeds, conf)

	// Separate messages with a new line
	fmt.Println()
}

// PrintMessage prints a message to the console
func (c *Client) PrintMessage(m *discordgo.Message) {
	c.PrintMessageComplex(m, c.Conf)
}

func (c *Client) addHandlers() {
	c.Cli.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		channel, err := c.Cli.State.Channel(m.ChannelID)
		if err != nil {
			log.Println(err)
			return
		}
		guild, err := c.Cli.State.Guild(channel.GuildID)
		if err != nil { // Message is probably a private message
			return
		}
		if m.ChannelID == c.ActiveChannel() {
			c.PrintMessage(m.Message)
		} else {
			// Add 1 unread message to the unread message counter
			c.MarkUnread(guild.ID, channel.ID, 1)
		}
	})
}

// MarkUnread marks a channel as unread
func (c *Client) MarkUnread(guildID, channelID string, numUnread int) {
	c.Lock()
	defer c.Unlock()

	m, ok := c.UnreadChannels[guildID]
	if !ok {
		c.UnreadChannels[guildID] = map[string]int{}
		m = c.UnreadChannels[guildID]
	}
	m[channelID] += numUnread
}

// MarkRead marks a channel as read, setting the unread message
// Counter back to zero.
func (c *Client) MarkRead(guildID, channelID string) {
	c.Lock()
	defer c.Unlock()

	m, ok := c.UnreadChannels[guildID]
	if !ok {
		c.UnreadChannels[guildID] = map[string]int{}
		m = c.UnreadChannels[guildID]
	}
	m[channelID] = 0
}

// ChannelUnreadMessages returns the number of unread messages in a channel.
func (c *Client) ChannelUnreadMessages(guildID, channelID string) int {
	c.Lock()
	defer c.Unlock()

	if g, ok := c.UnreadChannels[guildID]; ok {
		if numUnread, ok := g[channelID]; ok {
			return numUnread
		}
	}
	return 0
}

// GuildUnreadMessages returns the total number of unread messages
// in a guild's channels.
func (c *Client) GuildUnreadMessages(guildID string) int {
	c.Lock()
	defer c.Unlock()

	sum := 0
	if g, ok := c.UnreadChannels[guildID]; ok {
		for _, v := range g {
			sum += v
		}
	}
	return sum
}

// ActiveChannel returns the client's active channel
func (c *Client) ActiveChannel() string {
	c.Lock()
	retval := c.activeChannel
	c.Unlock()
	return retval
}

// SetChannel sets the client's current channel
func (c *Client) SetChannel(id string) {
	c.Lock()
	c.activeChannel = id
	c.Unlock()
}

// SetGuild sets the client's current guild
func (c *Client) SetGuild(id string) {
	c.Lock()
	c.activeGuild = id
	c.Unlock()
}

// ActiveGuild returns the client's current guild
func (c *Client) ActiveGuild() string {
	c.Lock()
	retval := c.activeGuild
	c.Unlock()
	return retval
}

// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@%%######%%@%+:*@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@%#++++++++++++..:.:@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@%+++************-.--:.-+@@@%##%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@+=***************: .. -+..-::. #@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@%=+*****************+:..   :--- #@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@=*****=***+************= -.....*@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@*+****: +**==***: =:**==* .+ -:.%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@++=**+ ..+.=-:+= .- -*.*+: +--*=:@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@=..=*. *=  .-:. .=+-  =*-- -=:**.%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@++@:=.:+-:+%*++..:=%+.:*:=.:-:**:-@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@- *=.:=@%%@*.:.*%- * - :.=***=-@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@=.=*--+@@@%@*=+%%. .    .****-::@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@#:==#@%%@#=-=*%@@@+.-.  .:+*+*-=#:#@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@#@@@@@@@@@@@@@*+==*-#%@# :-=-=@%+ -=* ---*+ :-@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@%=.+@@@@@@@@@@@@@@#:*  .-=.-=-:.--:   -   .*-.-@@@@@@@@@@@@@@@@@@@@*=%@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@#:+.-@@@@@@@@@@@@@@=: -:.      -+@+       :+.%@@@@@@@@@@@@@@@%@@@@%:.:%@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@-*@::++++++++++++++::=*+  .. :@@+...     .-****+++++++++++=+*--+=*-..+@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@+.*: #-+@@@@@@@%%%@@%@@.   .. :. ..  .   =@@@@@@@@@@@@@@@@@@@:##:%- -%@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@#-. :: :===---:---- :+  ..  ... .  ..   =*-:.-+++****#***=*#.=:-=-#@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@#*=########***+=-.  .... .:.   ...    ..:-===++**+-#%%#****#%%@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@* ... :--   .....  *@@@@@@@@@@@@=-*@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@....---: . ..... .@@@@@@@@@@@@@%-:-=@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@#*%@@@@@@@@@@@@  ...... ......  #@@@@@@@@@@@@@@%-:=:#@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@%+-=#@@@@@@@@@@@@+ ............... *@@@@@@@@@@@@@@##+-*.=@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@#+-:+%@@@@@@@@@@@@@* ..................@@@@@@@@@@@@@@@#+*-- :%@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@%= .. +@@@@@@@@@@@@@@+ . ....  .......... =@@@@@@@@@@@@@@@#=#-   *@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@#=..-:-*@@@@@@@@@@@@@@@+.  .--:..       ..--#@@@@@@@@@@@@@@@@@=#+   =@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@*::==-+-+@@@@@@@@@@@@@@@@-              .. *@@@@@@@@@@@@@@@@@@@@=#+   .%@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@*++: =-*@@@@@@@@@@@@@@@+. ...     ......... +@@@@@@@@@@@@@@@@@@@@=+.    %@@@@@@@@@
// @@@@@@@@@@@@@@@@@@#=   -@@@@@@@@@@@@@@@*: .................... -%@@@@@@@@@@@@@@@@@@@=.     %@@@@@@@@
// @@@@@@@@@@@@@@@@%-    +@@@@@@@@@@@@@@*: ....................... .*@@@@@@@@@@@@@@@@@@@: :  .:@@@@@@@@
// @@@@@@@@@@@@@@@#.   :#@@@@@@@@@@@@@+: ........................... :#@@@@@@@@@@@@@@@@@% .. :--@@@@@@@
// @@@@@@@@@@@@@@%    =@@@@@@@@@@@@%+. ............................... -%@@@@@@@@@@@@@@@+: .  : +@@@@@@
// @@@@@@@@@@@@@@.   +@@@@@@@@@@@#-. ..................................  =@@@@@@@@@@@@@@%. :     %@@@@@
// @@@@@@@@@@@@@-  .#@@@@@@@@@#+:  ...................................... .+@@@@@@@@@@@@@* .. .  -@@@@@
// @@@@@@@@@@@@#  -@@@@@@@@%+:. ........................................... :%@@@@@@@@@@@@  - -.  #@@@@
// @@@@@@@%@@@@. %@@@@@@@*-.  .............................................. .*@@@@@@@@@@@:.+     -@@@@
// @@@@@@@.#@@+ -@@@@@%*:-. .::    ........................................... =@@@@@@@@@@-:%.     %@@@
// @@@@@@@.=@@= #@@@@@+  =# :+@#=..=*#.   ..................................... :@@@@@@@@+ :@.     *@@@
// @@@@*#@-:@@- @@@@@@@=      .=+:   :#*. =%=      ............................. .%@@@@@@.::*     :+@@@
// @@@@+#@* @@- @@@@@@@#:             :-.   :*+.+%     .........................: .%@@@@-..=.:    =*@@@
// @@@@++@@ #@= *@@@@@@@@@#%*.:- +@=         =+ .-=++=.=+-     . ..................=#@@# . +:-.:  -%@@@
// @@@@#-@@-:@: :@@@@@@*=+%@@@@##@++=            .+*+-   .*+=.+**:   ::.   :...:-.-+:+@..  -: +=  =@@@@
// @@@@%.@@@ #*  #@@@@+ .:.=@@++*=*@@+--+@%-   .-    -*. -**#:-*+-+*#*+= +#*- +.=@*#@@-.  ::  %*  %@@@+
// @@@@@.=%*+ -. .@@@@%-  ::--:==%@@@@@@@@@#+*%@@+*#%@@%@- . *@@*:*#+*%#-#*#%*#%@@@@@* .  -.::@* +@@@@:
// @@@@@- #=#.    +@@@@@*.  .:::@@@@@@@@@@@@@@@@@@@@@@@@@:%%:@@@@@@@@@@@@%@@@@@@@@@@# . .  +.#@+ @@@@#-
// @@@@@+ -.-*     @@@@@@@*: .-@@@@@@@@@@@@@@@@@@@@@@@@@#-%=+@@@@@@@@@@@@@@@@@@@@@@% .    *-:@@-+@@@@-+
// @@@@@%.+: @-    :@@@@@@@@##@@@==@@@@@@@@@@@@@@@@@@@@@=:=:@@@@@@@@@@@@@@@@@@@@@@%.     == %@@*@@@@@.%
// @@@@@@-=@.=@.    .#@@@@@@@@@@@@-.=#@@@@@@@@@@@@@@@@@#...:+@@@@@@@@@@@@@@@@@@@@%.     :. #@@@@@@@@=:@
// @@@@@@@.@%.*%.-+:  +@@@@@@@@@@@@- :+@@@@@@@@@@@@@@@@:.....+@@@@@@@@@@@@@@@@@@%..       #@@@@@@@@# +@
// @@@@@@@*-@@#@@@@@=  :%@@@@@@@@@@@*:=:*@@@@@@@@@@@@@=     :%@@@@@@@@@@@@@@@@@#.-.      *@@@@@@@@%.-+@
// @@@@@@@@==@@@@@@@@.   +%@@@@@@@@@@%##-=*@@@@@@@@@@@+.  =#@@@@@@@@@@@@##@@@@+.-.     .#@%@@@@@@%..:-@
// @@@@@@@@@=+@@@@@@@#    .=%@@@@@%@@@@@@%=:=#@@@@@@@@@@%%@@@@@@@@@@@@#=#@#@#: :      :##%@@@@@@# .= #@
// @@@@@@@@@@=+@@@@@@@+      =%@@@%+*@@@@@@+:.:=#@@@@@@@@@@@@@@@@@@@*--%#-#+        =*==%@@@@@@# .+*:@@
// @@@@@@@@@@@*#@@@*@@@= :     -#@@@*:-#@@@@@#*=--+#@@@@@@@@@@@@@@+..*%=-+-.      :#*:+@@@@@@@@.+#%.%@@
// @@@@@@@@@@@@@@@@==@@@-.=      :+%@@*::*@@@@@@@@%#%%@@@@@@@@@%=  +%=.--=+.    .=+--%@@@@@@@@+*@@:+@@@
// @@@@@@@@@@@@@@@@@=:%@@-.+-   .   :+%@*:.-#%**%@@@@@@@@@@@@#-. -#=.:--+=     .::=%@@@@@@@@%*%@@=.@@@@
// @@@@@@@@@@@@@@@@@#-.*@@= =#+: ..    .=+-..:.......:--==++:  =##:--..:.      . *#*+%@@@@@@@@@@+ %@@@@
// @@@@@@@@@@@@@@@@@*:- -%@#::*%+:  .:     .:                  -:.=-    .        . .=+*++++**#@+ %@@@@@
// @@@@@@@@@@*=+#*+==-.   ..:   .                                          .. .    -=:-=+#%%*+- +@@@@@@
// @@@@%#@@#-::--.                                                            ..-.  :   .=*%@*:-=+@@@@@
// @@@@%*#%=-@@*##+.                                                     .          .:=+**++%%@@%#*%@@@
// @@@@@@*=- :*%**#%+-. ..::...                                    .:.          .:==+=-:   =*#+*###%@@@
// @@@@@@@@%*=:.---+**+-::-=+********+++*#*****+=:..:----:::::::  .::...:-==++***+-:. ...:-=--=++*%@@@@
// @@@@@@@@@@@@%+-.   .:..:::::::::::::::--=+***#*==----:::::::--==-::::------::     .:-+#%%@@@@@@@@@@@
// @@@@@@@@@@@@@@@@#+--:::::::::::.     .:::-=+**#%#*++++++********+++==+++=======+*#%@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@%#######%%%%%######*-::::-=++***##########%%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
