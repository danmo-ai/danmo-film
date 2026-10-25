package port

import "context"

type Health struct {
	OK         bool           `json:"ok"`
	Configured bool           `json:"configured,omitempty"`
	Error      string         `json:"error,omitempty"`
	Extra      map[string]any `json:"-"`
}

type Model struct {
	ID         string         `json:"id"`
	Media      string         `json:"media,omitempty"`
	Label      string         `json:"label,omitempty"`
	Modes      []string       `json:"modes,omitempty"` // t2v, i2v, flf, t2i
	MaxSec     float64        `json:"max_duration_sec,omitempty"`
	Sizes      []string       `json:"sizes,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
}

type ImageRequest struct {
	Model    string
	Prompt   string
	Size     string
	Raw      map[string]any
	SourceID string
}

type VideoMode string

const (
	VideoModeT2V VideoMode = "t2v"
	VideoModeI2V VideoMode = "i2v"
	VideoModeFLF VideoMode = "flf"
)

type VideoRequest struct {
	Model      string
	Prompt     string
	Mode       VideoMode
	DurationSec float64
	Size       string
	FPS        float64
	NumFrames  int
	FirstFrame string // provider-native staged id, URL, or data URI
	LastFrame  string
	Raw        map[string]any
}

type Task struct {
	ID               string         `json:"id"`
	Status           string         `json:"status"`
	Kind             string         `json:"kind,omitempty"`
	Progress         float64        `json:"progress,omitempty"`
	AssetIDs         []string       `json:"asset_ids,omitempty"`
	ProviderAssetIDs []string       `json:"provider_asset_ids,omitempty"`
	Error            string         `json:"error,omitempty"`
	Raw              map[string]any `json:"-"`
}

type ProviderAsset struct {
	ID  string
	URL string
	Raw map[string]any
}

type SSEEvent struct {
	Event string
	Data  string
}

// GenerationProvider is Film's canonical image/video job API.
type GenerationProvider interface {
	ID() string
	Health(ctx context.Context) (map[string]any, error)
	ListModels(ctx context.Context, media string) ([]map[string]any, error)
	GetRegistry(ctx context.Context) (map[string]any, error)
	ListAdapters(ctx context.Context, forModel string) (any, error)

	CreateImage(ctx context.Context, body map[string]any) (map[string]any, error)
	CreateVideo(ctx context.Context, body map[string]any) (map[string]any, error)
	EditVideo(ctx context.Context, body map[string]any) (map[string]any, error)

	GetTask(ctx context.Context, taskID string) (map[string]any, error)
	DownloadResult(ctx context.Context, providerRef, dest string) error
	StageInput(ctx context.Context, path, mime string) (map[string]any, error)
	IterTaskSSE(ctx context.Context, taskID string) (<-chan SSEEvent, error)
}
