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
)

const chatInstancesCollection = "chatInstances"

// ChatInstanceStore preserves the established
// botPlatforms/telegram/chatInstances/{chatInstance} storage layout.
type ChatInstanceStore struct {
	getDB   DBProvider
	newData func(botCode string) botsfwtgmodels.TgChatInstanceData
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
	if getDB == nil {
		panic("getDB is required")
	}
	if newData == nil {
		panic("newData is required")
	}
	return &ChatInstanceStore{getDB: getDB, newData: newData}
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
func NewChatInstanceKey(chatInstanceID string) *dal.Key {
	if chatInstanceID == "" {
		panic("chat instance ID is required")
	}
	platformKey := dal.NewKeyWithID("botPlatforms", telegram.PlatformID)
	return dal.NewKeyWithParentAndID(platformKey, chatInstancesCollection, chatInstanceID)
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
	if err := db.Get(ctx, dal.NewRecordWithData(NewChatInstanceKey(chatInstanceID), data)); err != nil {
		if dal.IsNotFound(err) {
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
		return tx.Set(ctx, dal.NewRecordWithData(NewChatInstanceKey(chatInstanceID), data))
	})
}
