import json
import subprocess, random, requests, tempfile, os

IN = "4.wav"
URL = "http://localhost:8080/api/upload-audio"
CLIP = 10

def dur(p):
    r = subprocess.run(
        [
            "ffprobe",
            "-v", "error",
            "-select_streams", "a:0",
            "-show_entries", "stream=duration",
            "-of", "default=noprint_wrappers=1:nokey=1",
            p,
        ],
        capture_output=True,
        text=True,
    )
    return float(r.stdout.strip())

d = dur(IN)
s = 0 if d <= CLIP else random.uniform(0, d - CLIP)

out = tempfile.NamedTemporaryFile(suffix=".mp3", delete=False).name
subprocess.run(
    [
        "ffmpeg",
        "-y",
        "-ss", str(s),
        "-t", str(CLIP),
        "-i", IN,
        "-vn",
        "-acodec", "libmp3lame",
        "-ab", "192k",
        out,
    ],
    stdout=subprocess.DEVNULL,
    stderr=subprocess.DEVNULL,
)

with open(out, "rb") as f:
    r = requests.post(
        URL,
        files={
            "audio": ("clip.mp3", f, "audio/mpeg")
        },
    )
    print(r.status_code)
    op = json.loads(r.text)
    for i in op["matches"]:
        for j in i:
            if j=="score" or j=="songTitle":
                print(f"{j}: {i[j]}")
        # if i=="score" or i=="songTitle":
        #     print(f"{i}: {op['matches'][i]}")

os.remove(out)
