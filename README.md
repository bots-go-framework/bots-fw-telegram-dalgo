# bots-fw-telegram-dalgo

DALgo implementation of Telegram's `ChatInstanceStore`. The adapter retains the
existing `botPlatforms/telegram/chatInstances/{chatInstance}` key layout, so
adoption changes composition only; no Telegram records require migration.
