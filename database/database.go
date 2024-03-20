package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

var (
	dbUrl string
)

func SetMySqlUrl(dbHost, dbPort, dbUser, dbPass, dbDaba string) {
    // Format MySQL connection URL
    url := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPass, dbHost, dbPort, dbDaba)
    log.Println("mysqlUrl: " + url)
    dbUrl = url
}

func openDBConnection() (*sql.DB, error) {
	if dbUrl == "" {
		return nil, fmt.Errorf("MySQL URL is not set")
	}
	return sql.Open("mysql", dbUrl)
}

func Setup(guildID string) error {
	// Ensure that database connection is set
	if dbUrl == "" {
		return fmt.Errorf("MySQL URL is not set")
	}

	// Open database connection
	db, err := sql.Open("mysql", dbUrl)
	if err != nil {
		log.Fatalf("failed to connect to MySQL: %v", err)
	}
	defer db.Close()

	// Create table for chat messages
	log.Println("Erstelle Tabelle für Chatnachrichten, falls diese noch nicht existiert")
	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS chat_%s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			channel_id VARCHAR(255),
			user_id VARCHAR(255),
			time DATETIME
		)
	`, guildID))
	if err != nil {
		return fmt.Errorf("problem creating chat table: %s", err)
	}

	// Create table for voice events
	log.Println("Erstelle Tabelle für Sprachevents, falls diese noch nicht existiert")
	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS voice_%s (
			id INT AUTO_INCREMENT PRIMARY KEY,
		user_id VARCHAR(255),
		channel_id VARCHAR(255),
		duration INT
		)
	`, guildID))
	if err != nil {
		return fmt.Errorf("problem creating voice table: %s", err)
	}

	// Create table for current voice channel presence
	log.Println("Erstelle Tabelle für laufende Telefonate, falls diese noch nicht existiert")
	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS voice_current_%s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			channel_id VARCHAR(255),
			user_id VARCHAR(255),
			start DATETIME
		)
	`, guildID))
	if err != nil {
		return fmt.Errorf("problem creating voice_current table: %s", err)
	}

	// Clear any existing entries in voice_current table
	log.Println("Lösche verbliebene Einträge der Tabelle für laufende Telefonate")
	_, err = db.Exec(fmt.Sprintf(`
		TRUNCATE TABLE voice_current_%s
	`, guildID))
	if err != nil {
		return fmt.Errorf("problem truncating voice_current table: %s", err)
	}

	return nil
}

func SaveMessage(guildID, userID, channelID string) error {
	// Current date and time
	now := time.Now()

	// Open database connection
	db, err := openDBConnection()
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %v", err)
	}
	defer db.Close()

	// Insert message into chat table
	_, err = db.Exec(fmt.Sprintf(`
		INSERT INTO chat_%s (channel_id, user_id, time)
		VALUES (?, ?, ?)
	`, guildID), channelID, userID, now.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("problem saving message: %v", err)
	}
	return nil
}

func SaveVoiceEvent(guildID, userID, channelID string) error {
	// Current date and time
	now := time.Now()
	log.Printf("Erfasse aktuelle Zeit. Es ist " + now.String())

	// Open database connection
	log.Printf("Stelle Verbindung zur Datenbank her")
	db, err := openDBConnection()
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %v", err)
	}
	defer db.Close()

	log.Printf("Connected to MySQL database for guild: %s\n", guildID)

	// Determine the type of voice event
	log.Printf("Bestimme den Typ des Voice-Events")
	var voiceEventType string
	if channelID != "" {
		// Check if user already exists in voice_current table
		log.Printf("Überprüfe, ob bereits ein Eintrag für den Nutzer in voice_current existiert")
		var currentChannelID string
		err := db.QueryRow(fmt.Sprintf("SELECT channel_id FROM voice_current_%s WHERE user_id = ?", guildID), userID).Scan(&currentChannelID)

		switch {
		case err == sql.ErrNoRows:
			voiceEventType = "join"
			log.Printf("Es ist kein Eintrag in voice_current vorhanden -> JOIN")
			log.Printf("User %s joined voice channel %s\n", userID, channelID)
		case err != nil:
			log.Printf("Fehler beim Abruf von Einträgen")
			return fmt.Errorf("problem retrieving current voice presence: %v", err)
		default:
			if currentChannelID != channelID {
				voiceEventType = "move"
				log.Printf("Es ist ein Eintrag vorhanden für den Nutzer und die IDs der Channel stimmen nicht überein -> MOVE")
				log.Printf("User %s moved to voice channel %s from %s\n", userID, channelID, currentChannelID)
			} else {
				return nil // Ignore the event if channel ID hasn't changed
			}
		}
	} else {
		voiceEventType = "leave"
		log.Printf("Es ist keine Channel ID übergeben worden -> LEAVE")
		log.Printf("User %s left the voice channel\n", userID)
	}

	// Handle different voice event types
	switch voiceEventType {
	case "join":
		log.Println("Da es ein JOIN-Event ist wird passend weiter verfahren")
		log.Println("Das Event wird in die voice_current Tabelle eingetragen")
		// Insert entry into voice_current table
		_, err := db.Exec(fmt.Sprintf(`
			INSERT INTO voice_current_%s (channel_id, user_id, start)
			VALUES (?, ?, ?)
		`, guildID), channelID, userID, now.Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("problem saving voice event: %v", err)
		}
		log.Printf("Voice event saved: User %s joined voice channel %s\n", userID, channelID)
	case "leave", "move":
		log.Printf("Da es ein LEAVE- oder MOVE-Event ist wird passend weiter verfahren")
		var startStr string
		var start time.Time
		var duration int

		// Retrieve start time of current voice presence
		log.Printf("Die Startzeit des vorherigen Voice-Events wird aus voice_current gelesen")
		err := db.QueryRow(fmt.Sprintf("SELECT start FROM voice_current_%s WHERE user_id = ?", guildID), userID).Scan(&startStr)
		if err != nil {
			return fmt.Errorf("problem retrieving start time of current voice presence: %v", err)
		}

		// Refactor the startStr into time obj
		start, err = time.Parse("2006-01-02 15:04:05", startStr)
		if err != nil {
			return fmt.Errorf("Error parsing start time: %v", err)
		}

		// Calculate duration in minutes
		log.Println("Die Länge des vergangenen Events wird berechnet")
		duration = int(now.Sub(start).Minutes())
		log.Printf("Die Länge ist (in Minuten): " + strconv.Itoa(duration))

		// Wenn negative Zahl, dann vermutlich weniger als 1 Minute im Call -> Setze auf 1
		if duration <1 {
			duration = 1
		}

		// Insert entry into voice table
		log.Println("Das vergangene abgeschlossene Event wird in die voice-Tabelle eingetragen")
		_, err = db.Exec(fmt.Sprintf(`
			INSERT INTO voice_%s (user_id, channel_id, duration)
			VALUES (?, ?, ?)
		`, guildID), userID, channelID, duration)
		if err != nil {
			return fmt.Errorf("problem saving voice event: %v", err)
		}
		log.Printf("Voice event saved: User %s left/moved from voice channel %s after %d minutes\n", userID, channelID, duration)

		// Delete entry from voice_current table
		log.Printf("Der nicht mehr benötigte Eintrag über das vergangene Event wird aus voice_current gelöscht")
		_, err = db.Exec(fmt.Sprintf("DELETE FROM voice_current_%s WHERE user_id = ?", guildID), userID)
		if err != nil {
			return fmt.Errorf("problem deleting current voice presence: %v", err)
		}
		log.Printf("Voice event deleted from voice_current: User %s\n", userID)

		// If it's a move event, insert new entry into voice_current table
		if voiceEventType == "move" {
			log.Printf("Da es ein MOVE-Event ist, wird zusätzlich noch der Eintrag in voice_current ersetzt.")
			_, err = db.Exec(fmt.Sprintf(`
				INSERT INTO voice_current_%s (channel_id, user_id, start)
				VALUES (?, ?, ?)
			`, guildID), channelID, userID, now.Format(time.RFC3339))
			if err != nil {
				return fmt.Errorf("problem saving voice event: %v", err)
			}
			log.Printf("Voice event saved: User %s moved to voice channel %s\n", userID, channelID)
		}
	default:
		log.Println("Es handelt sich um einen unbekannten Typ")
		return fmt.Errorf("invalid voice event type")
	}
	return nil
}
