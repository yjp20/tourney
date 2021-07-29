package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/peterbourgon/ff"
)

type User struct {
	ID     string
	Status string
	Team   int
}

type Tournament struct {
	Participants   []*User
	ParticipantMap map[string]*User
	Teams          []string
	Captains       []string
	Status         string
}

type Guild struct {
	Current     string
	Tournaments map[string]*Tournament
	History     []string
}

type State struct {
	Guilds map[string]*Guild
}

func main() {
	fs := flag.NewFlagSet("tournament", flag.ExitOnError)
	var (
		_     = fs.String("config", "config", "config file location")
		token = fs.String("token", "", "")
	)
	ff.Parse(fs, os.Args[1:],
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
		ff.WithEnvVarPrefix("Dynamic Esport"),
	)
	dg, err := discordgo.New("Bot " + *token)
	if err != nil {
		log.Fatal(err)
	}
	state := NewState()
	dg.AddHandler(messageHandler(state))
	err = dg.Open()
	if err != nil {
		log.Fatal("error opening connection,", err)
	}
	// Handle proper exit
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	dg.Close()
}

func NewState() State {
	s := State{}
	s.Guilds = make(map[string]*Guild)
	return s
}

func (s *State) GetGuild(guildID string) *Guild {
	if guild, ok := s.Guilds[guildID]; ok {
		return guild
	} else {
		g := &Guild{}
		g.Tournaments = make(map[string]*Tournament)
		s.Guilds[guildID] = g
		return g
	}
}

