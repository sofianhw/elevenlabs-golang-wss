# ElevenLabs Golang WebSocket TTS Client

This project is a command-line tool written in Go for streaming text-to-speech (TTS) synthesis using the ElevenLabs API over WebSockets. It allows you to input text, send it to the ElevenLabs TTS service, and play back the resulting audio on your Mac.

## Features

- Interactive CLI for entering text lines
- Streams audio using ElevenLabs' WebSocket API
- Supports voice, model, stability, similarity, and speed parameters
- Outputs audio as MP3 and plays it automatically (`afplay` on macOS)
- Keep-alive and inactivity timeout handling

## Requirements

- Go 1.24+
- macOS (uses `afplay` for playback)
- [ElevenLabs API key](https://elevenlabs.io/)

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/yourusername/elevenlabs-golang-wss.git
    cd elevenlabs-golang-wss
    ```

2. Install dependencies:
    ```sh
    go mod tidy
    ```

3. Build the binary:
    ```sh
    go build -o elevenlabs-wss
    ```

## Usage

```sh
./elevenlabs-wss -api-key YOUR_API_KEY -voice-id VOICE_ID [options]
```

### Options

- `-api-key` (required): Your ElevenLabs API key
- `-voice-id` (required): Voice ID to use
- `-model-id`: TTS model to use (default: `eleven_flash_v2_5`)
- `-stability`: Voice stability (0.0–1.0, default: 0.5)
- `-similarity`: Similarity boost (0.0–1.0, default: 0.8)
- `-speed`: Playback speed multiplier (default: 1.0)
- `-timeout`: WebSocket inactivity timeout in seconds (default: 120, max: 180)

### Example

```sh
./elevenlabs-wss -api-key YOUR_API_KEY -voice-id YOUR_VOICE_ID
```

You will be prompted to enter lines of text. Type `done` to synthesize and play, or `exit` to quit.

## File Structure

- [`main.go`](main.go): Main application source code
- [`go.mod`](go.mod): Go module definition
- [`.gitignore`](.gitignore): Git ignore rules