package imvu

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// BaseResponse represents the common structure of all IMVU API responses
type BaseResponse struct {
	Status       string                `json:"status"`
	ID           string                `json:"id,omitempty"`
	Denormalized map[string]EntityData `json:"denormalized,omitempty"`
	HTTP         map[string]HTTPData   `json:"http,omitempty"`
}

// EntityData represents the data structure for an entity in the denormalized section
type EntityData struct {
	Data      json.RawMessage   `json:"data"`
	Relations map[string]string `json:"relations,omitempty"`
	Updates   map[string]string `json:"updates,omitempty"`
}

// HTTPData represents HTTP metadata for an entity
type HTTPData struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Meta    any               `json:"meta,omitempty"`
}

// User represents a user entity in the IMVU API
type User struct {
	Created               string  `json:"created"`
	Registered            int64   `json:"registered"`
	Gender                string  `json:"gender"`
	DisplayName           string  `json:"display_name"`
	Age                   *int    `json:"age"`
	Country               *string `json:"country"`
	State                 *string `json:"state"`
	AvatarImage           string  `json:"avatar_image"`
	AvatarPortraitImage   string  `json:"avatar_portrait_image"`
	IsVIP                 bool    `json:"is_vip"`
	IsAP                  bool    `json:"is_ap"`
	IsAPPlus              bool    `json:"is_ap_plus"`
	IsAPPlusFounder       bool    `json:"is_ap_plus_founder"`
	IsCreator             bool    `json:"is_creator"`
	IsAdult               bool    `json:"is_adult"`
	IsAgeVerified         bool    `json:"is_ageverified"`
	IsStaff               bool    `json:"is_staff"`
	IsGreeter             bool    `json:"is_greeter"`
	GreeterScore          int     `json:"greeter_score"`
	BadgeLevel            int     `json:"badge_level"`
	Username              string  `json:"username"`
	RelationshipStatus    int     `json:"relationship_status"`
	Orientation           int     `json:"orientation"`
	LookingFor            int     `json:"looking_for"`
	Interests             string  `json:"interests"`
	LegacyCID             int64   `json:"legacy_cid"`
	PersonaType           int     `json:"persona_type"`
	Availability          string  `json:"availability"`
	IsDiscussionModerator bool    `json:"is_discussion_moderator"`
	Online                bool    `json:"online"`
	Tagline               string  `json:"tagline"`
	ThumbnailURL          string  `json:"thumbnail_url"`
	IsHost                int     `json:"is_host"`
	HasNFT                bool    `json:"has_nft"`
	VIPTier               int     `json:"vip_tier"`
	VIPPlatform           any     `json:"vip_platform"`
	HasLegacyVIP          bool    `json:"has_legacy_vip"`
}

// UserResponse represents a response containing user data
type UserResponse struct {
	BaseResponse
	User *User `json:"-"` // Not part of JSON, populated by ParseUser
}

