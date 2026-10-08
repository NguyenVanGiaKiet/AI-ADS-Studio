# Backend: local Vietnamese TTS

Speech generation runs locally with Piper and bundled Vietnamese models: VAI 1000, 25Hours, and Piper v3 with five speakers (Ngọc Lan, Minh Anh, Quang Huy, Thu Hà, and Yến Nhi). The Piper v3 model and config are from [CakeByVPBank on Hugging Face](https://huggingface.co/CakeByVPBank/piper-pgl-v4-vi_VN-version39_epoch39) and are listed as MIT licensed by the model publisher. It does not call an external TTS service or require paid TTS credits. Groq is still used to generate the Vietnamese advertisement script, so configure `GROQ_API_KEY` in `backend/.env` for script generation. VAI 1000 is the default voice.

Video remix modes are `standard` (normal cuts), `exclude_faces` (OpenCV samples the video and removes face-positive moments, including a short margin around each detection), and `product_zoom` (zooms into the lower-center area to reduce visible faces; this heuristic does not identify the product itself). Face detection runs locally with OpenCV's frontal-face Haar cascade and may miss profiles, occluded faces, or people whose faces are not visible. Install its runtime with the other Python dependencies below.

Optional karaoke subtitles are burned into the remixed video when `followSubtitles` is enabled. `subtitlePosition` accepts `bottom` (above the shopping-cart area) or `top` (below the search area); `subtitleStyle` accepts `white_yellow` (white upcoming words, yellow active word) or `white_gray` (gray upcoming words, white active word). Word timestamps are estimated from the generated script and target duration.

To reduce repeated remixes, the backend generates up to five variants by shuffling clip order and choosing different safe trim points. Each candidate is checked against prior output videos using SHA-256 plus sampled, time-aligned frame hashes. If candidates remain above the 60% visual-match threshold, it keeps the least-similar one and adds a warning to the task status. Absolute perceptual uniqueness cannot be guaranteed when the same source footage is reused; use additional source footage for stronger variety.

For each remix with voice replacement, the backend first measures the selected local Piper voice's natural syllables-per-second rate and caches that calibration for the process lifetime. Groq uses the measured rate, target video duration, and selected 0.7x-1.3x speed to write a script sized to fill the video naturally. FFmpeg then applies a final pitch-preserving `atempo` correction and pads/trims to the exact video duration; subtitles are timed over the same narration duration.

When a remix task requests multiple output videos with voice replacement enabled, Groq writes a separate script for each video using a different creative approach and the scripts already generated as context. Piper generates a separate narration for each script, and that same script is used for its video's subtitles. Exact duplicate scripts are retried up to three times; if Groq still repeats one, the task stops rather than silently producing repeated narration.

Script styles: `professional` is a clear, trustworthy ad read; `adam_drama` uses playful everyday drama and light dialogue; `adam_viral` uses fast viral pacing and occasional strong exclamations; `dan_da` uses plain, warm, rustic Vietnamese phrasing. All styles remain constrained to facts supplied in the product description.

Install Piper and the Python dependency used for local face detection:

```powershell
python -m pip install -r requirements.txt
```

Run the Go backend from this directory so its existing relative storage paths resolve as expected:

```powershell
go run .
```

The backend uses `python` by default. Set `PIPER_PYTHON` if Piper is installed under another interpreter, or `PIPER_MODEL_PATH` to select another local Piper model. FFmpeg is required for remixing videos and for adjusting the generated speech rate.

Remix jobs run in a backend worker and continue while navigating between app sections. The frontend keeps the active task ID locally and resumes status polling after a page reload. Completed task outputs can be downloaded together as a ZIP from `GET /api/remix/tasks/{id}/download`. Download the entire output library from `GET /api/videos/download`, or send `POST /api/videos/download` with `{"videoIds":["..."]}` to download only selected videos as a ZIP.
