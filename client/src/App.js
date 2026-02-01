import React, { useEffect, useState, useRef } from "react";
import io from "socket.io-client";
import Listen from "./components/Listen";
import CarouselSliders from "./components/CarouselSliders";
import { ToastContainer, toast, Slide } from "react-toastify";
import "react-toastify/dist/ReactToastify.css";
import { MediaRecorder, register } from "extendable-media-recorder";
import { connect } from "extendable-media-recorder-wav-encoder";
import { FFmpeg } from "@ffmpeg/ffmpeg";
import { fetchFile } from "@ffmpeg/util";
import AnimatedNumber from "./components/AnimatedNumber";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "./components/ui/card";
import { Button } from "./components/ui/button";
import { cn } from "./lib/utils";

const server = process.env.REACT_APP_BACKEND_URL || "http://localhost:5500";
const recordStereo = process.env.REACT_APP_RECORD_STEREO === "true" || false;

const socket = io("http://localhost:5000", {
  transports: ["polling", "websocket"],
});

const emitWithLog = (event, payload) => {
  console.log(`[socket:out] ${event}`, payload);
  socket.emit(event, payload);
};

function App() {
  let ffmpegLoaded = false;
  const ffmpeg = new FFmpeg();
  const uploadRecording = true;

  const [stream, setStream] = useState();
  const [matches, setMatches] = useState([]);
  const [totalSongs, setTotalSongs] = useState(10);
  const [isListening, setIsListening] = useState(false);
  const [genFingerprint, setGenFingerprint] = useState(null);
  const [registeredMediaEncoder, setRegisteredMediaEncoder] = useState(false);

  const streamRef = useRef(stream);
  const sendRecordingRef = useRef(true);

  useEffect(() => {
    streamRef.current = stream;
  }, [stream]);

  /* =======================
     SOCKET LIFECYCLE
  ======================= */
  useEffect(() => {
    socket.on("connect", () => {
      console.log("[socket] connected", socket.id);
      emitWithLog("totalSongs", "");
    });

    socket.on("matches", (payload) => {
      console.log("[socket] matches raw", payload);
      const parsed = JSON.parse(payload);
      if (parsed) setMatches(parsed.slice(0, 5));
      else toast("No song found.");
      cleanUp();
    });

    socket.on("downloadStatus", (payload) => {
      const msg = JSON.parse(payload);
      if (["info", "success", "error"].includes(msg.type))
        toast[msg.type](msg.message);
      else toast(msg.message);
    });

    socket.on("totalSongs", (songsCount) => {
      setTotalSongs(songsCount);
    });

    const id = setInterval(() => emitWithLog("totalSongs", ""), 8000);
    return () => clearInterval(id);
  }, []);

  /* =======================
     WASM LOAD
  ======================= */
  useEffect(() => {
    (async () => {
      try {
        const go = new window.Go();
        const result = await WebAssembly.instantiateStreaming(
          fetch("/fingerprint.wasm"),
          go.importObject
        );
        go.run(result.instance);
        if (typeof window.generateFingerprint === "function") {
          console.log("[wasm] fingerprint ready");
          setGenFingerprint(() => window.generateFingerprint);
        }
      } catch (e) {
        console.error("[wasm] load error", e);
      }
    })();
  }, []);

  /* =======================
     RECORD
  ======================= */
  async function record() {
    try {
      if (!genFingerprint) {
        console.error("[record] wasm not ready");
        return;
      }

      if (!ffmpegLoaded) {
        await ffmpeg.load();
        ffmpegLoaded = true;
      }

      if (!registeredMediaEncoder) {
        await register(await connect());
        setRegisteredMediaEncoder(true);
      }

      // Always use mic
      const mediaDevice = navigator.mediaDevices.getUserMedia.bind(navigator.mediaDevices);

      const stream = await mediaDevice({
        audio: {
          autoGainControl: false,
          channelCount: 1,
          echoCancellation: false,
          noiseSuppression: false,
          sampleSize: 16,
        },
      });

      const audioTracks = stream.getAudioTracks();
      const audioStream = new MediaStream(audioTracks);
      setStream(audioStream);

      audioTracks[0].onended = stopListening;

      const mediaRecorder = new MediaRecorder(audioStream, {
        mimeType: "audio/wav",
      });

      const chunks = [];
      mediaRecorder.ondataavailable = (e) => chunks.push(e.data);

      mediaRecorder.start();
      console.log("[record] started");
      setIsListening(true);
      sendRecordingRef.current = true;

      // Auto stop after 20s
      setTimeout(() => {
        if (mediaRecorder.state !== "inactive") {
          mediaRecorder.stop()
        }
      }, 20000);

      mediaRecorder.addEventListener("stop", async () => {
        console.log("[record] stopped");
        const blob = new Blob(chunks, { type: "audio/wav" });
        cleanUp(); // Clean up stream immediately after stop

        await ffmpeg.writeFile("input.wav", await fetchFile(blob));

        await ffmpeg.exec([
          "-i", "input.wav",
          "-c", "pcm_s16le",
          "-ar", "44100",
          "-ac", recordStereo ? "2" : "1",
          "-f", "wav", "out.wav",
        ]);

        const data = await ffmpeg.readFile("out.wav");
        const audioBlob = new Blob([data.buffer], { type: "audio/wav" });

        const reader = new FileReader();
        reader.readAsArrayBuffer(audioBlob);
        reader.onload = async (e) => {
          const ab = e.target.result;
          const ctx = new AudioContext();
          const decoded = await ctx.decodeAudioData(ab.slice(0));
          const audioArray = Array.from(decoded.getChannelData(0));

          const result = genFingerprint(
            audioArray,
            decoded.sampleRate,
            decoded.numberOfChannels
          );

          if (result.error !== 0) {
            console.error("[fingerprint] error", result);
            toast.error("Fingerprint error");
            return;
          }

          const fp = result.data.reduce((d, i) => {
            d[i.address] = i.anchorTime;
            return d;
          }, {});

          if (sendRecordingRef.current)
            emitWithLog("newFingerprint", JSON.stringify({ fingerprint: fp }));

          if (uploadRecording) {
            const bytes = new Uint8Array(ab);
            let raw = "";
            for (let i = 0; i < bytes.length; i++)
              raw += String.fromCharCode(bytes[i]);

            const view = new DataView(ab);
            const recordData = {
              audio: btoa(raw),
              channels: view.getUint16(22, true),
              sampleRate: view.getUint16(24, true),
              sampleSize: view.getUint16(34, true),
              duration: decoded.duration,
            };

            emitWithLog("newRecording", JSON.stringify(recordData));
          }
        };
      });
    } catch (e) {
      console.error("[record] fatal", e);
      cleanUp();
    }
  }

  function cleanUp() {
    if (streamRef.current)
      streamRef.current.getTracks().forEach((t) => t.stop());
    setStream(null);
    setIsListening(false);
  }

  function stopListening() {
    sendRecordingRef.current = false;
    cleanUp();
  }

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col items-center justify-center p-4">
      <div className="max-w-2xl w-full flex flex-col items-center space-y-8">

        {/* Header Section */}
        <div className="text-center space-y-2">
          <h1 className="text-4xl font-extrabold tracking-tight lg:text-5xl">
            Wave ID
          </h1>
          <p className="text-muted-foreground flex items-center justify-center gap-2">
            <span className="font-semibold text-primary">
              <AnimatedNumber includeComma animateToNumber={totalSongs} />
            </span>
            Songs Indexed
          </p>
        </div>

        {/* Main Action Card */}
        <Card className="w-full border-border/50 bg-card/50 backdrop-blur-sm">
          <CardHeader className="text-center">
            <CardTitle>Identify Music</CardTitle>
            <CardDescription>
              Tap the button to start listening specifically for music around you.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col items-center justify-center py-6">
            <Listen
              stopListening={stopListening}
              startListening={record}
              isListening={isListening}
            />
          </CardContent>
        </Card>

        {/* Results Section */}
        <div className="w-full">
          <CarouselSliders matches={matches} />
        </div>

      </div>

      <ToastContainer
        position="bottom-right"
        autoClose={5000}
        hideProgressBar
        newestOnTop
        closeOnClick
        rtl={false}
        pauseOnFocusLoss
        draggable
        pauseOnHover
        theme="dark"
        transition={Slide}
      />
    </div>
  );
}

export default App;
