package telegramdalgo

import (
	"context"
	"fmt"
	"testing"

	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/dal-go/dalgo/adapters/dalgo2memory"
)

func TestChatInstanceStore_RoundTripUsesExistingKeyLayout(t *testing.T) {
	store := NewChatInstanceStore(dalgo2memory.New(dalgo2memory.FirestoreProfile()), func(string) botsfwtgmodels.TgChatInstanceData {
		return &botsfwtgmodels.TgChatInstanceBaseData{}
	})
	data := &botsfwtgmodels.TgChatInstanceBaseData{TgChatID: -100, PreferredLanguage: "en-US"}
	if err := store.Save(context.Background(), "bot-1", "instance-1", data); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	got, found, err := store.Get(context.Background(), "bot-1", "instance-1")
	if err != nil || !found {
		t.Fatalf("Get() = (%v, %v, %v), want found record", got, found, err)
	}
	if got.GetTgChatID() != -100 || got.GetPreferredLanguage() != "en-US" {
		t.Fatalf("Get() data = %#v", got)
	}
	storedKey := NewChatInstanceKey("instance-1")
	if storedKey.Collection() != chatInstancesCollection || storedKey.Parent().Collection() != "botPlatforms" || fmt.Sprint(storedKey.Parent().ID) != "telegram" {
		t.Fatalf("chat-instance key changed: %v", storedKey)
	}
}