// ParseResponse parses an HTTP response into the given response struct
func ParseResponse(resp *http.Response, v any) error {
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

// ExtractEntity extracts and parses an entity from the denormalized data
func ExtractEntity[T any](response *BaseResponse, entityID string) (*T, error) {
	// If entityID doesn't have the full URL, try to find it by suffix
	if !strings.HasPrefix(entityID, "https://") {
		for key := range response.Denormalized {
			if strings.HasSuffix(key, entityID) {
				entityID = key
				break
			}
		}
	}

	entityData, ok := response.Denormalized[entityID]
	if !ok {
		return nil, fmt.Errorf("entity not found: %s", entityID)
	}

	var entity T
	if err := json.Unmarshal(entityData.Data, &entity); err != nil {
		return nil, fmt.Errorf("failed to parse entity data: %w", err)
	}

	return &entity, nil
}

// ParseUser parses the user data from a UserResponse
func (r *UserResponse) ParseUser() error {
	// Extract the user ID from the response ID
	userID := r.ID

	user, err := ExtractEntity[User](&r.BaseResponse, userID)
	if err != nil {
		return err
	}

	r.User = user
	return nil
}

// MeData represents the data field inside the denormalized section for the "me" endpoint
type MeData struct {
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	Sauce     string `json:"sauce"`
	SessionID string `json:"session_id"`
	Source    string `json:"source"`
}

// MeResponse represents the response from the "me" endpoint
type MeResponse struct {
	BaseResponse
	Me *MeData `json:"-"` // Populated by ParseMe
}

// ParseMe extracts and parses the MeData from the denormalized map
func (r *MeResponse) ParseMe() error {
	entityID := r.ID
	meData, err := ExtractEntity[MeData](&r.BaseResponse, entityID)
	if err != nil {
		return err
	}
	r.Me = meData
	return nil
}

// ChatParticipantData represents the data field within a chat participant entity
type ChatParticipantData struct {
	SeatNumber          int    `json:"seat_number"`
	SeatFurniID         int    `json:"seat_furni_id"`
	AssetURL            string `json:"asset_url"`
	LookImage           string `json:"look_image"`
	LookURL             string `json:"look_url"`
	RenderedImage       string `json:"rendered_image"`
	LookThumbnail       string `json:"look_thumbnail"`
	LegacyOutfitMessage string `json:"legacy_outfit_message"`
	LegacySeatMessage   string `json:"legacy_seat_message"`
	Created             string `json:"created"`
	LastUpdated         string `json:"last_updated"`
	OutfitGender        string `json:"outfit_gender"`
	NFTProductIDs       []int  `json:"nft_product_ids"`
}

// EnterChatResponse represents the response when entering a chat
type EnterChatResponse struct {
	BaseResponse
	Participant *ChatParticipantData `json:"-"` // Populated by ParseEnterChatResponse
	User        *User                `json:"-"` // Populated by ParseEnterChatResponse
}

// ParseEnterChatResponse extracts and parses the relevant data from the denormalized map
func (r *EnterChatResponse) ParseEnterChatResponse() error {
	// Extract the participant ID from the response ID
	participantID := r.ID

	// Get the entity data for the participant
	entityData, ok := r.Denormalized[participantID]
	if !ok {
		return fmt.Errorf("chat participant entity not found: %s", participantID)
	}

	// Unmarshal the data field into ChatParticipantData
	var participantData ChatParticipantData
	if err := json.Unmarshal(entityData.Data, &participantData); err != nil {
		return fmt.Errorf("failed to parse chat participant data: %w", err)
	}
	r.Participant = &participantData

	// Extract the user ID from the participant's relations
	if entityData.Relations != nil {
		if userRef, ok := entityData.Relations["ref"]; ok {
			user, err := ExtractEntity[User](&r.BaseResponse, userRef)
			if err != nil {
				// Log the error but don't fail if user data isn't strictly necessary
				log.Printf("Warning: Failed to parse user data from chat participant relations: %v", err)
			}
			r.User = user
		}
	}

	return nil
}

// StringOrInt is a type that can be unmarshalled from a JSON string or number.
type StringOrInt string

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *StringOrInt) UnmarshalJSON(data []byte) error {
	// First, try to unmarshal as a string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = StringOrInt(str)
		return nil
	}

	// If it's not a string, try to unmarshal as an integer
	var num int64
	if err := json.Unmarshal(data, &num); err == nil {
		*s = StringOrInt(strconv.FormatInt(num, 10))
		return nil
	}

	return fmt.Errorf("value must be a string or an integer")
}

// MarshalJSON implements the json.Marshaler interface.
func (s StringOrInt) MarshalJSON() ([]byte, error) {
	// try to convert to int
	if i, err := strconv.ParseInt(string(s), 10, 64); err == nil {
		return json.Marshal(i)
	}
	// otherwise, marshal as string
	return json.Marshal(string(s))
}

// String returns the string representation.
func (s StringOrInt) String() string {
	return string(s)
}

// Int converts the value to an int.
func (s StringOrInt) Int() (int, error) {
	return strconv.Atoi(string(s))
}

// Int64 converts the value to an int64.
func (s StringOrInt) Int64() (int64, error) {
	return strconv.ParseInt(string(s), 10, 64)
}

type ChatMessagePayload struct {
	ChatID  StringOrInt `json:"chatId"`
	Message string      `json:"message"`
	To      StringOrInt `json:"to"`
	UserID  StringOrInt `json:"userId"`
}

type chatMessageEncodedPayload ChatMessagePayload

// UnmarshalJSON decodes a base64 encoded JSON string into a ChatMessagePayload.
func (b *ChatMessagePayload) UnmarshalJSON(data []byte) error {
	dataStr := string(data[1 : len(data)-1])
	decodedJSON, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return fmt.Errorf("failed to decode base64 string: %w", err)
	}

	var alias chatMessageEncodedPayload
	if err := json.Unmarshal(decodedJSON, &alias); err != nil {
		return fmt.Errorf("failed to unmarshal decoded JSON payload: %w", err)
	}

	*b = ChatMessagePayload(alias)

	return nil
}

// MarshalJSON encodes the ChatMessagePayload into a base64 encoded JSON string.
func (b ChatMessagePayload) MarshalJSON() ([]byte, error) {
	payloadJSON, err := json.Marshal(chatMessageEncodedPayload(b))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload to JSON: %w", err)
	}

	base64String := base64.StdEncoding.EncodeToString(payloadJSON)

	// Wrap the base64 string in quotes to make it a valid JSON string.
	return []byte(`"` + base64String + `"`), nil
}
