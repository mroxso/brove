package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/fiatjaf/eventstore/postgresql"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip86"
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	// create the relay instance
	relay := khatru.NewRelay()

	// set up some basic properties (will be returned on the NIP-11 endpoint)
	relay.Info.Name = getEnv("RELAY_NAME", "brove relay")
	relay.Info.PubKey = getEnv("RELAY_PUBKEY", "82c1b69ddb84fb9a8cc68616118a9a1c794dfeb29c8d2ea2cec59af21f9df804")
	relay.Info.Description = getEnv("RELAY_DESCRIPTION", "this is my custom and private relay")
	relay.Info.Icon = getEnv("RELAY_ICON", "https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Fliquipedia.net%2Fcommons%2Fimages%2F3%2F35%2FSCProbe.jpg&f=1&nofb=1&ipt=0cbbfef25bce41da63d910e86c3c343e6c3b9d63194ca9755351bb7c2efa3359&ipo=images")
	relay.Info.Version = "0.1.0"
	relay.Info.Software = "https://github.com/mroxso/brove"

	// Initialize the event store database
	db := postgresql.PostgresBackend{DatabaseURL: "postgresql://postgres:postgres@db:5432/khatru-relay?sslmode=disable"}
	if err := db.Init(); err != nil {
		panic(err)
	}

	// Initialize the normal database manager for other data
	dbManager, err := NewDBManager("postgresql://postgres:postgres@db:5432/khatru-relay?sslmode=disable")
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize database manager: %v", err))
	}
	defer dbManager.Close()

	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.CountEvents = append(relay.CountEvents, db.CountEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)
	relay.ReplaceEvent = append(relay.ReplaceEvent, db.ReplaceEvent)

	relay.RejectEvent = append(relay.RejectEvent,
		// built-in policies
		policies.ValidateKind,
		policies.PreventLargeTags(100),

		func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
			ownerPubKey := getEnv("RELAY_PUBKEY", "")
			// Check if the pubkey is allowed in the database
			isAllowed, err := dbManager.IsAllowedPubkey(event.PubKey)
			if err != nil {
				log.Printf("Error checking if pubkey is allowed: %v", err)
				return true, "error checking authorization"
			}

			if isAllowed || event.PubKey == ownerPubKey {
				return false, "" // allowed pubkey or owner can write
			}
			return true, "this is a private relay, only authorized users can write here"
		},
	)

	// you can request auth by rejecting an event or a request with the prefix "auth-required: "
	relay.RejectFilter = append(relay.RejectFilter,
		// built-in policies
		policies.NoComplexFilters,

		// define your own policies
		func(ctx context.Context, filter nostr.Filter) (reject bool, msg string) {
			ownerPubKey := getEnv("RELAY_PUBKEY", "")
			if pubkey := khatru.GetAuthed(ctx); pubkey != "" {
				log.Printf("request from %s\n", pubkey)
				// Check if the authenticated pubkey is allowed in the database
				isAllowed, err := dbManager.IsAllowedPubkey(pubkey)
				if err != nil {
					log.Printf("Error checking if pubkey is allowed: %v", err)
					return true, "error checking authorization"
				}

				if isAllowed || pubkey == ownerPubKey {
					return false, "" // allowed pubkey or owner can read
				}
				return true, "this is a private relay, only authorized users can read here"
			}
			return true, "auth-required: only authenticated users can read from this relay"
			// (this will cause an AUTH message to be sent and then a CLOSED message such that clients can
			//  authenticate and then request again)
		},
	)

	// management endpoints
	relay.ManagementAPI.RejectAPICall = append(relay.ManagementAPI.RejectAPICall,
		func(ctx context.Context, mp nip86.MethodParams) (reject bool, msg string) {
			user := khatru.GetAuthed(ctx)
			ownerPubKey := getEnv("RELAY_PUBKEY", "")
			if user != ownerPubKey {
				return true, "go away, intruder"
			}
			return false, ""
		})

	relay.ManagementAPI.AllowPubKey = func(ctx context.Context, pubkey string, reason string) error {
		return dbManager.AddAllowedPubkey(pubkey, reason)
	}

	relay.ManagementAPI.BanPubKey = func(ctx context.Context, pubkey string, reason string) error {
		return dbManager.RemoveAllowedPubkey(pubkey)
	}

	relay.ManagementAPI.ListAllowedPubKeys = func(ctx context.Context) ([]nip86.PubKeyReason, error) {
		pubkeys, err := dbManager.GetAllowedPubkeys()
		if err != nil {
			return nil, err
		}

		var result []nip86.PubKeyReason
		for _, pubkey := range pubkeys {
			result = append(result, nip86.PubKeyReason{
				PubKey: pubkey,
				Reason: "", // If you have a reason stored, use it here
			})
		}
		return result, nil
	}

	relay.ManagementAPI.ListBannedPubKeys = func(ctx context.Context) ([]nip86.PubKeyReason, error) {
		// We do not ban here since this is an allow only relay
		// Our ban is simply not allowing the pubkey to write
		return nil, nil
	}

	// mux := relay.Router()
	// set up other http handlers
	// mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	// Get the directory where the current executable is located
	// 	execPath, err := os.Executable()
	// 	if err != nil {
	// 		log.Printf("Error getting executable path: %v", err)
	// 		http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 		return
	// 	}

	// 	// Get the directory of the executable
	// 	execDir := filepath.Dir(execPath)
	// 	indexPath := filepath.Join(execDir, "index.html")

	// 	// Check if the file exists
	// 	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
	// 		// Fallback to current working directory
	// 		indexPath = "index.html"
	// 		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
	// 			http.Error(w, "index.html not found", http.StatusNotFound)
	// 			return
	// 		}
	// 	}

	// 	http.ServeFile(w, r, indexPath)
	// })

	// start the server
	fmt.Println("running on :3334")
	http.ListenAndServe(":3334", relay)
}
