package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kanshi-dev/core/internal/db"
)

var (
	ErrInvalidProfile  = errors.New("invalid profile capture")
	ErrProfileNotFound = errors.New("profile capture not found")
	ErrProfileConflict = errors.New("agent already has an active profile capture")
	ErrTargetNotFound  = errors.New("profile target not reported by agent")
)

type ProfileCaptureInput struct {
	TargetName      string `json:"targetName"`
	ProfileType     string `json:"profileType"`
	DurationSeconds int16  `json:"durationSeconds"`
}

type ProfileCapture struct {
	ID              string    `json:"id"`
	AgentID         string    `json:"agentId"`
	TargetName      string    `json:"targetName"`
	ProfileType     string    `json:"profileType"`
	DurationSeconds int16     `json:"durationSeconds"`
	State           string    `json:"state"`
	Error           string    `json:"error,omitempty"`
	Filename        string    `json:"filename,omitempty"`
	ContentType     string    `json:"contentType,omitempty"`
	Size            int       `json:"size"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	ExpiresAt       time.Time `json:"expiresAt"`
}

type ProfileArtifact struct {
	Filename    string
	ContentType string
	Data        []byte
}

type ProfilesService struct{ queries *db.Queries }

func NewProfilesService(q *db.Queries) *ProfilesService { return &ProfilesService{queries: q} }

func (in ProfileCaptureInput) validate() error {
	validDuration := false
	switch in.ProfileType {
	case "cpu":
		validDuration = in.DurationSeconds == 5 || in.DurationSeconds == 10 || in.DurationSeconds == 30
	case "trace":
		validDuration = in.DurationSeconds == 1 || in.DurationSeconds == 5
	case "heap", "allocs", "goroutine", "mutex", "block", "threadcreate":
		validDuration = in.DurationSeconds == 0
	default:
		return fmt.Errorf("%w: unsupported profile type", ErrInvalidProfile)
	}
	if !validDuration {
		return fmt.Errorf("%w: invalid duration for %s", ErrInvalidProfile, in.ProfileType)
	}
	if strings.TrimSpace(in.TargetName) == "" {
		return fmt.Errorf("%w: targetName is required", ErrInvalidProfile)
	}
	return nil
}

func (s *ProfilesService) Create(ctx context.Context, agentID string, in ProfileCaptureInput) (ProfileCapture, error) {
	if s.queries == nil {
		return ProfileCapture{}, ErrNoDatabase
	}
	if agentID == "" || len(agentID) > 255 {
		return ProfileCapture{}, fmt.Errorf("%w: invalid agent id", ErrInvalidProfile)
	}
	if err := in.validate(); err != nil {
		return ProfileCapture{}, err
	}
	raw, err := s.queries.GetAgentProfileTargets(ctx, agentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProfileCapture{}, ErrTargetNotFound
	}
	if err != nil {
		return ProfileCapture{}, err
	}
	var targets []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(raw), &targets); err != nil {
		return ProfileCapture{}, err
	}
	found := false
	for _, target := range targets {
		found = found || target.Name == in.TargetName
	}
	if !found {
		return ProfileCapture{}, ErrTargetNotFound
	}
	row, err := s.queries.CreateProfileCapture(ctx, db.CreateProfileCaptureParams{
		ID: uuid.NewString(), AgentID: agentID, TargetName: in.TargetName,
		ProfileType: in.ProfileType, DurationSeconds: in.DurationSeconds,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ProfileCapture{}, ErrProfileConflict
	}
	if err != nil {
		return ProfileCapture{}, err
	}
	return mapProfile(row), nil
}

func (s *ProfilesService) List(ctx context.Context, agentID string, limit int32) ([]ProfileCapture, error) {
	if s.queries == nil {
		return nil, ErrNoDatabase
	}
	if agentID == "" || limit < 1 || limit > 100 {
		return nil, ErrInvalidProfile
	}
	rows, err := s.queries.ListProfileCapturesByAgent(ctx, db.ListProfileCapturesByAgentParams{AgentID: agentID, Lim: limit})
	if err != nil {
		return nil, err
	}
	out := make([]ProfileCapture, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapProfile(row))
	}
	return out, nil
}

func (s *ProfilesService) Get(ctx context.Context, id string) (ProfileCapture, error) {
	row, err := s.get(ctx, id)
	if err != nil {
		return ProfileCapture{}, err
	}
	return mapProfile(row), nil
}

func (s *ProfilesService) Download(ctx context.Context, id string) (ProfileArtifact, error) {
	row, err := s.get(ctx, id)
	if err != nil {
		return ProfileArtifact{}, err
	}
	if row.State != "completed" || len(row.Artifact) == 0 {
		return ProfileArtifact{}, ErrProfileNotFound
	}
	return ProfileArtifact{Filename: row.Filename.String, ContentType: row.ContentType.String, Data: row.Artifact}, nil
}

func (s *ProfilesService) get(ctx context.Context, id string) (db.ProfileCapture, error) {
	if s.queries == nil {
		return db.ProfileCapture{}, ErrNoDatabase
	}
	if id == "" {
		return db.ProfileCapture{}, ErrProfileNotFound
	}
	row, err := s.queries.GetProfileCapture(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.ProfileCapture{}, ErrProfileNotFound
	}
	return row, err
}

func mapProfile(row db.ProfileCapture) ProfileCapture {
	return ProfileCapture{ID: row.ID, AgentID: row.AgentID, TargetName: row.TargetName, ProfileType: row.ProfileType,
		DurationSeconds: row.DurationSeconds, State: row.State, Error: row.Error.String,
		Filename: row.Filename.String, ContentType: row.ContentType.String, Size: len(row.Artifact),
		CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time, ExpiresAt: row.ExpiresAt.Time}
}
