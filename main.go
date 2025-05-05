package main

import (
    "bufio"
    "encoding/base64"
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "os"
    "os/exec"
    "time"

    "github.com/gorilla/websocket"
)

func main() {
    log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

    // CLI flags
    apiKey        := flag.String("api-key", "", "Your ElevenLabs API key (required)")
    voiceID       := flag.String("voice-id", "", "Voice ID to use")
    modelID       := flag.String("model-id", "eleven_flash_v2_5", "TTS model to use")
    stability     := flag.Float64("stability", 0.5, "Voice stability (0.0–1.0)")
    similarity    := flag.Float64("similarity", 0.8, "Similarity boost (0.0–1.0)")
    speed         := flag.Float64("speed", 1.0, "Playback speed multiplier")
    inactivitySec := flag.Int("timeout", 120, "WS inactivity timeout in seconds (max 180)")
    flag.Parse()

    if *apiKey == "" || *voiceID == "" {
        flag.Usage()
        os.Exit(1)
    }

    scanner := bufio.NewScanner(os.Stdin)
    fmt.Println(`Enter lines to synthesize. Type "done" to flush/play, "exit" to quit.`)

    for {
        // 1) (Re)open WS for this segment
        conn := dialWS(*apiKey, *voiceID, *modelID, *inactivitySec)
        first := true

        // keep-alive setup
        var ticker *time.Ticker
        var doneKA chan struct{}

        // 2) Collect & send lines until "done" or "exit"
        for {
            fmt.Print("> ")
            if !scanner.Scan() {
                log.Println("Input closed, exiting.")
                conn.Close()
                return
            }
            line := scanner.Text()

            switch line {
            case "":
                continue
            case "exit":
                log.Println("Exiting.")
                conn.Close()
                return
            case "done":
                // 3a) flush
                log.Println("→ sending flush")
                if err := conn.WriteJSON(map[string]interface{}{"text": "", "flush": true}); err != nil {
                    log.Fatalf("flush send error: %v", err)
                }
                // stop keep-alive
                if ticker != nil {
                    ticker.Stop()
                    close(doneKA)
                }
                break
            default:
                // 3b) on first real text, init + keep-alive
                if first {
                    ticker = time.NewTicker(15 * time.Second)
                    doneKA = make(chan struct{})
                    go keepAlive(conn, ticker, doneKA)

                    initMsg := map[string]interface{}{
                        "text": " ",
                        "voice_settings": map[string]float64{
                            "stability":        *stability,
                            "similarity_boost": *similarity,
                            "speed":            *speed,
                        },
                    }
                    log.Println("→ sending init")
                    if err := conn.WriteJSON(initMsg); err != nil {
                        log.Fatalf("init send error: %v", err)
                    }
                    first = false
                }

                // 3c) send this line
                log.Printf("→ sending text: %q", line)
                if err := conn.WriteJSON(map[string]string{"text": line}); err != nil {
                    log.Fatalf("text send error: %v", err)
                }
                continue
            }

            // break out of sending loop on "done"
            break
        }

        // 4) receive, write & play
        audioData := receiveAudio(conn)
        conn.Close()

        // write & play
        outFile := "output.mp3"
        log.Printf("writing %d bytes to %s", len(audioData), outFile)
        if err := ioutil.WriteFile(outFile, audioData, 0644); err != nil {
            log.Fatalf("write error: %v", err)
        }
        log.Println("playing via afplay")
        cmd := exec.Command("afplay", outFile)
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        if err := cmd.Run(); err != nil {
            log.Fatalf("playback error: %v", err)
        }
        log.Println("playback complete")
        // loop around for next segment
    }
}

func dialWS(apiKey, voiceID, modelID string, inactivitySec int) *websocket.Conn {
    url := fmt.Sprintf(
        "wss://api.elevenlabs.io/v1/text-to-speech/%s/stream-input?model_id=%s&inactivity_timeout=%d",
        voiceID, modelID, inactivitySec,
    )
    log.Printf("Dialing WebSocket: %s", url)
    conn, _, err := websocket.DefaultDialer.Dial(url, map[string][]string{
        "xi-api-key": {apiKey},
    })
    if err != nil {
        log.Fatalf("WebSocket dial error: %v", err)
    }
    log.Println("WebSocket connected")
    return conn
}

func keepAlive(conn *websocket.Conn, ticker *time.Ticker, done chan struct{}) {
    for {
        select {
        case <-ticker.C:
            log.Println("→ keep-alive ping")
            if err := conn.WriteJSON(map[string]string{"text": " "}); err != nil {
                log.Printf("keep-alive error: %v", err)
                return
            }
        case <-done:
            return
        }
    }
}

// receiveAudio reads from conn until it sees final or a 5s timeout
func receiveAudio(conn *websocket.Conn) []byte {
    var audioData []byte
    log.Println("← awaiting audio…")
    for {
        conn.SetReadDeadline(time.Now().Add(5 * time.Second))
        _, msg, err := conn.ReadMessage()
        if err != nil {
            if ne, ok := err.(net.Error); ok && ne.Timeout() {
                log.Println("← read timeout, end of stream")
                break
            }
            if ce, ok := err.(*websocket.CloseError); ok && ce.Code == websocket.CloseNormalClosure {
                log.Println("← normal close (1000), end of stream")
                break
            }
            log.Fatalf("read error: %v", err)
        }

        var resp map[string]interface{}
        if err := json.Unmarshal(msg, &resp); err != nil {
            log.Printf("skip malformed JSON: %v", err)
            continue
        }

        if enc, ok := resp["audio"].(string); ok && enc != "" {
            log.Printf("← got chunk (%d chars)", len(enc))
            chunk, err := base64.StdEncoding.DecodeString(enc)
            if err != nil {
                log.Fatalf("decode error: %v", err)
            }
            audioData = append(audioData, chunk...)
            log.Printf("    buffered %d bytes", len(audioData))
        }

        if fb, ok := resp["final"].(bool); ok && fb {
            log.Println("← final=true, end of stream")
            break
        }
        if _, hasObj := resp["final"]; hasObj && resp["final"] != false {
            log.Println("← final object, end of stream")
            break
        }
    }
    return audioData
}