func messageHandler(state State) interface{} {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		msg, _ := json.MarshalIndent(m, "", "  ")
		os.Stdout.Write(msg)
		if m.Author.ID == s.State.User.ID {
			return
		}
		if m.Content[0] != '!' {
			return
		}
		args := strings.Split(m.Content, " ")
		guild := state.GetGuild(m.GuildID)
		name := strings.Join(args[1:], "")
		if len(name) == 0 {
			name = guild.Current
		}

		switch args[0] {
		case "!new":
			if len(strings.Join(args[1:], "")) == 0 {
				s.ChannelMessageSend(m.ChannelID, ">>> New tournament must have a name")
				return
			}
			if _, ok := guild.Tournaments[name]; ok {
				s.ChannelMessageSend(m.ChannelID, ">>> Tournament with same name found, use another name")
				return
			}
			tournament := &Tournament{Status: "Setup"}
			tournament.ParticipantMap = make(map[string]*User)
			guild.Tournaments[name] = tournament
			guild.Current = name
			guild.History = append(guild.History, name)
			s.ChannelMessageSend(m.ChannelID, ">>> Tournament created with name: **"+name+"**")

		case "!join":
			if _, ok := guild.Tournaments[name]; !ok {
				s.ChannelMessageSend(m.ChannelID, ">>> Tournament not found: **"+name+"**")
				return
			}
			tournament := guild.Tournaments[name]
			if participant, ok := tournament.ParticipantMap[m.Author.ID]; ok {
				if participant.Status == "Left" {
					participant.Status = "Joined"
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(">>> Rejoined tournament **%s**", name))
				} else {
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(">>> Already joined tournament **%s**", name))
				}
			} else {
				user := User{
					ID:     m.Author.ID,
					Status: "Joined",
					Team:   -1,
				}
				tournament.Participants = append(tournament.Participants, &user)
				tournament.ParticipantMap[m.Author.ID] = &user
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(">>> Joined tournament **%s**", name))
			}

		case "!leave":
			if _, ok := guild.Tournaments[name]; !ok {
				s.ChannelMessageSend(m.ChannelID, ">>> Tournament not found: **"+name+"**")
				return
			}
			tournament := guild.Tournaments[name]
			if participant, ok := tournament.ParticipantMap[m.Author.ID]; ok {
				participant.Status = "Left"
			} else {
				user, err := s.User(participant.ID)
				if err == nil {
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(">>> User **%s** not found in tournament **%s**", user.Username, name))
				}
			}

		case "!captain":
			if _, ok := guild.Tournaments[name]; !ok {
				s.ChannelMessageSend(m.ChannelID, ">>> Tournament not found: **"+name+"**")
				return
			}
			tournament := guild.Tournaments[name]
			if participant, ok := tournament.ParticipantMap[m.Author.ID]; ok {
				idx := len(tournament.Teams)
				tournament.Teams = append(tournament.Teams, fmt.Sprintf("Team %d", idx+1))
				participant.Team = idx
				s.ChannelMessageSend(m.ChannelID, ">>> User now captain")
			} else {
				user, err := s.User(participant.ID)
				if err == nil {
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(">>> User **%s** not found in tournament **%s**", user.Username, name))
				}
			}

		case "!uncaptain":

		case "!draft":
			if len(args) >= 2 {
				idx, err := strconv.Atoi(args[1])
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, ">>> Not a number, users must be given as a number")
					return
				}
				idx-- // compensate for human viewable numbers
				name := strings.Join(args[2:], "")
				if len(name) == 0 {
					name = guild.Current
				}
				if _, ok := guild.Tournaments[name]; ok {
					s.ChannelMessageSend(m.ChannelID, ">>> Tournament not found: **"+name+"**")
					return
				}
				tournament := guild.Tournaments[name]
				if tournament.ParticipantMap[m.Author.ID].Team == -1 {
					s.ChannelMessageSend(m.ChannelID, ">>> Can't draft if not in team")
					return
				}
				if len(tournament.Participants) > idx {
					s.ChannelMessageSend(m.ChannelID, ">>> Index larger than the number of players")
					return
				}
				if tournament.Participants[idx].Team != -1 {
					s.ChannelMessageSend(m.ChannelID, ">>> Player already drafted")
					return
				}
				team := tournament.ParticipantMap[m.Author.ID].Team
				tournament.Participants[idx].Team = team
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(">>> Player now drafted in %s", tournament.Teams[team]))
			}

		case "!start":
			if _, ok := guild.Tournaments[name]; !ok {
				s.ChannelMessageSend(m.ChannelID, ">>> Tournament not found: **"+name+"**")
			}
			tournament := guild.Tournaments[name]
			tournament.Status = "Start"
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(">>> Tournament **%s** started", name))

		case "!status":
			if tournament, ok := guild.Tournaments[name]; ok {
				str := ">>> "
				str += fmt.Sprintf("Name: **%s**\n", name)
				str += fmt.Sprintf("Status: **%s**\n", tournament.Status)
				str += "\n"
				teamMap := make(map[int][]string)
				for idx, participant := range tournament.Participants {
					user, err := s.User(participant.ID)
					if err == nil {
						teamname := ""
						if participant.Team != -1 {
							teamname = fmt.Sprintf("*%s*", tournament.Teams[participant.Team])
						}
						teamMap[participant.Team] = append(teamMap[participant.Team], user.Username)
						if participant.Status == "Joined" {
							str += fmt.Sprintf("%d: **%s**  %s\n", idx+1, user.Username, teamname)
						} else {
							str += fmt.Sprintf("~~%d: **%s**  %s~~\n", idx+1, user.Username, teamname)
						}
					}
				}
				str += "\n"
				for idx, team := range teamMap {
					teamname := "Undrafted"
					if idx != -1 {
						teamname = fmt.Sprintf("*%s*", tournament.Teams[idx])
					}
					str += fmt.Sprintf("__%s__\n", teamname)
					for jdx, player := range team {
						str += fmt.Sprintf("%d: **%s**\n", jdx+1, player)
					}
					str += "\n"
				}
				s.ChannelMessageSend(m.ChannelID, str)
			} else {
				s.ChannelMessageSend(m.ChannelID, ">>> Tournament not found: **"+name+"**")
			}

		case "!kick":
		case "!help":
		case "!finish":
			if tournament, ok := guild.Tournaments[name]; ok {
				tournament.Status = "Finished"
			} else {
				s.ChannelMessageSend(m.ChannelID, ">>> Tournament not found: **"+name+"**")
			}

		default:
			s.ChannelMessageSend(m.ChannelID, ">>> Command not found, try __!help__")
		}
	}
}
