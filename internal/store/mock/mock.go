package mock

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emibotz/chat-server/internal/user"
	"github.com/google/uuid"
)

type MockSessionStore struct {
	sessions map[string]string
}

func SessionStore() user.SessionStore {
	return &MockSessionStore{
		sessions: make(map[string]string),
	}
}

func (s *MockSessionStore) Create(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error) {
	token := userID.String()
	s.sessions[token] = token

	return token, nil
}

func (s *MockSessionStore) Get(ctx context.Context, token string) (uuid.UUID, error) {
	session, ok := s.sessions[token]
	if !ok {
		return uuid.Nil, fmt.Errorf("failed to get session: no session with token %s.", session)
	}

	id, err := uuid.Parse(session)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (s *MockSessionStore) RefreshTTL(ctx context.Context, token string, ttl time.Duration) error {
	return nil
}

func (s *MockSessionStore) Delete(ctx context.Context, token string) error {
	delete(s.sessions, token)
	return nil
}

func (s *MockSessionStore) DeleteAllByUserID(ctx context.Context, userID uuid.UUID) error {

	for session, id := range s.sessions {
		if id == userID.String() {
			delete(s.sessions, session)
		}
	}

	return nil
}

type MockUserStore struct {
	users     []*user.User
	usersByID map[uuid.UUID]*user.User
}

func UserStore() user.Store {
	return &MockUserStore{
		users:     make([]*user.User, 0),
		usersByID: make(map[uuid.UUID]*user.User),
	}
}

func (s *MockUserStore) Create(ctx context.Context, user *user.User) error {
	s.users = append(s.users, user)
	s.usersByID[user.ID] = user
	return nil
}

// 通过 ID 获取用户，没有指定用户时返回 (nil, nil)
func (s *MockUserStore) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	u, ok := s.usersByID[id]
	if !ok {
		return nil, nil
	}

	return u, nil
}

// 通过多个 ID 查询多个用户，返回数量不一定和传入数量相同。
func (s *MockUserStore) GetByIDs(ctx context.Context, ids ...uuid.UUID) ([]*user.User, error) {

	result := make([]*user.User, 0)

	for _, id := range ids {

		u, ok := s.usersByID[id]
		if ok {
			result = append(result, u)
		}

	}

	return result, nil
}

// 通过用户名获取用户，没有指定用户时返回 (nil, nil)
func (s *MockUserStore) GetByName(ctx context.Context, username string) (*user.User, error) {

	for _, u := range s.users {
		if u.Name == username {
			return u, nil
		}
	}

	return nil, nil
}

func (s *MockUserStore) Update(ctx context.Context, user *user.User) error {

	u, ok := s.usersByID[user.ID]
	if !ok {
		return fmt.Errorf("no user %s %s", user.Name, user.ID.String())
	}

	u.Name = user.Name
	u.Auth = user.Auth

	return nil
}

func (s *MockUserStore) Delete(ctx context.Context, user *user.User) error {
	i := slices.Index(s.users, user)
	s.users = slices.Delete(s.users, i, i+1)

	delete(s.usersByID, user.ID)

	return nil
}
