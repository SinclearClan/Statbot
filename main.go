package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
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

	// Füge einen Handler für alle Nachrichten hinzu, die keine Slash-Commands sind
	dg.AddHandler(messageHandler)

	// Öffne die Discord-Sitzung
	err = dg.Open()
	if err != nil {
		log.Println("Fehler beim Öffnen der Discord-Sitzung:", err)
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

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Speichere die Nachricht in der Datenbank
	saveMessage(m.GuildID, m.Author.ID, m.Author.Username)
}

func saveMessage(guildID, userID, username string) {
	// Aktuelles Datum und Zeit
	now := time.Now()

	// Name der Datenbankdatei nach dem Schema "GuildID-Month.db"
	dbName := fmt.Sprintf("%s-%d.db", guildID, now.Month())

	// Überprüfen, ob die Datenbankdatei bereits existiert
	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		// Wenn nicht, dann erstelle sie
		db, err := sql.Open("sqlite3", dbName)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS users (
				id TEXT PRIMARY KEY,
				username TEXT,
				created_at DATETIME
			)
		`)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Öffne die Datenbankverbindung
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Überprüfe, ob der Nutzer bereits in der Liste der Nutzer ist
	_, err = db.Exec(`
		INSERT OR IGNORE INTO users (id, username, created_at) VALUES (?, ?, ?)
	`, userID, username, now.Format(time.RFC3339))
	if err != nil {
		log.Fatal(err)
	}

	// Überprüfe, ob die Tabelle für den Nutzer bereits existiert
	userTableName := normalizeUsername(username)
	log.Println("UserTableName:", userTableName)
	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY,
			created_at DATETIME
		)
	`, userTableName))
	if err != nil {
		log.Fatal(err)
	}

	// Füge die Nachricht hinzu
	_, err = db.Exec(fmt.Sprintf(`
		INSERT INTO %s (created_at) VALUES (?)
	`, userTableName), now.Format(time.RFC3339))
	if err != nil {
		log.Fatal(err)
	}
}

func normalizeUserID(userID string) string {
	// Ersetze "-" durch "_"
	return strings.ReplaceAll(userID, "-", "_")
}

func normalizeUsername(username string) string {
	// Erlaube nur Buchstaben, Zahlen und Unterstriche im Tabellennamen
	return regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(username, "_")
}
