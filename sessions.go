package main

type SessionProvider interface {
	SaveOrUpdate(newSession *session)
	TryGet(chatId int64) (*session, bool)
}

type sessions map[int64]*session

type session struct {
	chatId        int64
	oauthToken    *token
	csrfToken     *token
	defaultDevice *device
}

type token struct {
	value     string
	expiresIn *int
}

func NewToken(value string, expiresIn *int) *token {
	return &token{value, expiresIn}
}

func NewSession(chatId int64, oauthToken *token, csrfToken *token) *session {
	return &session{chatId, oauthToken, csrfToken, nil}
}

func NewSessionWithDevice(s *session, d *device) *session {
	return &session{s.chatId, s.oauthToken, s.csrfToken, d}
}

//goland:noinspection GoExportedFuncWithUnexportedType
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
