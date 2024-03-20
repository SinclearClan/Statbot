package main

import (
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"os/signal"
	"github.com/SinclearClan/Statbot/database"
	"github.com/bwmarrin/discordgo"
)

var (
	dg *discordgo.Session
)

func main() {
	// Lade die Umgebungsvariablen aus der .env-Datei
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Fehler beim Laden der .env-Datei")
	}

	// Erstelle und setze die MySQL-URL
	database.SetMySqlUrl(os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_DABA"))

	// Lese den Discord-Bot-Token aus der Umgebungsvariable
	token := os.Getenv("DISCORD_BOT_TOKEN")

	// Lege alle Discord Slash-Commands an
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "Antwortet, wenn der Bot online ist",
		},
	}

	// Lege die passenden Handler für die Slash-Commands an
	commandHandlers := map[string]func(dg *discordgo.Session, i *discordgo.InteractionCreate){
		"ping": func(dg *discordgo.Session, i *discordgo.InteractionCreate) {
			dg.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Pong!",
				},
			})
		},
	}

	// Erstelle eine neue Instanz des Discord-Bots
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Println("Fehler beim Erstellen der Discord-Sitzung:", err)
		return
	}

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Bot ist online und läuft als %s#%s\n", r.User.Username, r.User.Discriminator)
	})

	// Füge die Handler für die Slash-Commands hinzu
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if handler, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			handler(s, i)
		}
	})

	// Füge einen Handler für alle Nachrichten hinzu, und einen für die Voice-Events
	dg.AddHandler(messageHandler)
	dg.AddHandler(voiceStateUpdateHandler)

	// Öffne die Discord-Sitzung
	err = dg.Open()
	if err != nil {
		log.Println("Fehler beim Öffnen der Discord-Sitzung:", err)
		return
	}

	// Rufe die Liste der Guilds ab, auf denen der Bot Mitglied ist
	guilds, err := dg.UserGuilds(100, "", "")
	if err != nil {
		log.Fatal("Fehler beim Abrufen der Guilds:", err)
		return
	}

	// Setup der Datenbank für jede Guild
	log.Println("Starte Setup der Datenbanken für die Guilds")
	for _, guild := range guilds {
		err := database.Setup(guild.ID)
		if err != nil {
			log.Fatalf("Fehler beim Einrichten der Datenbank für Guild %s: %s", guild.Name, err)
		}
	}

	// Füge die Slash-Commands hinzu
	log.Println("Slash-Commands werden angelegt...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", v)
		if err != nil {
			log.Panicf("Fehler beim Anlegen des Slash-Commands %s: %s", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer dg.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Bot läuft. Drücke CTRL+C, um zu beenden.")
	<-stop

	// Lösche die Slash-Commands
	log.Println("Slash-Commands werden gelöscht...")
	for _, v := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, "", v.ID)
		if err != nil {
			log.Panicf("Fehler beim Löschen des Slash-Commands %s: %s", v.Name, err)
		}
	}

	log.Println("Bot wird anständig beendet.")
	os.Exit(0)
}

// messageHandler handles incoming messages and saves them to the database
func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	err := database.SaveMessage(m.GuildID, m.Author.ID, m.ChannelID)
	if err != nil {
		log.Println(err)
	}
}

// voiceStateUpdateHandler handles voice state updates and saves them to the database
func voiceStateUpdateHandler(s *discordgo.Session, m *discordgo.VoiceStateUpdate) {
	err := database.SaveVoiceEvent(m.GuildID, m.UserID, m.ChannelID)
	if err != nil {
		log.Println(err)
	}
}
