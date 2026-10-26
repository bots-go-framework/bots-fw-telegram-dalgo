package telegramdalgo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/dal-go/dalgo/adapters/dalgo2memory"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/record"
)

func dummyNewData(string) botsfwtgmodels.TgChatInstanceData {
	return &botsfwtgmodels.TgChatInstanceBaseData{}
}

func TestChatInstanceStore_PanicsOnNil(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic on nil db")
			}
		}()
		_ = NewChatInstanceStore(nil, dummyNewData)
	})

	t.Run("nil getDB", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic on nil getDB")
			}
		}()
		_ = NewChatInstanceStoreWithProvider(nil, dummyNewData)
	})

	t.Run("nil newData", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic on nil newData")
			}
		}()
		_ = NewChatInstanceStoreWithProvider(func(context.Context) (dal.DB, error) { return nil, nil }, nil)
	})

	t.Run("empty chatInstanceID", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic on empty chatInstanceID")
			}
		}()
		_ = NewChatInstanceKey("")
	})
}

func TestChatInstanceStore_ProviderErrors(t *testing.T) {
	t.Run("getDB error", func(t *testing.T) {
		store := NewChatInstanceStoreWithProvider(func(context.Context) (dal.DB, error) {
			return nil, errors.New("simulated getDB error")
		}, dummyNewData)

		if _, _, err := store.Get(context.Background(), "bot-1", "inst-1"); err == nil {
			t.Fatal("expected error from Get when getDB fails")
		}
		if err := store.Save(context.Background(), "bot-1", "inst-1", &botsfwtgmodels.TgChatInstanceBaseData{}); err == nil {
			t.Fatal("expected error from Save when getDB fails")
		}
	})

	t.Run("getDB returns nil db", func(t *testing.T) {
		store := NewChatInstanceStoreWithProvider(func(context.Context) (dal.DB, error) {
			return nil, nil
		}, dummyNewData)

		if _, _, err := store.Get(context.Background(), "bot-1", "inst-1"); err == nil {
			t.Fatal("expected error from Get when getDB returns nil")
		}
		if err := store.Save(context.Background(), "bot-1", "inst-1", &botsfwtgmodels.TgChatInstanceBaseData{}); err == nil {
			t.Fatal("expected error from Save when getDB returns nil")
		}
	})
}

func TestChatInstanceStore_GetCases(t *testing.T) {
	memDB := dalgo2memory.New(dalgo2memory.FirestoreProfile())

	t.Run("newData returns nil", func(t *testing.T) {
		store := NewChatInstanceStore(memDB, func(string) botsfwtgmodels.TgChatInstanceData {
			return nil
		})
		got, found, err := store.Get(context.Background(), "bot-1", "inst-1")
		if err != nil || found || got != nil {
			t.Fatalf("expected (nil, false, nil), got (%v, %v, %v)", got, found, err)
		}
	})

	t.Run("record not found", func(t *testing.T) {
		store := NewChatInstanceStore(memDB, dummyNewData)
		got, found, err := store.Get(context.Background(), "bot-1", "non-existent")
		if err != nil || found || got != nil {
			t.Fatalf("expected (nil, false, nil), got (%v, %v, %v)", got, found, err)
		}
	})

	t.Run("db.Get generic error", func(t *testing.T) {
		mockDB := mockGetErrorDB{DB: memDB}
		store := NewChatInstanceStore(mockDB, dummyNewData)
		_, _, err := store.Get(context.Background(), "bot-1", "inst-1")
		if err == nil {
			t.Fatal("expected error when db.Get fails")
		}
	})
}

type mockGetErrorDB struct {
	dal.DB
}

func (m mockGetErrorDB) Get(ctx context.Context, record record.Record) error {
	return errors.New("simulated db.Get error")
}

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

func TestConfiguredChatInstancesExcludeOtherParents(t *testing.T) {
	db := dalgo2memory.New(dalgo2memory.FirestoreProfile())
	ctx := context.Background()
	for _, parent := range []*record.Key{record.NewKeyWithID("botPlatforms", "telegram"), record.NewKeyWithParentAndID(record.NewKeyWithID("ext", "foreign"), "platforms", "telegram")} {
		if err := db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
			return tx.Set(ctx, record.NewRecordWithData(chatInstanceKey(parent, "same"), &botsfwtgmodels.TgChatInstanceBaseData{TgChatID: 77}))
		}); err != nil {
			t.Fatal(err)
		}
	}
	parent := record.NewKeyWithParentAndID(record.NewKeyWithID("ext", "owner"), "platforms", "telegram")
	store := NewChatInstanceStoreWithProviderOptions(func(context.Context) (dal.DB, error) { return db, nil }, dummyNewData, WithPlatformKey(parent))
	parent.Parent().ID = "mutated"
	if _, found, err := store.Get(ctx, "bot", "same"); err != nil || found {
		t.Fatal(found, err)
	}
	if err := store.Save(ctx, "bot", "same", &botsfwtgmodels.TgChatInstanceBaseData{TgChatID: 88}); err != nil {
		t.Fatal(err)
	}
	got, found, err := store.Get(ctx, "bot", "same")
	if err != nil || !found || got.GetTgChatID() != 88 {
		t.Fatal(got, found, err)
	}
	legacy := NewChatInstanceStore(db, dummyNewData)
	got, found, err = legacy.Get(ctx, "bot", "same")
	if err != nil || !found || got.GetTgChatID() != 77 {
		t.Fatal(got, found, err)
	}
}
func TestConfiguredChatInstanceRejectsInvalidPlatform(t *testing.T) {
	for _, key := range []*record.Key{nil, record.NewIncompleteKey("platforms", reflect.String, nil)} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("missing panic")
				}
			}()
			WithPlatformKey(key)
		}()
	}
}

type invalidPlatformID string

func (invalidPlatformID) Validate() error { return errors.New("invalid platform") }
func TestConfiguredChatInstanceRejectsInvalidTypedID(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("invalid platform accepted")
		}
	}()
	WithPlatformKey(record.NewKeyWithID("platforms", invalidPlatformID("bad")))
}
