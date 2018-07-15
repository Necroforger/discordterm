# discordterm

A quick terminal client for discord I wrote on a raspberry pi when my computer was broken.

## Installing
`go get -u github.com/Necroforger/discordterm/cmd/discordterm`

## Shortcuts
[Here are the text shortcuts available](https://github.com/chzyer/readline/blob/master/doc/shortcut.md)

## Flags

| Flag           | Description                                                                                 |
|----------------|---------------------------------------------------------------------------------------------|
| username       | Username to log in with                                                                     |
| password       | password to log in with only provide if you are entering a username as well                 |
| token          | A user or bot token to log in with. If using a bot token, remember to prefix it with 'Bot ' |
| show-nicknames | Show users' nicknames in place of usernames when possible                                   |
| show-images    | Automatically print images                                                                  |
| img-width      | Sets the default width of images                                                            |
| color-images   | if enabled, images will have color                                                          |
| color-text     | if enabled, text will be colored                                                            |

## Help

When using commands, exclude the `/` prefix

```
====| Commands: |==============================================
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
```
