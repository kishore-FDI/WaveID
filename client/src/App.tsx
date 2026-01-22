import { useState, useRef, useEffect } from 'react'
import './App.css'

type StatusType = 'idle' | 'requesting' | 'recording' | 'uploading' | 'success' | 'error'

interface Match {
  songId: number
  songTitle: string
  songArtist: string
  youtubeId: string
  timestamp: number
  score: number
}

interface UploadResponse {
  success: boolean
  filename: string
  message: string
  matches?: Match[]
  matchCount?: number
  searchDuration?: string
  topMatch?: {
    songTitle: string
    songArtist: string
    score: number
  }
  findError?: string
}

function App() {
  const [status, setStatus] = useState<StatusType>('idle')
  const [countdown, setCountdown] = useState(10)
  const [message, setMessage] = useState('')
  const [progress, setProgress] = useState(0)
  const [matches, setMatches] = useState<Match[]>([])
  const [topMatch, setTopMatch] = useState<Match | null>(null)
  const [searchDuration, setSearchDuration] = useState<string>('')
  
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const audioChunksRef = useRef<Blob[]>([])
  const countdownIntervalRef = useRef<number | null>(null)
  const progressIntervalRef = useRef<number | null>(null)

  const cleanup = () => {
    // Clear intervals
    if (countdownIntervalRef.current) {
      clearInterval(countdownIntervalRef.current)
      countdownIntervalRef.current = null
    }
    if (progressIntervalRef.current) {
      clearInterval(progressIntervalRef.current)
      progressIntervalRef.current = null
    }
    
    // Stop media recorder
    if (mediaRecorderRef.current && mediaRecorderRef.current.state !== 'inactive') {
      try {
        mediaRecorderRef.current.stop()
      } catch (e) {
        console.error('Error stopping recorder:', e)
      }
    }
    
    // Stop all tracks
    if (streamRef.current) {
      streamRef.current.getTracks().forEach(track => track.stop())
      streamRef.current = null
    }
    
    mediaRecorderRef.current = null
    audioChunksRef.current = []
  }

  const startRecording = async () => {
    try {
      setStatus('requesting')
      setMessage('Requesting microphone access...')
      setProgress(0)
      
      // Request microphone access
      const stream = await navigator.mediaDevices.getUserMedia({ 
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true
        } 
      })
      
      streamRef.current = stream

      // Check for supported MIME types
      let mimeType = 'audio/webm'
      if (MediaRecorder.isTypeSupported('audio/webm;codecs=opus')) {
        mimeType = 'audio/webm;codecs=opus'
      } else if (MediaRecorder.isTypeSupported('audio/webm')) {
        mimeType = 'audio/webm'
      } else if (MediaRecorder.isTypeSupported('audio/mp4')) {
        mimeType = 'audio/mp4'
      }

      const mediaRecorder = new MediaRecorder(stream, {
        mimeType: mimeType
      })
      
      mediaRecorderRef.current = mediaRecorder
      audioChunksRef.current = []

      mediaRecorder.ondataavailable = (event) => {
        if (event.data && event.data.size > 0) {
          audioChunksRef.current.push(event.data)
        }
      }

      mediaRecorder.onstop = async () => {
        if (audioChunksRef.current.length > 0) {
          const audioBlob = new Blob(audioChunksRef.current, { type: mimeType })
          await uploadAudio(audioBlob)
        } else {
          setStatus('error')
          setMessage('No audio data recorded. Please try again.')
        }
      }

      mediaRecorder.onerror = (event) => {
        console.error('MediaRecorder error:', event)
        setStatus('error')
        setMessage('Recording error occurred. Please try again.')
        cleanup()
      }

      // Start recording
      setStatus('recording')
      setMessage('Recording audio...')
      setCountdown(10)
      setProgress(0)
      
      mediaRecorder.start(100) // Collect data every 100ms

      // Start countdown
      let secondsLeft = 10
      countdownIntervalRef.current = window.setInterval(() => {
        secondsLeft--
        setCountdown(secondsLeft)
        setProgress(((10 - secondsLeft) / 10) * 100)
        
        if (secondsLeft <= 0) {
          if (countdownIntervalRef.current) {
            clearInterval(countdownIntervalRef.current)
            countdownIntervalRef.current = null
          }
          stopRecording()
        }
      }, 1000)

    } catch (error) {
      console.error('Error accessing microphone:', error)
      setStatus('error')
      if (error instanceof Error) {
        if (error.name === 'NotAllowedError' || error.name === 'PermissionDeniedError') {
          setMessage('Microphone access denied. Please allow microphone access and try again.')
        } else if (error.name === 'NotFoundError' || error.name === 'DevicesNotFoundError') {
          setMessage('No microphone found. Please connect a microphone and try again.')
        } else {
          setMessage(`Error: ${error.message}`)
        }
      } else {
        setMessage('Could not access microphone. Please check your browser permissions.')
      }
      cleanup()
    }
  }

  const stopRecording = () => {
    if (mediaRecorderRef.current && mediaRecorderRef.current.state === 'recording') {
      mediaRecorderRef.current.stop()
      setStatus('uploading')
      setMessage('Processing recording...')
      
      if (countdownIntervalRef.current) {
        clearInterval(countdownIntervalRef.current)
        countdownIntervalRef.current = null
      }
    }
  }

  const uploadAudio = async (audioBlob: Blob) => {
    try {
      setStatus('uploading')
      setMessage('Uploading audio to server...')
      setProgress(50)
      
      const formData = new FormData()
      formData.append('audio', audioBlob, `recording_${Date.now()}.webm`)

      const response = await fetch('http://localhost:8080/api/upload-audio', {
        method: 'POST',
        body: formData,
      })

      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`Upload failed: ${response.status} ${errorText}`)
      }

      const result: UploadResponse = await response.json()
      setStatus('success')
      setMessage(`Success! Audio saved as ${result.filename}`)
      setProgress(100)
      
      // Store match results
      if (result.matches && result.matches.length > 0) {
        setMatches(result.matches)
        setTopMatch(result.matches[0])
      } else {
        setMatches([])
        setTopMatch(null)
      }
      
      if (result.searchDuration) {
        setSearchDuration(result.searchDuration)
      }
      
      // Don't auto-reset if we have matches to show
      if (!result.matches || result.matches.length === 0) {
        setTimeout(() => {
          resetState()
        }, 3000)
      }
      
    } catch (error) {
      console.error('Error uploading audio:', error)
      setStatus('error')
      if (error instanceof Error) {
        if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
          setMessage('Could not connect to server. Make sure the server is running on http://localhost:8080')
        } else {
          setMessage(`Upload failed: ${error.message}`)
        }
      } else {
        setMessage('Upload failed. Please try again.')
      }
      setProgress(0)
    } finally {
      cleanup()
    }
  }

  const resetState = () => {
    setStatus('idle')
    setMessage('')
    setCountdown(10)
    setProgress(0)
    setMatches([])
    setTopMatch(null)
    setSearchDuration('')
    cleanup()
  }

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      cleanup()
    }
  }, [])

  const canRecord = status === 'idle' || status === 'error' || status === 'success'
  const isRecording = status === 'recording'
  const isProcessing = status === 'requesting' || status === 'uploading'

  return (
    <div className="app">
      <div className="container">
        <header>
          <h1>🎵 Audio Recorder</h1>
          <p className="subtitle">Record 10 seconds of audio and identify the song</p>
        </header>

        <div className="recorder-card">
          <div className="recorder-content">
            {status === 'idle' && (
              <div className="idle-state">
                <div className="mic-icon">🎤</div>
                <p>Click the button below to start recording</p>
              </div>
            )}

            {status === 'requesting' && (
              <div className="requesting-state">
                <div className="spinner"></div>
                <p>Requesting microphone access...</p>
              </div>
            )}

            {status === 'recording' && (
              <div className="recording-state">
                <div className="recording-visual">
                  <div className="recording-circle">
                    <div className="pulse-ring"></div>
                    <div className="pulse-ring"></div>
                    <div className="pulse-ring"></div>
                  </div>
                </div>
                <div className="countdown-display">
                  <span className="countdown-number">{countdown}</span>
                  <span className="countdown-label">seconds remaining</span>
                </div>
                <div className="progress-bar-container">
                  <div className="progress-bar" style={{ width: `${progress}%` }}></div>
                </div>
              </div>
            )}

            {status === 'uploading' && (
              <div className="uploading-state">
                <div className="spinner"></div>
                <p>Uploading audio...</p>
                <div className="progress-bar-container">
                  <div className="progress-bar" style={{ width: `${progress}%` }}></div>
                </div>
              </div>
            )}

            {status === 'success' && (
              <div className="success-state">
                <div className="success-icon">✓</div>
                <p className="success-message">Audio saved successfully!</p>
                {topMatch && (
                  <div className="top-match">
                    <h3>🎵 Match Found!</h3>
                    <div className="match-card">
                      <div className="match-title">{topMatch.songTitle}</div>
                      <div className="match-artist">by {topMatch.songArtist}</div>
                      <div className="match-score">Score: {topMatch.score.toFixed(2)}</div>
                    </div>
                  </div>
                )}
                {matches.length === 0 && !topMatch && (
                  <div className="no-matches">
                    <p>No matches found in the database.</p>
                  </div>
                )}
              </div>
            )}

            {status === 'error' && (
              <div className="error-state">
                <div className="error-icon">⚠</div>
                <p className="error-message">{message || 'An error occurred'}</p>
              </div>
            )}

            {message && status !== 'recording' && status !== 'uploading' && (
              <div className={`message ${status === 'error' ? 'error' : status === 'success' ? 'success' : ''}`}>
                {message}
              </div>
            )}
          </div>

          <div className="controls">
            {canRecord && (
              <button 
                onClick={startRecording} 
                className="btn btn-primary"
                disabled={isProcessing}
              >
                <span>Start Recording</span>
                <span className="btn-subtitle">10 seconds</span>
              </button>
            )}

            {isRecording && (
              <button 
                onClick={stopRecording} 
                className="btn btn-stop"
              >
                Stop Recording
              </button>
            )}

            {(status === 'error' || status === 'success') && (
              <button 
                onClick={resetState} 
                className="btn btn-secondary"
              >
                Try Again
              </button>
            )}
          </div>
        </div>

        {matches.length > 0 && status === 'success' && (
          <div className="matches-section">
            <h2>All Matches ({matches.length})</h2>
            {searchDuration && (
              <p className="search-info">Search took: {searchDuration}</p>
            )}
            <div className="matches-list">
              {matches.slice(0, 10).map((match, index) => (
                <div key={match.songId} className="match-item">
                  <div className="match-rank">#{index + 1}</div>
                  <div className="match-details">
                    <div className="match-title-small">{match.songTitle}</div>
                    <div className="match-artist-small">by {match.songArtist}</div>
                  </div>
                  <div className="match-score-small">{match.score.toFixed(2)}</div>
                </div>
              ))}
            </div>
          </div>
        )}

        <footer>
          <p className="info-text">
            Make sure the server is running on <code>http://localhost:8080</code>
          </p>
        </footer>
      </div>
    </div>
  )
}

export default App
