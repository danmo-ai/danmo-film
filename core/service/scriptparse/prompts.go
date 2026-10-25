package scriptparse

const jsonOutputCritical = `Your entire reply must be one raw JSON object parseable by json.Unmarshal().
- Start with { and end with }.
- No markdown fences, no preamble, no explanation, no English meta-commentary.
- Begin your reply with { — do not plan or reason before the JSON.
- Keep each string value concise (one line where possible).`

const jsonSchemaInstruction = `Replace every <…> in the schema with real content from the user message. The <…> text describes each field — do not copy it literally.`

const DecomposeSystem = `# Role

Decompose script text into a structured story artifact for segmented I2V production.

## Output (critical)

` + jsonOutputCritical + `

` + jsonSchemaInstruction + `

Schema:

` + "```json\n" + `{
  "title": "<chapter or story title>",
  "synopsis": "<2-3 sentences>",
  "mood": "<emotional tone>",
  "style_anchor": "<palette, lens, texture — one line>",
  "beats": [
    {
      "index": 0,
      "title": "<scene title>",
      "location": "<place · time>",
      "narrative": "<what happens; full character names; no wardrobe tags>",
      "enhancement_cues": ["<visual intent>"],
      "suggested_shot_size": "<wide|medium|close-up>",
      "estimated_duration_sec": 8
    }
  ],
  "characters": [
    {
      "name": "<full name>",
      "role": "protagonist|supporting|extra",
      "looks": [{"label": "<look slug>", "body": "<Role | Appearance | Wardrobe>"}]
    }
  ],
  "scenes": [
    {
      "name": "<location name>",
      "looks": [{"label": "default", "body": "<environment description>"}]
    }
  ]
}` + "\n```" + `

## Rules

- **beats**: 2–24 items; one beat = one story moment; **enhancement_cues** required (≥1 per beat).
- **characters**: every on-screen named person; exactly one **protagonist**; each character ≥1 look with non-empty **body**.
- **scenes**: one entry per distinct location in beats; each scene ≥1 look.
- Use **full character names** in beat narrative — no wardrobe tags.
- Do not embed style/mood boilerplate inside beat narratives.`

const BeatPlanSystem = `# Role

Plan I2V segments per beat: visibility timeline, anchor splits, spatial layout, durations.

## Output (critical)

` + jsonOutputCritical + `

` + jsonSchemaInstruction + `

Schema:

` + "```json\n" + `{
  "beats": [
    {
      "beat_index": 0,
      "shot_intent": "<why this beat exists>",
      "narrative_role": "establish_context|introduce_subject|build_tension|deliver_payload|transition|emotional_beat|evidence|comparison|resolution|call_to_action",
      "segments": [
        {
          "role": "pre_anchor|face_anchor|post_anchor|keyframe|establishing",
          "duration_sec": 5,
          "shot_size": "<景别>",
          "characters_on_screen": ["<Name>"],
          "start_visibility": "silhouette|partial|full_face|invisible",
          "end_visibility": "silhouette|partial|full_face|invisible",
          "first_frame_requirement": "<hard t=0 constraint>",
          "reachability": "identity_critical|establishing|action_wide|empty",
          "is_intentional_empty": false
        }
      ]
    }
  ]
}` + "\n```" + `

## Rules

- One row per beat_index from the user message.
- **face_anchor** segments: **start_visibility MUST be full_face**.
- Other roles: **start_visibility MUST NOT be full_face**.
- **identity_critical** beats MUST include a **face_anchor** segment.
- At most one face_anchor per beat; characters_on_screen length 1 for face_anchor.
- Segment durations **4–maxClip** (see user message soft band / model max); default hint is a fallback, not a uniform chop.
- Time from dialogue, pause, action, and reaction capacity — split long beats; do **not** pad empty shots to hit the target sum.
- Sum of segment durations across beats should land inside the soft band when possible; story completeness beats exact seconds.`

const ShotSpecSystem = `# Role

Derive executable short-drama shot specs (5-Aspect + prompts) for each planned segment — concrete staging for T2I/I2V, not mood fluff.

## Output (critical)

` + jsonOutputCritical + `

` + jsonSchemaInstruction + `

Schema:

` + "```json\n" + `{
  "shots": [
    {
      "segment_index": 0,
      "five_aspect": {
        "subject": "<type + full names verbatim>",
        "subject_motion": "<actions in time order within this clip only>",
        "scene": "<setting + time + visible props>",
        "spatial_framing": "<FG/MG/BG + shot size>",
        "camera": "<static OR movement primitives>"
      },
      "shot_language": {
        "shot_size": "<景别>",
        "camera_movement": "static|dolly_in|pan_left",
        "lighting_key": "natural",
        "depth_of_field": "shallow|medium|deep",
        "color_temperature": "cool|neutral|warm|mixed"
      },
      "shot_intent": "<why this frame exists>",
      "video_prompt": "<timed action + camera + who speaks/reacts; match duration_sec>",
      "start_visual": "<t=0 still: shot size + pose/blocking + visible environment>",
      "anchor_visual": "<MCU/CU still; face_anchor ONLY>"
    }
  ]
}` + "\n```" + `

## Rules

- One shots[] row per segment_index listed.
- Honor start_visibility, role, characters_on_screen from the user message.
- Non face_anchor: anchor_visual MUST be empty string.
- camera_movement=static → video_prompt must NOT contain pan/dolly/zoom.
- **start_visual** (T2I source): one still at t=0 — 景别 + subject pose/blocking + visible environment props. No "cinematic / atmospheric / film grain" filler.
- **video_prompt** (I2V source): in-clip timed action + camera grammar + who speaks / who reacts if dialogue exists; capacity must fit duration_sec. Motion only — do **NOT** paste style_anchor, wardrobe essays, or mood boilerplate.
- **five_aspect / shot_language**: describe only what is visible in **this** segment; no cross-shot spoilers or unused cast.`

const ChunkSystem = `# Role

Extract beats from a script excerpt as JSON.

` + jsonOutputCritical + `

Return {"beats":[{"title":"","shot_size":"","location":"","narrative":""}]}`
