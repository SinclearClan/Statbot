package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var (
	dgp *discordgo.Session
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

	// Füge einen Handler für alle Nachrichten hinzu, und einen für die Voice-Events
	dg.AddHandler(messageHandler)
	dg.AddHandler(voiceStateUpdateHandler)

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

	dgp = dg

	defer dg.Close()

	// Erstelle den Webserver
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Erhalte die Guild-ID aus der URL
		guildID := r.URL.Path[1:]

		// Hier könntest du die Statistik-Daten abrufen und die Diagramme generieren
		// Verwende die Guild-ID, um die entsprechenden Daten abzurufen

		// Beispiel:
		channelData := getChannelData(guildID)
		userData := getUserData(guildID)

		// Erstelle die HTML-Seite mit den Diagrammen
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Statistiken</title>
			<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
		</head>
		<body>
			<div style="width:400px;height:400px;">
				<canvas id="channelChart" width="400" height="400"></canvas>
			</div>
			<br />
			<div style="width:400px;height:400px;">
				<canvas id="userChart" width="400" height="400"></canvas>
			</div>
			<script>
				// Code zum Erstellen der Diagramme mit Chart.js
				var channelData = JSON.parse('%s'); // JSON mit den Channel-Daten
				var userData = JSON.parse('%s'); // JSON mit den User-Daten

				// Hier Chart.js verwenden, um die Diagramme zu erstellen
				var ctxChannel = document.getElementById('channelChart').getContext('2d');
				var ctxUser = document.getElementById('userChart').getContext('2d');

				var channelLabels = channelData.map(function(item) {
					return item.name;
				});

				var channelDataValues = channelData.map(function(item) {
					return item.count;
				});

				var userLabels = userData.map(function(item) {
					return item.name;
				});
				var userDataValues = userData.map(function(item) {
					return item.count;
				});

				var channelChart = new Chart(ctxChannel, {
					type: 'bar',
					data: {
						labels: channelLabels,
						datasets: [{
							label: 'Nachrichten pro Channel',
							data: channelDataValues,
							backgroundColor: 'rgba(75, 192, 192, 0.2)',
							borderColor: 'rgba(75, 192, 192, 1)',
							borderWidth: 1
						}]
					},
					options: {
						scales: {
							x: {
								beginAtZero: true,
								title: {
									display: true,
									text: 'Channel'
								}
							},
							y: {
								beginAtZero: true,
								title: {
									display: true,
									text: 'Anzahl Nachrichten'
								}
							}
						}
					}
				});

				var userChart = new Chart(ctxUser, {
					type: 'bar',
					data: {
						labels: userLabels,
						datasets: [{
							label: 'Nachrichten pro User',
							data: userDataValues,
							backgroundColor: 'rgba(255, 99, 132, 0.2)',
							borderColor: 'rgba(255, 99, 132, 1)',
							borderWidth: 1
						}]
					},
					options: {
						scales: {
							x: {
								beginAtZero: true,
								title: {
									display: true,
									text: 'User'
								}
							},
							y: {
								beginAtZero: true,
								title: {
									display: true,
									text: 'Anzahl Nachrichten'
								}
							}
						}
					}
				});
			</script>
		</body>
		</html>
		`

		// Sende die HTML-Seite zurück
		fmt.Fprintf(w, html, channelData, userData)
	})

	// Starte den Webserver
	go http.ListenAndServe(":9786", nil)

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

func messageHandler(dg *discordgo.Session, m *discordgo.MessageCreate) {
	// Speichere die Nachricht in der Datenbank
	saveMessage(m.GuildID, m.Author.ID, m.ChannelID)
}

func voiceStateUpdateHandler(dg *discordgo.Session, m *discordgo.VoiceStateUpdate) {
	// Speichere das Voice-Event in der Datenbank
	if m.ChannelID != "" {
		saveVoiceEvenet("join", m.GuildID, m.UserID, m.ChannelID)
	} else {
		saveVoiceEvenet("leave", m.GuildID, m.UserID, m.ChannelID)
	}
}

func saveMessage(guildID, userID, channelID string) {
	// Aktuelles Datum und Zeit
	now := time.Now()

	// Name der Datenbankdatei nach dem Schema "GuildID-Month.db"
	dbName := fmt.Sprintf("%s-%d.db", guildID, now.Month())

	// Überprüfen, ob die Datenbankdatei bereits existiert
	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		// Wenn nicht, dann erstelle sie
		db, err := sql.Open("sqlite3", dbName)
		if err != nil {
			log.Fatal("Problem beim Erstellen der Datenbank: ", err)
		}
		defer db.Close()

		// Tabelle für die Nachrichten
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS chat (
				id INTEGER PRIMARY KEY,
				channel_id TEXT,
				user_id TEXT,
				time DATETIME
			)
		`)
		if err != nil {
			log.Fatal("Problem beim Erstellen der Tabelle für die Nachrichten: ", err)
		}

		// Tabelle für die Voice-Events
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS voice (
				id INTEGER PRIMARY KEY,
				type TEXT,
				channel_id TEXT,
				user_id TEXT,
				time DATETIME
			)
		`)
		if err != nil {
			log.Fatal("Problem beim Erstellen der Tabelle für die Voice-Events: ", err)
		}
	}

	// Öffne die Datenbankverbindung
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatal("Problem beim Öffnen der Datenbank: ", err)
	}
	defer db.Close()

	// Füge die Nachricht der Tabelle hinzu
	_, err = db.Exec(`
		INSERT INTO chat (channel_id, user_id, time)
		VALUES (?, ?, ?)
	`, channelID, userID, now.Format(time.RFC3339))
	if err != nil {
		log.Fatal("Problem beim Speichern der Nachricht: ", err)
	}
}

func saveVoiceEvenet(voiceEventType, guildID, userID, channelID string) {
	// Aktuelles Datum und Zeit
	now := time.Now()

	// Name der Datenbankdatei nach dem Schema "GuildID-Month.db"
	dbName := fmt.Sprintf("%s-%d.db", guildID, now.Month())

	// Überprüfen, ob die Datenbankdatei bereits existiert
	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		// Wenn nicht, dann erstelle sie
		db, err := sql.Open("sqlite3", dbName)
		if err != nil {
			log.Fatal("Problem beim Erstellen der Datenbank: ", err)
		}
		defer db.Close()

		// Tabelle für die Voice-Events
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS voice (
				id INTEGER PRIMARY KEY,
				type TEXT,
				channel_id TEXT,
				user_id TEXT,
				time DATETIME
			)
		`)
		if err != nil {
			log.Fatal("Problem beim Erstellen der Tabelle für die Voice-Events: ", err)
		}

		// Tabelle für die Nachrichten
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS chat (
				id INTEGER PRIMARY KEY,
				channel_id TEXT,
				user_id TEXT,
				time DATETIME
			)
		`)
		if err != nil {
			log.Fatal("Problem beim Erstellen der Tabelle für die Nachrichten: ", err)
		}
	}

	// Öffne die Datenbankverbindung
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatal("Problem beim Öffnen der Datenbank: ", err)
	}
	defer db.Close()

	// Füge die Nachricht der Tabelle hinzu
	_, err = db.Exec(`
		INSERT INTO voice (type, channel_id, user_id, time)
		VALUES (?, ?, ?, ?)
	`, voiceEventType, channelID, userID, now.Format(time.RFC3339))
	if err != nil {
		log.Fatal("Problem beim Speichern des Voice-Events: ", err)
	}
}

// Rufe die Statistik-Daten für die angefragte GuildID ab und gebe sie als JSON zurück
// Wenn die angefragte GuildID nicht existiert, dann gebe "{}" zurück
func getChannelData(guildID string) string {
	// Prüfe, ob die Datenbankdatei existiert, erstelle dabei aber KEINE neue Datei
	dbName := fmt.Sprintf("%s-%d.db", guildID, time.Now().Month())
	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		return "{}"
	}

	// Öffne die Datenbankverbindung
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatal("Problem beim Öffnen der Datenbank: ", err)
	}
	defer db.Close()

	// Hole die Statistik-Daten aus der Datenbank
	rows, err := db.Query(`
		SELECT channel_id, COUNT(*) AS count
		FROM chat
		GROUP BY channel_id
	`)
	if err != nil {
		log.Fatal("Problem beim Abrufen der Statistik-Daten: ", err)
	}
	defer rows.Close()

	// Erstelle ein Array mit den Statistik-Daten
	var channelData []string
	for rows.Next() {
		var channelID string
		var count int
		rows.Scan(&channelID, &count)
		name := getChannelName(channelID)
		channelData = append(channelData, fmt.Sprintf(`{ "name": "%s", "count": %d }`, name, count))
	}

	// Erstelle das JSON mit den Statistik-Daten
	return fmt.Sprintf(`[%s]`, strings.Join(channelData, ","))
}

func getUserData(guildID string) string {
	// Prüfe, ob die Datenbankdatei existiert, erstelle dabei aber KEINE neue Datei
	dbName := fmt.Sprintf("%s-%d.db", guildID, time.Now().Month())
	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		return "{}"
	}

	// Öffne die Datenbankverbindung
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatal("Problem beim Öffnen der Datenbank: ", err)
	}
	defer db.Close()

	// Hole die Statistik-Daten aus der Datenbank
	rows, err := db.Query(`
		SELECT user_id, COUNT(*) AS count
		FROM chat
		GROUP BY user_id
	`)
	if err != nil {
		log.Fatal("Problem beim Abrufen der Statistik-Daten: ", err)
	}
	defer rows.Close()

	// Erstelle ein Array mit den Statistik-Daten
	var userData []string
	for rows.Next() {
		var userID string
		var count int
		rows.Scan(&userID, &count)
		name := getUserName(userID)
		userData = append(userData, fmt.Sprintf(`{ "name": "%s", "count": %d }`, name, count))
	}

	// Erstelle das JSON mit den Statistik-Daten
	return fmt.Sprintf(`[%s]`, strings.Join(userData, ","))
}

func getChannelName(channelID string) string {
	channel, err := dgp.Channel(channelID)
	if err != nil {
		log.Println("Fehler beim Abrufen des Channels:", err)
		return ""
	}
	return channel.Name
}

func getUserName(userID string) string {
	user, err := dgp.User(userID)
	if err != nil {
		log.Println("Fehler beim Abrufen des Users:", err)
		return ""
	}
	return user.Username
}

// Liest die Voice-Events aus der Datenbank aus und gibt die Zeit zurück, die der Nutzer zwischen den letzten beiden leave Events in einem Voice-Channel verbracht hat.
// Dabei ist es möglich, dass mehrer join Events hintereinander eingetragen sind, das bedeutet, dass der Nutzer in der Zwischenzeit den Channel gewechselt hat. Dann muss die Zeit zusammengezählt werden, beziehungsweise nur der älteste join zählt, seitdem kein leave Event mehr eingetragen wurde.
func calculateTimeInVoice(guildID, userID string) time.Duration {
	// Prüfe, ob die Datenbankdatei existiert, erstelle dabei aber KEINE neue Datei
	dbName := fmt.Sprintf("%s-%d.db", guildID, time.Now().Month())
	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		return -1
	}

	// Öffne die Datenbankverbindung
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatal("Problem beim Öffnen der Datenbank: ", err)
	}
	defer db.Close()

	// Hole die Voice-Events aus der Datenbank
	rows, err := db.Query(`
		SELECT type, channel_id, user_id, time
		FROM voice
		WHERE user_id = ?
		ORDER BY time ASC
	`, userID)
	if err != nil {
		log.Fatal("Problem beim Abrufen der Voice-Events: ", err)
	}
	defer rows.Close()

	var lastLeaveTime time.Time
	var totalTime time.Duration
	for rows.Next() {
		var voiceEventType, channelID, userID string
		var timeString string
		rows.Scan(&voiceEventType, &channelID, &userID, &timeString)
		t, err := time.Parse(time.RFC3339, timeString)
		if err != nil {
			log.Fatal("Problem beim Parsen der Zeit: ", err)
		}
		if voiceEventType == "join" {
			// Wenn der Nutzer den Channel wechselt, dann wird der letzte leave-Eintrag überschrieben
			lastLeaveTime = t
		} else {
			// Wenn der Nutzer den Channel verlässt, dann wird die Zeit zwischen dem letzten leave-Eintrag und dem aktuellen leave-Eintrag addiert
			totalTime += t.Sub(lastLeaveTime)
		}
	}

	// Die totalTime ist eine Duration, die in Nanosekunden angegeben wird. Um die Zeit in Stunden, Minuten und Sekunden zu erhalten, muss diese Duration in einen time.Time umgewandelt werden.
	// Dazu wird ein time.Time mit dem Unix-Timestamp 0 erstellt und dann die totalTime in Nanosekunden addiert.
	// Dann kann die totalTime in Stunden, Minuten und Sekunden aufgeteilt werden.
	// Die totalTime wird in Stunden, Minuten und Sekunden aufgeteilt, weil die totalTime größer als 24 Stunden sein

	// Konvertiere die totalTime in time.Duration
	return time.Duration(totalTime)
	//TODO: CODE IN DIESER FUNKTION PRÜFEN, IST KOMPLETT AI-GENERIERT!!!
}
