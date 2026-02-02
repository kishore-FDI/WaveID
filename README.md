

https://github.com/user-attachments/assets/8b2fd3a9-afd7-4cfb-816f-bff2cbaba9c5

That was a small demo on how the project works :)

### So Here is how it works

The audio is stored in stereo format. This format is what also us to make the song imerssive (If you have ever listened to 8D audio or in theaters) you would know. We have to convert this to mono so we can process all the audio data together.

`ffmpeg -i input.mp3 -ac 1 output_mono.mp3`
