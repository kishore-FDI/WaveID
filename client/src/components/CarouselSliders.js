import React, { useEffect, useRef, useState } from "react";
import YouTube from "react-youtube";
import { Card, CardContent } from "./ui/card";
import { cn } from "../lib/utils";

const CarouselSliders = (props) => {
  const [activeVideoID, setActiveVideoID] = useState(null);
  const players = useRef({});

  useEffect(() => {
    if (props.matches.length > 0) {
      const validMatches = props.matches.filter((match) => match.YouTubeID);
      if (validMatches.length > 0) {
        const firstVideoID = validMatches[0].YouTubeID;
        setActiveVideoID(firstVideoID);
      }
    }
  }, [props.matches]);

  const onReady = (event, videoId) => {
    players.current[videoId] = event.target;
  };

  const onPlay = (event) => {
    const videoId = event.target.getVideoData().video_id;
    setActiveVideoID(videoId);

    Object.values(players.current).forEach((player) => {
      if (player && typeof player.getVideoData === 'function') {
        const otherVideoId = player.getVideoData().video_id;
        if (otherVideoId !== videoId && player.getPlayerState() === 1) {
          player.pauseVideo();
        }
      }
    });
  };

  if (!props.matches.length) return null;

  const activeMatch = props.matches.find(m => m.YouTubeID === activeVideoID) || props.matches[0];
  const otherMatches = props.matches.filter(m => m.YouTubeID !== activeVideoID);

  return (
    <div className="space-y-6">

      {/* Active Video - Main Stage */}
      {activeMatch && activeMatch.YouTubeID && (
        <Card className="overflow-hidden border-primary/20 shadow-lg">
          <div className="aspect-video w-full bg-black">
            <YouTube
              videoId={activeMatch.YouTubeID}
              opts={{
                width: '100%',
                height: '100%',
                playerVars: {
                  start: (parseInt(activeMatch.Timestamp) / 1000) | 0,
                  rel: 0
                },
              }}
              className="w-full h-full"
              onReady={(event) => onReady(event, activeMatch.YouTubeID)}
              onPlay={onPlay}
            />
          </div>
          <CardContent className="p-4">
            <h3 className="font-semibold text-lg">{activeMatch.TrackName || "Unknown Track"}</h3>
            <p className="text-sm text-muted-foreground">{activeMatch.Artist || "Unknown Artist"}</p>
          </CardContent>
        </Card>
      )}

      {/* Playlist / Queue */}
      {otherMatches.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {otherMatches.map((match) => (
            <Card
              key={match.YouTubeID}
              className="cursor-pointer hover:bg-accent/50 transition-colors"
              onClick={() => setActiveVideoID(match.YouTubeID)}
            >
              <CardContent className="p-3 flex items-center space-x-4">
                <div className="h-16 w-24 bg-muted rounded overflow-hidden flex-shrink-0 relative">
                  <img
                    src={`https://img.youtube.com/vi/${match.YouTubeID}/default.jpg`}
                    alt="thumbnail"
                    className="object-cover w-full h-full opacity-80"
                  />
                </div>
                <div className="overflow-hidden">
                  <p className="font-medium truncate">{match.TrackName || "Unknown"}</p>
                  <p className="text-xs text-muted-foreground truncate">{match.Artist || "Unknown"}</p>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
};

export default CarouselSliders;
