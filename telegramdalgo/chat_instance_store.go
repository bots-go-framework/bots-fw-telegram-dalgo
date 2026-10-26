// Package telegramdalgo contains the DALgo implementation of Telegram's
// platform-specific persistence port.
package telegramdalgo

import (
	"context"
	"errors"
	"fmt"

	"github.com/bots-go-framework/bots-fw-telegram-models/botsfwtgmodels"
	"github.com/bots-go-framework/bots-fw-telegram/telegram"
	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/record"
)

const chatInstancesCollection = "chatInstances"

// ChatInstanceStore stores chat instances beneath its configured platform.
// Unconfigured stores retain botPlatforms/telegram for existing deployments.
type ChatInstanceStore struct {
	getDB       DBProvider
	newData     func(botCode string) botsfwtgmodels.TgChatInstanceData
	platformKey *record.Key
}

// DBProvider returns the DALgo database for a request context.
type DBProvider func(ctx context.Context) (dal.DB, error)

var _ telegram.ChatInstanceStore = (*ChatInstanceStore)(nil)

func NewChatInstanceStore(db dal.DB, newData func(botCode string) botsfwtgmodels.TgChatInstanceData) *ChatInstanceStore {
	if db == nil {
		panic("db is required")
	}
	return NewChatInstanceStoreWithProvider(func(context.Context) (dal.DB, error) { return db, nil }, newData)
}

// NewChatInstanceStoreWithProvider creates a store whose database is selected
// for each request.
func NewChatInstanceStoreWithProvider(getDB DBProvider, newData func(botCode string) botsfwtgmodels.TgChatInstanceData) *ChatInstanceStore {
	return NewChatInstanceStoreWithProviderOptions(getDB, newData)
}

// NewChatInstanceStoreWithProviderOptions configures one request-scoped store.
func NewChatInstanceStoreWithProviderOptions(getDB DBProvider, newData func(botCode string) botsfwtgmodels.TgChatInstanceData, opts ...ChatInstanceStoreOption) *ChatInstanceStore {
	if getDB == nil {
		panic("getDB is required")
	}
	if newData == nil {
		panic("newData is required")
	}
	s := &ChatInstanceStore{getDB: getDB, newData: newData, platformKey: record.NewKeyWithID("botPlatforms", telegram.PlatformID)}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *ChatInstanceStore) db(ctx context.Context) (dal.DB, error) {
	db, err := s.getDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("get DALgo database: %w", err)
	}
	if db == nil {
		return nil, errors.New("DALgo database provider returned nil")
	}
	return db, nil
}

// NewChatInstanceKey creates the historical Telegram chat-instance key.
func NewChatInstanceKey(chatInstanceID string) *record.Key {
	return chatInstanceKey(record.NewKeyWithID("botPlatforms", telegram.PlatformID), chatInstanceID)
}

func chatInstanceKey(platformKey *record.Key, chatInstanceID string) *record.Key {
	if chatInstanceID == "" {
		panic("chat instance ID is required")
	}
	return record.NewKeyWithParentAndID(platformKey, chatInstancesCollection, chatInstanceID)
}

func (s *ChatInstanceStore) Get(ctx context.Context, botCode, chatInstanceID string) (botsfwtgmodels.TgChatInstanceData, bool, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, false, err
	}
	data := s.newData(botCode)
	if data == nil {
		return nil, false, nil
	}
	if err := db.Get(ctx, record.NewRecordWithData(chatInstanceKey(s.platformKey, chatInstanceID), data)); err != nil {
		if record.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

func (s *ChatInstanceStore) Save(ctx context.Context, _ string, chatInstanceID string, data botsfwtgmodels.TgChatInstanceData) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	return db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return tx.Set(ctx, record.NewRecordWithData(chatInstanceKey(s.platformKey, chatInstanceID), data))
	})
}

// ChatInstanceStoreOption configures one store without changing global defaults.
type ChatInstanceStoreOption func(*ChatInstanceStore)

// WithPlatformKey fixes the exact owner for this store, with no fallback reads.
func WithPlatformKey(key *record.Key) ChatInstanceStoreOption {
	if key == nil {
		panic("platform key is required")
	}
	for parent := key; parent != nil; parent = parent.Parent() {
		if parent.ID == nil {
			panic("complete platform key is required")
		}
		if err := parent.Validate(); err != nil {
			panic(err)
		}
	}
	saved := copyKey(key)
	return func(s *ChatInstanceStore) { s.platformKey = copyKey(saved) }
}
func copyKey(key *record.Key) *record.Key {
	if key == nil {
		return nil
	}
	return record.NewKeyWithParentAndID(copyKey(key.Parent()), key.Collection(), key.ID)
}
