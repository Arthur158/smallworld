package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// Ce client se connecte au serveur, envoie une action, puis écoute les réponses.
	// Pour tester, lancez le serveur, puis lancez deux instances du client (par exemple: go run client.go dans deux terminaux).
	// Une fois que les deux clients sont connectés, ils devraient recevoir un message "Partie démarrée".
	// Ce client envoie ensuite une action, puis reçoit l'état mis à jour.

	// Connexion au serveur
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatal("Connexion échouée:", err)
	}
	defer conn.Close()

	// On attend un petit peu pour être sûr que l'autre client se connecte.
	// Dans un vrai cas, on gérerait ça plus proprement.
	time.Sleep(2 * time.Second)

	// On envoie une action
	err = conn.WriteJSON(Message{Type: "action", Data: "MonActionTest"})
	if err != nil {
		log.Println("error sending:", err)
		return
	}

	// On lit quelques messages (par exemple 3 messages)
	for i := 0; i < 3; i++ {
		var msg Message
		err = conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Erreur de lecture:", err)
			return
		}
		if msg.Type == "state" {
			// On peut afficher l'état
			var actions []string
			json.Unmarshal([]byte(msg.Data), &actions)
			fmt.Println("Etat du jeu reçu:", actions)
		} else {
			fmt.Println("Message reçu:", msg.Type, msg.Data)
		}
	}
}
