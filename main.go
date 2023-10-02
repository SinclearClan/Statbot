package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	// Lade die Umgebungsvariablen aus der .env-Datei
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Fehler beim Laden der .env-Datei")
	}

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
		fmt.Println("Fehler beim Erstellen der Discord-Sitzung:", err)
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

	// Öffne die Discord-Sitzung
	err = dg.Open()
	if err != nil {
		fmt.Println("Fehler beim Öffnen der Discord-Sitzung:", err)
		return
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
