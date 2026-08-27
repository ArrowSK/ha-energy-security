package ha

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const supervisorWebSocketURL = "ws://supervisor/core/websocket"

// State is the subset of a Home Assistant entity state used by the topology learner.
type State struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes"`
	LastChanged time.Time      `json:"last_changed"`
	LastUpdated time.Time      `json:"last_updated"`
}

// EntityRegistryEntry is the subset of the entity registry needed to distinguish
// physical device entities from template or virtual accounting entities.
type EntityRegistryEntry struct {
	EntityID     string  `json:"entity_id"`
	DeviceID     string  `json:"device_id"`
	Platform     string  `json:"platform"`
	OriginalName string  `json:"original_name"`
	Name         *string `json:"name"`
	DisabledBy   *string `json:"disabled_by"`
	HiddenBy     *string `json:"hidden_by"`
}

// DeviceRegistryEntry carries stable device identity and a human-friendly name.
type DeviceRegistryEntry struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	NameByUser   *string `json:"name_by_user"`
	Manufacturer string  `json:"manufacturer"`
	Model        string  `json:"model"`
}

// RegistrySnapshot is delivered once at the beginning of every websocket session.
type RegistrySnapshot struct {
	States   []State
	Entities []EntityRegistryEntry
	Devices  []DeviceRegistryEntry
}

// StateChange is a normalized Home Assistant state_changed event.
type StateChange struct {
	EntityID  string
	OldState  *State
	NewState  *State
	TimeFired time.Time
}

type wsResult struct {
	ID      int             `json:"id"`
	Type    string          `json:"type"`
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func wsWriteJSON(conn *websocket.Conn, v any) error {
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteJSON(v)
}

func wsRequest(conn *websocket.Conn, id int, typ string, out any) error {
	if err := wsWriteJSON(conn, map[string]any{"id": id, "type": typ}); err != nil {
		return err
	}
	for {
		var msg wsResult
		if err := conn.ReadJSON(&msg); err != nil {
			return err
		}
		if msg.ID != id || msg.Type != "result" {
			continue
		}
		if !msg.Success {
			return fmt.Errorf("home assistant websocket %s failed: %s: %s", typ, msg.Error.Code, msg.Error.Message)
		}
		if out == nil || len(msg.Result) == 0 || string(msg.Result) == "null" {
			return nil
		}
		if err := json.Unmarshal(msg.Result, out); err != nil {
			return fmt.Errorf("decode home assistant websocket %s result: %w", typ, err)
		}
		return nil
	}
}

// WatchStates opens one authenticated Home Assistant websocket session, fetches
// the initial state and registries, then streams state_changed events until the
// context ends or the socket fails. Callers are expected to reconnect.
func (c *Client) WatchStates(ctx context.Context, ready func(RegistrySnapshot), changed func(StateChange)) error {
	if strings.TrimSpace(c.token) == "" {
		return fmt.Errorf("home assistant websocket token is unavailable")
	}

	headers := http.Header{}
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, supervisorWebSocketURL, headers)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("connect home assistant websocket: http %d: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("connect home assistant websocket: %w", err)
	}
	defer conn.Close()
	conn.SetReadLimit(32 << 20)

	closed := make(chan struct{})
	defer close(closed)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-closed:
		}
	}()

	var auth struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	if err := conn.ReadJSON(&auth); err != nil {
		return fmt.Errorf("read home assistant websocket auth challenge: %w", err)
	}
	if auth.Type != "auth_required" {
		return fmt.Errorf("unexpected home assistant websocket handshake %q", auth.Type)
	}
	if err := wsWriteJSON(conn, map[string]any{"type": "auth", "access_token": c.token}); err != nil {
		return fmt.Errorf("send home assistant websocket auth: %w", err)
	}
	if err := conn.ReadJSON(&auth); err != nil {
		return fmt.Errorf("read home assistant websocket auth result: %w", err)
	}
	if auth.Type != "auth_ok" {
		if auth.Message == "" {
			auth.Message = auth.Type
		}
		return fmt.Errorf("home assistant websocket authentication failed: %s", auth.Message)
	}

	var states []State
	if err := wsRequest(conn, 1, "get_states", &states); err != nil {
		return err
	}
	var entities []EntityRegistryEntry
	if err := wsRequest(conn, 2, "config/entity_registry/list", &entities); err != nil {
		return err
	}
	var devices []DeviceRegistryEntry
	if err := wsRequest(conn, 3, "config/device_registry/list", &devices); err != nil {
		return err
	}

	if err := wsWriteJSON(conn, map[string]any{"id": 4, "type": "subscribe_events", "event_type": "state_changed"}); err != nil {
		return err
	}
	var subscribed wsResult
	for {
		if err := conn.ReadJSON(&subscribed); err != nil {
			return err
		}
		if subscribed.ID == 4 && subscribed.Type == "result" {
			break
		}
	}
	if !subscribed.Success {
		return fmt.Errorf("subscribe home assistant state changes failed: %s: %s", subscribed.Error.Code, subscribed.Error.Message)
	}

	if ready != nil {
		ready(RegistrySnapshot{States: states, Entities: entities, Devices: devices})
	}

	for {
		var raw struct {
			ID    int    `json:"id"`
			Type  string `json:"type"`
			Event struct {
				EventType string `json:"event_type"`
				TimeFired time.Time `json:"time_fired"`
				Data      struct {
					EntityID string `json:"entity_id"`
					OldState *State `json:"old_state"`
					NewState *State `json:"new_state"`
				} `json:"data"`
			} `json:"event"`
		}
		if err := conn.ReadJSON(&raw); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if raw.Type != "event" || raw.Event.EventType != "state_changed" || raw.Event.Data.EntityID == "" {
			continue
		}
		if changed != nil {
			changed(StateChange{
				EntityID:  raw.Event.Data.EntityID,
				OldState:  raw.Event.Data.OldState,
				NewState:  raw.Event.Data.NewState,
				TimeFired: raw.Event.TimeFired,
			})
		}
	}
}
