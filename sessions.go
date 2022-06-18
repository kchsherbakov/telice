package main

type SessionProvider interface {
	SaveOrUpdate(newSession *session)
	TryGet(chatId int64) (*session, bool)
}

type sessions map[int64]*session

type session struct {
	chatId int64
	client *YandexClient
}

type token struct {
	value     string
	expiresIn *int
}

func NewToken(value string, expiresIn *int) *token {
	return &token{value, expiresIn}
}

func NewSession(chatId int64, client *YandexClient) *session {
	return &session{chatId, client}
}

func NewInMemorySessionProvider() sessions {
	return make(sessions)
}

func (ss sessions) SaveOrUpdate(newSession *session) {
	ss[newSession.chatId] = newSession
}

func (ss sessions) TryGet(chatId int64) (*session, bool) {
	s, ok := ss[chatId]
	return s, ok
}
