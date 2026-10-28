package port

import "context"

type Health struct {
	OK         bool           `json:"ok"`
	Configured bool           `json:"configured,omitempty"`
	Error      string         `json:"error,omitempty"`
	Extra      map[string]any `json:"-"`
}

type Model struct {
	ID     string         `json:"id"`
	Media  string         `json:"media,omitempty"`
	Label  string         `json:"label,omitempty"`
	Modes  []string       `json:"modes,omitempty"` // capability catalog: t2v, i2v, flf, r2v, v2v
	MaxSec float64        `json:"max_duration_sec,omitempty"`
	Sizes  []string       `json:"sizes,omitempty"`
	Extra  map[string]any `json:"extra,omitempty"`
}

// ImageMode is Film's canonical image-generation task kind.
type ImageMode string

const (
	ImageModeT2I     ImageMode = "t2i"     // text-to-image
	ImageModeI2I     ImageMode = "i2i"     // reference / img2img
	ImageModeEdit    ImageMode = "edit"    // instruction edit on a source image
	ImageModeInpaint ImageMode = "inpaint" // masked local redraw
)

type ImageRequest struct {
	Model           string
	Prompt          string
	NegativePrompt  string
	Mode            ImageMode
	Size            string
	Seed            int
	SourceID        string   // primary image for edit / inpaint / single-ref i2i
	MaskID          string   // inpaint
	ReferenceImages []string // i2i multi-ref
	Raw             map[string]any
}

// VideoMode is a capability catalog token derived from filled media slots.
// It is not an HTTP request field — task kind is the role set on VideoRequest.
// Do not merge r2v into t2v — reference images are a distinct exclusive group
// from first/last frames (Wan 3 / Seedance mutual exclusion).
type VideoMode string

const (
	VideoModeT2V VideoMode = "t2v" // text only
	VideoModeI2V VideoMode = "i2v" // lock first frame
	VideoModeFLF VideoMode = "flf" // first + last frame
	VideoModeR2V VideoMode = "r2v" // reference images (+ optional ref videos in same exclusive group)
	VideoModeV2V VideoMode = "v2v" // reference / continuation video
)

// VideoRequest is Film's canonical video job. Task kind is which slots are filled
// (first/last frames vs reference images/videos). Slots are mutually exclusive groups.
type VideoRequest struct {
	Model             string
	Prompt            string
	NegativePrompt    string
	DurationSec       float64
	Size              string
	Resolution        string
	Ratio             string
	FPS               float64
	NumFrames         int
	Seed              int
	FirstFrame        string   // staged id / URL / data URI
	LastFrame         string
	ReferenceImages   []string // reference images (exclusive vs first/last)
	ReferenceVideos   []string // reference / continuation videos
	SourceVideo       string   // primary clip for continuation (alias of first reference video)
	Raw               map[string]any
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
// Video: CreateVideo / EditVideo take filled media slots (no body.mode).
// Task kind is derived from the role set; model capability lives in modes[].
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
