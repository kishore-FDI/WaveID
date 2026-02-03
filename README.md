

https://github.com/user-attachments/assets/8b2fd3a9-afd7-4cfb-816f-bff2cbaba9c5

That was a small demo on how the project works :)

## So Here is how it works

The audio is stored in stereo format. This format is what also us to make the song imerssive (If you have ever listened to 8D audio or in theaters) you would know. We have to convert this to mono so we can process all the audio data together. We should also convert the song to WAV format for better processing.

`ffmpeg -i input.mp3 -ac 1 output_mono.wav`

### Pre-Processing Phase

We need to pre-process the audio to make the next phase effecient! We will start of by applying Low-Pass Filter to remove frequencies over 5Khz
<img width="317" height="252" alt="image" src="https://github.com/user-attachments/assets/1eb395ed-e064-4584-bef0-4565479ea964" />
The songs are generally sampled at 42k , 44k , etc . Processing these many samples would take alot of time, compute and space. We can uniformly downsample the audio to 11025 hz. This is also crutial because we need to sycn the sampling rate at the client. 
 
---

### Spectrogram

Now it’s time to convert the audio signal into something we can **see and analyze**.

A spectrogram represents audio in a **3-dimensional space**:

* **X-axis** → Time
* **Y-axis** → Frequency
* **Color / Intensity** → Amplitude (energy)


To generate this, we apply a **Short-Time Fourier Transform (STFT)**.

Instead of analyzing the entire song at once, we:

1. Split the audio into **small overlapping time windows**
2. Apply FFT on each window
3. Stack the frequency results over time

To get these windows we can use Hamming or Hanning Window methods.

This lets us observe **how frequencies evolve**, which is crucial for fingerprinting.

<img width="311" height="162" alt="image" src="https://github.com/user-attachments/assets/d9d7efa7-6a69-4fbc-a373-30fe76602c9c" />


---

### Peak Detection 

A raw spectrogram contains **too much data**.
Most of it is noise or unimportant background information.

So instead of using everything, we extract only the **most dominant frequency peaks**.

For each small region in the spectrogram:

* Find **local maxima**
* Ignore weak frequencies
* Keep only the strongest peaks

These peaks are:

* Stable across devices
* Resistant to noise
* Survive compression (MP3, AAC, etc.)


Think of these peaks as the **“constellation points”** of the song.

---

### Audio Fingerprinting

Now we turn these peaks into a **fingerprint**.

Instead of storing raw frequencies, we:

1. Pick an **anchor peak**
2. Pair it with nearby peaks in the future (5 peaks)
3. Create hashes using:

   * Anchor frequency
   * Target frequency
   * Time difference between them

This creates hashes like:

```
hash(f1, f2, Δt)
```

Each hash also stores:

* **Song ID**
* **Time offset**

This is extremely compact and fast to search.

<img width="500" alt="Fingerprint Hashing" src="https://github.com/user-attachments/assets/REPLACE_WITH_HASH_IMAGE" />

---

### Database Storage

All fingerprints from all songs are stored in a database like:

| Hash   | Song ID | Time Offset |
| ------ | ------- | ----------- |
| A1B2C3 | song_42 | 13.2s       |
| D4E5F6 | song_42 | 14.0s       |
| X9Y8Z7 | song_12 | 05.6s       |

This allows **O(1)** lookup during matching.

---

### Matching & Recognition 

When a user records a short audio clip:

1. The same pipeline is applied:

   * Mono conversion
   * Filtering
   * Spectrogram
   * Peak detection
   * Hash generation
2. Generated hashes are matched against the database
3. We then look for the matching hashes. We only count it as a valid hash if the time difference between the matches is less than 100ms
   
If many hashes point to the same song **with the same offset**,
**we’ve found the match**.
---

### Why This Works So Well

* Works with **very short clips (5–10 seconds)**
* Resistant to:

  * Noise
  * Compression
  * Phone microphone quality
* Extremely fast lookup
* Proven technique (used by Shazam itself)

---

If you want, next I can:

* Tighten the language even more (README polish)
* Add **GIF placement suggestions** (where animations help most)
* Write a **“Project Architecture”** or **“Future Improvements”** section
* Or adapt this README for **GitHub stars bait ⭐**



