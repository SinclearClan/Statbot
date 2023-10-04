package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var (
	dgp    *discordgo.Session
	dbLock sync.Mutex
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
		voiceData := getVoiceData(guildID)

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
			<div style="width:400px;height:400px;">
    			<canvas id="voiceChart" width="400" height="400"></canvas>
			</div>
			<script>
				// Code zum Erstellen der Diagramme mit Chart.js
				var channelData = JSON.parse('%s'); // JSON mit den Channel-Daten
				var userData = JSON.parse('%s'); // JSON mit den User-Daten
				var voiceData = JSON.parse('%s'); // JSON mit den Voice-Daten

				// Hier Chart.js verwenden, um die Diagramme zu erstellen
				var ctxChannel = document.getElementById('channelChart').getContext('2d');
				var ctxUser = document.getElementById('userChart').getContext('2d');
				var ctxVoice = document.getElementById('voiceChart').getContext('2d');

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

				var voiceLabels = voiceData.map(function(item) {
					return item.name;
				});
				var voiceDataValues = voiceData.map(function(item) {
					return item.duration;
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

				var voiceChart = new Chart(ctxVoice, {
					type: 'bar',
					data: {
						labels: voiceLabels,
						datasets: [{
							label: 'Zeit im Voice-Chat (in Minuten)',
							data: voiceDataValues,
							backgroundColor: 'rgba(255, 159, 64, 0.2)',
							borderColor: 'rgba(255, 159, 64, 1)',
							borderWidth: 1
						}]
					},
					options: {
						scales: {
							x: {
								beginAtZero: true,
								title: {
									display: true,
									text: 'Nutzer'
								}
							},
							y: {
								beginAtZero: true,
								title: {
									display: true,
									text: 'Zeit (in Minuten)'
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
		fmt.Fprintf(w, html, channelData, userData, voiceData)
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
		saveVoiceEvent("join", m.GuildID, m.UserID, m.ChannelID)
	} else {
		saveVoiceEvent("leave", m.GuildID, m.UserID, m.ChannelID)
	}
}

func checkDatabaseSetup(guildID string) bool {
	// Name der Datenbankdatei nach dem Schema "GuildID-Month.db"
	dbName := fmt.Sprintf("%s-%d.db", guildID, time.Now().Month())

	// Überprüfen, ob die Datenbankdatei bereits existiert
	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		log.Println("Datenbankdatei existiert nicht. Erstelle neue Datenbankdatei...")

		// Erstelle die Datenbankdatei
		file, err := os.Create(dbName)
		if err != nil {
			log.Fatal("Problem beim Erstellen der Datenbankdatei: ", err)
			return false
		}
		file.Close()
	}

	// Öffne die Datenbankverbindung
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatal("Problem beim Öffnen der Datenbank: ", err)
		return false
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
		return false
	}

	// Tabelle für die Voice-Events
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS voice (
			id INTEGER PRIMARY KEY,
			user_id TEXT,
			channel_id TEXT,
			duration INTEGER
		)
	`)
	if err != nil {
		log.Fatal("Problem beim Erstellen der Tabelle für die Voice-Events: ", err)
		return false
	}

	// Tabelle für aktuelle Aufenthalte im Voice-Channel
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS voice_current (
			id INTEGER PRIMARY KEY,
			channel_id TEXT,
			user_id TEXT,
			start DATETIME
		)
	`)
	if err != nil {
		log.Fatal("Problem beim Erstellen der Tabelle für die aktuellen Aufenthalte im Voice-Channel: ", err)
		return false
	}

	return true
}

func saveMessage(guildID, userID, channelID string) {
	dbLock.Lock()
	defer dbLock.Unlock()

	if !checkDatabaseSetup(guildID) {
		log.Fatal("Datenbank nicht eingerichtet")
	}

	// Aktuelles Datum und Zeit
	now := time.Now()

	// Name der Datenbankdatei nach dem Schema "GuildID-Month.db"
	dbName := fmt.Sprintf("%s-%d.db", guildID, now.Month())

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

func saveVoiceEvent(voiceEventType, guildID, userID, channelID string) {
	dbLock.Lock()
	defer dbLock.Unlock()

	if !checkDatabaseSetup(guildID) {
		log.Fatal("Datenbank nicht eingerichtet")
	}

	// Aktuelles Datum und Zeit
	now := time.Now()

	// Name der Datenbankdatei nach dem Schema "GuildID-Month.db"
	dbName := fmt.Sprintf("%s-%d.db", guildID, now.Month())

	// Öffne die Datenbankverbindung
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatal("Problem beim Öffnen der Datenbank: ", err)
	}
	defer db.Close()

	// Füge die Nachricht der Tabelle für
	// Wenn es ein join-Event ist, dann füge es der Tabelle für die aktuellen Aufenthalte hinzu
	// Wenn es ein leave-Event ist, dann berechne die Zeit, die der Nutzer im Voice-Channel verbracht hat
	if voiceEventType == "join" {
		// Prüfe erst, ob bereits ein join-Event für diesen Nutzer in der Tabelle ist
		rows, err := db.Query(`
			SELECT id
			FROM voice_current
			WHERE user_id = ?
		`, userID)
		if err != nil {
			log.Fatal("Problem beim Abrufen des aktuellen Aufenthalts: ", err)
		}
		defer rows.Close()

		// Wenn es bereits einen Eintrag für diesen Nutzer gibt, dann berechne die Zeit, die er im Voice-Channel verbracht hat und trage diese in die Tabelle voice ein
		if rows.Next() {
			// Hole die ID des Eintrags
			var id int
			rows.Scan(&id)

			// Hole die Zeit, zu der der Nutzer den Voice-Channel betreten hat
			var start time.Time
			err = db.QueryRow(`
				SELECT start
				FROM voice_current
				WHERE id = ?
			`, id).Scan(&start)
			if err != nil {
				log.Fatal("Problem beim Abrufen der Startzeit aktuellen Aufenthalts: ", err)
			}

			// Berechne die Zeit, die der Nutzer im Voice-Channel verbracht hat
			duration := now.Sub(start)

			// Trage die Zeit in die Tabelle voice ein
			_, err = db.Exec(`
				INSERT INTO voice (user_id, channel_id, duration)
				VALUES (?, ?, ?)
			`, userID, channelID, duration.Minutes())
			if err != nil {
				log.Fatal("Problem beim Speichern des Voice-Events: ", err)
			}

			// Lösche den Eintrag aus der Tabelle für die aktuellen Aufenthalte
			_, err = db.Exec(`
				DELETE FROM voice_current
				WHERE id = ?
			`, id)
			if err != nil {
				log.Fatal("Problem beim Löschen des aktuellen Aufenthalts: ", err)
			}
		}

		_, err = db.Exec(`
			INSERT INTO voice_current (channel_id, user_id, start)
			VALUES (?, ?, ?)
		`, channelID, userID, now.Format(time.RFC3339))
		if err != nil {
			log.Fatal("Problem beim Speichern des aktuellen Aufenthalts: ", err)
		}
	} else if voiceEventType == "leave" {
		// Rufe den Zeitpunkt des letzten join-Events für diesen Nutzer ab
		var start time.Time
		err = db.QueryRow(`
			SELECT start
			FROM voice_current
			WHERE user_id = ?
		`, userID).Scan(&start)
		if err != nil {
			log.Fatal("Problem beim Abrufen der Startzeit aktuellen Aufenthalts: ", err)
		}

		// Berechne die Zeit, die der Nutzer im Voice-Channel verbracht hat
		duration := now.Sub(start)

		// Da bei einem leave-Event die Channel-ID leer ist, muss diese aus der Tabelle für die aktuellen Aufenthalte abgerufen werden
		err = db.QueryRow(`
			SELECT channel_id
			FROM voice_current
			WHERE user_id = ?
		`, userID).Scan(&channelID)
		if err != nil {
			log.Fatal("Problem beim Abrufen der Channel-ID des aktuellen Aufenthalts: ", err)
		}

		// Trage die Zeit in die Tabelle voice ein
		_, err = db.Exec(`
			INSERT INTO voice (user_id, channel_id, duration)
			VALUES (?, ?, ?)
		`, userID, channelID, duration.Minutes())
		if err != nil {
			log.Fatal("Problem beim Speichern des Voice-Events: ", err)
		}

		// Lösche den Eintrag aus der Tabelle für die aktuellen Aufenthalte
		_, err = db.Exec(`
			DELETE FROM voice_current
			WHERE user_id = ?
		`, userID)
		if err != nil {
			log.Fatal("Problem beim Löschen des aktuellen Aufenthalts: ", err)
		}

	} else {
		log.Fatal("Ungültiger Voice-Event-Typ")
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

func getVoiceData(guildID string) string {
	//TODO
	return "{}"
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
