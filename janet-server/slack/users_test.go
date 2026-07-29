package slack

import (
	"errors"
	"sync"
	"testing"
	"time"

	goslack "github.com/slack-go/slack"
	"github.com/troyxmccall/janet/database"
)

var errUnexpectedSlackCall = errors.New("unexpected slack client call")

type fakeSlackClient struct {
	mu            sync.Mutex
	userBatches   [][]goslack.User
	getUsersCalls int
}

func (f *fakeSlackClient) GetUsers(...goslack.GetUsersOption) ([]goslack.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	callIndex := f.getUsersCalls
	f.getUsersCalls++
	if len(f.userBatches) == 0 {
		return nil, nil
	}
	if callIndex >= len(f.userBatches) {
		callIndex = len(f.userBatches) - 1
	}
	users := make([]goslack.User, len(f.userBatches[callIndex]))
	copy(users, f.userBatches[callIndex])
	return users, nil
}

func (f *fakeSlackClient) getUsersCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.getUsersCalls
}

func (f *fakeSlackClient) GetPermalink(*goslack.PermalinkParameters) (string, error) {
	return "", errUnexpectedSlackCall
}

func (f *fakeSlackClient) GetConversationHistory(*goslack.GetConversationHistoryParameters) (*goslack.GetConversationHistoryResponse, error) {
	return nil, errUnexpectedSlackCall
}

func (f *fakeSlackClient) GetUserInfo(string) (*goslack.User, error) {
	return nil, errUnexpectedSlackCall
}

func (f *fakeSlackClient) GetConversations(*goslack.GetConversationsParameters) ([]goslack.Channel, string, error) {
	return nil, "", errUnexpectedSlackCall
}

func (f *fakeSlackClient) SearchMessages(string, goslack.SearchParameters) (*goslack.SearchMessages, error) {
	return nil, errUnexpectedSlackCall
}

func TestEnrichUsersWithSlackInfoFetchesUserListOnceForMisses(t *testing.T) {
	client := &fakeSlackClient{
		userBatches: [][]goslack.User{
			{
				{
					ID:       "U1",
					Name:     "alice",
					RealName: "Alice Adams",
					Profile: goslack.UserProfile{
						DisplayName: "Alice",
						Image192:    "https://example.com/alice.png",
					},
				},
			},
		},
	}
	svc := NewWebService(client, ServiceOptions{})

	users := []*database.UserSummary{
		{Username: "alice"},
		{Username: "renamed-user"},
		{Username: "departed-user"},
	}

	svc.EnrichUsersWithSlackInfo(users)

	if got := client.getUsersCallCount(); got != 1 {
		t.Fatalf("expected one Slack users-list call, got %d", got)
	}
	if users[0].DisplayName == nil || *users[0].DisplayName != "Alice" {
		t.Fatalf("expected alice display name from Slack, got %#v", users[0].DisplayName)
	}
	if users[0].RealName == nil || *users[0].RealName != "Alice Adams" {
		t.Fatalf("expected alice real name from Slack, got %#v", users[0].RealName)
	}
	if users[0].AvatarURL == nil || *users[0].AvatarURL != "https://example.com/alice.png" {
		t.Fatalf("expected alice avatar from Slack, got %#v", users[0].AvatarURL)
	}
	if users[0].IsDeleted {
		t.Fatal("expected matched user to remain active")
	}
	if !users[1].IsDeleted || !users[2].IsDeleted {
		t.Fatalf("expected missing users to be marked deleted, got %v and %v", users[1].IsDeleted, users[2].IsDeleted)
	}
}

func TestEnrichUsersWithSlackInfoRefreshesExpiredUserCache(t *testing.T) {
	client := &fakeSlackClient{
		userBatches: [][]goslack.User{
			{{ID: "U1", Name: "alice"}},
			{{ID: "U2", Name: "bob"}},
		},
	}
	svc := NewWebService(client, ServiceOptions{})

	firstUsers := []*database.UserSummary{{Username: "alice"}}
	svc.EnrichUsersWithSlackInfo(firstUsers)
	if got := client.getUsersCallCount(); got != 1 {
		t.Fatalf("expected initial Slack users-list call, got %d", got)
	}
	if firstUsers[0].IsDeleted {
		t.Fatal("expected alice to be found in initial cache")
	}

	svc.userCacheMu.Lock()
	svc.userCacheFetchedAt = time.Now().Add(-slackUserCacheTTL - time.Second)
	svc.userCacheMu.Unlock()

	nextUsers := []*database.UserSummary{{Username: "bob"}}
	svc.EnrichUsersWithSlackInfo(nextUsers)

	if got := client.getUsersCallCount(); got != 2 {
		t.Fatalf("expected expired cache to refresh once, got %d Slack users-list calls", got)
	}
	if nextUsers[0].IsDeleted {
		t.Fatal("expected bob to be found after cache refresh")
	}
}
