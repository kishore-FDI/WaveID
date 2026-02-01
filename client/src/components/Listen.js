import React, { useEffect, useState } from "react";
import { Button } from "../components/ui/button";
import { cn } from "../lib/utils";

const Listen = ({ startListening, stopListening, isListening }) => {
  const [pulse, setPulse] = useState(false);

  useEffect(() => {
    if (isListening) {
      setPulse(true);
    } else {
      setPulse(false);
    }
  }, [isListening]);

  const toggleListen = () => {
    if (isListening) {
      stopListening();
    } else {
      startListening();
    }
  };

  return (
    <div className="relative flex items-center justify-center">
      {/* Ripple Effect */}
      {isListening && (
        <>
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary/20 opacity-75 duration-1000"></span>
          <span className="animate-ping absolute inline-flex h-3/4 w-3/4 rounded-full bg-primary/30 opacity-75 delay-150 duration-1000"></span>
        </>
      )}

      <Button
        size="lg"
        onClick={toggleListen}
        className={cn(
          "h-32 w-32 rounded-full text-lg font-bold shadow-2xl transition-all duration-300 z-10",
          isListening
            ? "bg-destructive hover:bg-destructive/90 animate-pulse"
            : "bg-primary hover:bg-primary/90 hover:scale-105"
        )}
      >
        {isListening ? "Listening" : "Listen"}
      </Button>
    </div>
  );
};

export default Listen;
